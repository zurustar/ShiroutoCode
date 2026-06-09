// Package agent implements the autonomous plan→act→observe loop (FR-3). It is
// front-agnostic: progress is reported through the Frontend port (implemented
// by U5), tool execution always goes through the guardrail Dispatcher (U3), and
// the loop is guaranteed to terminate (completion / max steps / cancellation).
package agent

import (
	"context"
	"errors"
	"strings"

	"github.com/zurustar/shiroutocode/internal/llm"
	"github.com/zurustar/shiroutocode/internal/log"
	"github.com/zurustar/shiroutocode/internal/tools"
)

// Task is a single user instruction.
type Task struct {
	Prompt string
}

// Status is the terminal state of a run.
type Status int

const (
	Completed Status = iota
	StoppedMaxSteps
	Aborted
	Failed
)

func (s Status) String() string {
	switch s {
	case Completed:
		return "completed"
	case StoppedMaxSteps:
		return "stopped_max_steps"
	case Aborted:
		return "aborted"
	default:
		return "failed"
	}
}

// Result is the outcome of a run.
type Result struct {
	Status       Status
	Summary      string
	ChangedFiles []string
	Steps        int
}

// Frontend receives progress events (implemented by U5; default is no-op).
type Frontend interface {
	OnAssistantText(delta string)
	OnToolCall(name string, args map[string]any)
	OnToolResult(name, output string, err error)
	OnStep(current, max int)
}

// NoopFrontend ignores all events.
type NoopFrontend struct{}

func (NoopFrontend) OnAssistantText(string)             {}
func (NoopFrontend) OnToolCall(string, map[string]any)  {}
func (NoopFrontend) OnToolResult(string, string, error) {}
func (NoopFrontend) OnStep(int, int)                    {}

// Dispatcher executes a tool call subject to guardrail evaluation (U3).
type Dispatcher interface {
	Dispatch(ctx context.Context, call tools.ToolCall) (tools.ToolResult, error)
}

// DefaultSystemPrompt guides the model through the tool-using loop.
const DefaultSystemPrompt = `You are ShiroutoCode, an autonomous coding agent.
Work step by step: plan, call a tool, observe its result, then continue.
Use the provided tools to read and modify files, run commands, use git, or fetch web pages.
When the task is complete, reply with your final answer and DO NOT call any tool.`

// Runner executes the agent loop.
type Runner struct {
	llm      llm.LLMClient
	disp     Dispatcher
	reg      *tools.Registry
	fe       Frontend
	logger   log.Logger
	maxSteps int
	toolMode llm.ToolMode
	system   string
}

// Option configures a Runner.
type Option func(*Runner)

func WithFrontend(f Frontend) Option     { return func(r *Runner) { r.fe = f } }
func WithLogger(l log.Logger) Option     { return func(r *Runner) { r.logger = l } }
func WithMaxSteps(n int) Option          { return func(r *Runner) { r.maxSteps = n } }
func WithToolMode(m llm.ToolMode) Option { return func(r *Runner) { r.toolMode = m } }
func WithSystemPrompt(s string) Option   { return func(r *Runner) { r.system = s } }

// NewRunner builds a Runner with the given LLM client, dispatcher and registry.
func NewRunner(client llm.LLMClient, disp Dispatcher, reg *tools.Registry, opts ...Option) *Runner {
	r := &Runner{
		llm:      client,
		disp:     disp,
		reg:      reg,
		fe:       NoopFrontend{},
		logger:   log.New("error", log.FormatText, discard{}),
		maxSteps: 25,
		toolMode: llm.ToolModeAuto,
		system:   DefaultSystemPrompt,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Run executes the loop until completion, the step limit, or cancellation (F1).
func (r *Runner) Run(ctx context.Context, task Task) (Result, error) {
	if strings.TrimSpace(task.Prompt) == "" {
		return Result{Status: Failed}, errors.New("empty prompt")
	}

	msgs := []llm.Message{
		{Role: llm.RoleSystem, Content: r.system},
		{Role: llm.RoleUser, Content: task.Prompt},
	}
	var changed []string

	for step := 1; step <= r.maxSteps; step++ {
		if err := ctx.Err(); err != nil {
			return Result{Status: Aborted, ChangedFiles: changed, Steps: step - 1}, nil
		}

		req := llm.Request{
			Messages: msgs,
			Tools:    r.specs(),
			Stream:   true,
			ToolMode: r.toolMode,
		}
		stream, err := r.llm.Complete(ctx, req)
		if err != nil {
			return r.failOrAbort(ctx, err, changed, step), nil
		}
		res, cerr := llm.CollectStreaming(stream, r.fe.OnAssistantText)
		stream.Close()
		if cerr != nil {
			return r.failOrAbort(ctx, cerr, changed, step), nil
		}

		r.fe.OnStep(step, r.maxSteps)

		if len(res.ToolCalls) == 0 {
			return Result{Status: Completed, Summary: res.Text, ChangedFiles: changed, Steps: step}, nil
		}

		appendAssistant(&msgs, res, stream.Mode())
		for _, tc := range res.ToolCalls {
			r.fe.OnToolCall(tc.Name, tc.Args)
			out, derr := r.disp.Dispatch(ctx, tools.ToolCall{Name: tc.Name, Args: tc.Args})
			r.fe.OnToolResult(tc.Name, out.Output, derr)
			changed = append(changed, out.Changed...)
			appendObservation(&msgs, tc, out, derr, stream.Mode())
		}
	}

	return Result{Status: StoppedMaxSteps, ChangedFiles: changed, Steps: r.maxSteps}, nil
}

func (r *Runner) failOrAbort(ctx context.Context, err error, changed []string, step int) Result {
	if ctx.Err() != nil {
		return Result{Status: Aborted, ChangedFiles: changed, Steps: step - 1}
	}
	r.logger.Error("agent: llm error", "step", step)
	return Result{Status: Failed, ChangedFiles: changed, Steps: step - 1}
}

// discard drops log output (default sink).
type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

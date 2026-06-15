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
	Err          error // underlying cause when Status == Failed (e.g. *llm.LLMError)
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
const DefaultSystemPrompt = `You are ShiroutoCode, an autonomous coding agent operating inside the user's workspace.

Work in a loop: think briefly, call a tool, observe its result, then continue. Tools:
- read_file: read a workspace file.
- write_file: create, overwrite, edit, or delete a workspace file. This is how you SAVE code.
- run_command: run a shell command in the workspace.
- git: run git operations.
- web_fetch: fetch a web page.

Rules:
- When the user asks you to create, generate, or modify code or files, you MUST call write_file to write the content to the workspace. Code shown only as text in your reply is NOT saved to disk — always persist it with write_file.
- Prefer taking concrete actions (tool calls) over merely describing them.
- After editing, you may verify with read_file or run_command.
- The conversation continues across turns: remember earlier instructions, files you created, and results when handling follow-up requests.
- To inspect a file's current content, call read_file; do not rely on memory.
- Always end your turn with a short textual reply to the user — answer their question or summarize what you did. Never end a turn silently with no text.
- Only when the task is fully complete, give that final textual reply and DO NOT call any tool.`

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

	// history is the running conversation. It persists across Run calls so a
	// REPL session has multi-turn memory (follow-up prompts see prior turns).
	history []llm.Message
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

// Reset clears the conversation history, starting a fresh session while
// keeping the same client, tools and configuration.
func (r *Runner) Reset() { r.history = nil }

// Run executes the loop until completion, the step limit, or cancellation (F1).
func (r *Runner) Run(ctx context.Context, task Task) (Result, error) {
	if strings.TrimSpace(task.Prompt) == "" {
		return Result{Status: Failed}, errors.New("empty prompt")
	}

	// Seed the system message once, then carry the conversation across turns.
	if len(r.history) == 0 {
		r.history = append(r.history, llm.Message{Role: llm.RoleSystem, Content: r.system})
	}
	// mark is the rollback point (after system, before this turn). On an LLM
	// failure we trim back to here so a half-formed turn — e.g. a malformed
	// tool call that the server then 500s on — does not poison later turns.
	mark := len(r.history)
	r.history = append(r.history, llm.Message{Role: llm.RoleUser, Content: task.Prompt})
	var changed []string

	for step := 1; step <= r.maxSteps; step++ {
		if err := ctx.Err(); err != nil {
			return Result{Status: Aborted, ChangedFiles: changed, Steps: step - 1}, nil
		}

		req := llm.Request{
			Messages: r.history,
			Tools:    r.specs(),
			Stream:   true,
			ToolMode: r.toolMode,
		}
		stream, err := r.llm.Complete(ctx, req)
		if err != nil {
			return r.finishErr(ctx, err, changed, step, mark), nil
		}
		res, cerr := llm.CollectStreaming(stream, r.fe.OnAssistantText)
		stream.Close()
		if cerr != nil {
			return r.finishErr(ctx, cerr, changed, step, mark), nil
		}

		r.fe.OnStep(step, r.maxSteps)

		if len(res.ToolCalls) == 0 {
			// Record the final answer so follow-up turns remember it.
			r.history = append(r.history, llm.Message{Role: llm.RoleAssistant, Content: res.Text})
			return Result{Status: Completed, Summary: res.Text, ChangedFiles: changed, Steps: step}, nil
		}

		appendAssistant(&r.history, res, stream.Mode())
		for _, tc := range res.ToolCalls {
			r.fe.OnToolCall(tc.Name, tc.Args)
			out, derr := r.disp.Dispatch(ctx, tools.ToolCall{Name: tc.Name, Args: tc.Args})
			r.fe.OnToolResult(tc.Name, out.Output, derr)
			changed = append(changed, out.Changed...)
			appendObservation(&r.history, tc, out, derr, stream.Mode())
		}
	}

	// Step limit reached without completing. Summarize progress for handoff and
	// compact the conversation so a follow-up can continue from the summary
	// (and the bloated history does not overflow the model's context).
	summary := r.summarizeForHandoff(ctx)
	if summary != "" {
		r.history = []llm.Message{
			{Role: llm.RoleSystem, Content: r.system},
			{Role: llm.RoleAssistant, Content: handoffPrefix + summary},
		}
	}
	return Result{Status: StoppedMaxSteps, Summary: summary, ChangedFiles: changed, Steps: r.maxSteps}, nil
}

// handoffPrefix labels a compacted progress summary in the conversation.
const handoffPrefix = "（前回の進捗要約 / handoff）\n"

// stepLimitSummaryPrompt asks the model to produce a continuation handoff.
const stepLimitSummaryPrompt = `ステップ上限に達したため一旦停止します。次のセッションで作業を継続できるよう、日本語で簡潔に要約してください:
1) これまでに完了したこと（作成・編集したファイルを含む）
2) 現在の状態
3) 残っている作業と、具体的な次のステップ
ツールは呼び出さず、要約テキストだけを返してください。`

// summarizeForHandoff makes one tool-free LLM call to summarize progress. It is
// best-effort: on cancellation or any error it returns "" and the caller keeps
// the full history rather than compacting.
func (r *Runner) summarizeForHandoff(ctx context.Context) string {
	if ctx.Err() != nil {
		return ""
	}
	msgs := append(append([]llm.Message{}, r.history...),
		llm.Message{Role: llm.RoleUser, Content: stepLimitSummaryPrompt})
	stream, err := r.llm.Complete(ctx, llm.Request{Messages: msgs, Stream: true, ToolMode: r.toolMode})
	if err != nil {
		return ""
	}
	res, cerr := llm.CollectStreaming(stream, r.fe.OnAssistantText)
	stream.Close()
	if cerr != nil {
		return ""
	}
	return res.Text
}

// finishErr classifies a failure and, when it is a genuine failure (not a
// user abort), rolls the conversation back to mark so the failed turn does not
// remain in history and break subsequent requests.
func (r *Runner) finishErr(ctx context.Context, err error, changed []string, step, mark int) Result {
	res := r.failOrAbort(ctx, err, changed, step)
	if res.Status == Failed && mark <= len(r.history) {
		r.history = r.history[:mark]
	}
	return res
}

func (r *Runner) failOrAbort(ctx context.Context, err error, changed []string, step int) Result {
	if ctx.Err() != nil {
		return Result{Status: Aborted, ChangedFiles: changed, Steps: step - 1}
	}
	r.logger.Error("agent: llm error", "step", step)
	return Result{Status: Failed, ChangedFiles: changed, Steps: step - 1, Err: err}
}

// discard drops log output (default sink).
type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

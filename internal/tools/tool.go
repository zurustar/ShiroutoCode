// Package tools implements the agent's executable tools (file, terminal, git,
// web). Tools never enforce safety themselves; all execution is mediated by the
// guardrail ToolDispatcher (single interceptor, App design Q5).
package tools

import "context"

// ToolCall is a normalized request to run a tool.
type ToolCall struct {
	Name string
	Args map[string]any
}

// ToolResult is the outcome of a tool execution.
type ToolResult struct {
	Output    string   // human/LLM-facing summary or captured output
	ExitCode  int      // for command execution (0 == success)
	Changed   []string // paths created/modified/deleted
	Truncated bool     // output was capped
}

// Tool is the common contract for every tool.
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, args map[string]any) (ToolResult, error)
}

// Registry holds the available tools by name.
type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry { return &Registry{tools: map[string]Tool{}} }

func (r *Registry) Register(t Tool) { r.tools[t.Name()] = t }

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.tools))
	for n := range r.tools {
		out = append(out, n)
	}
	return out
}

// argString safely extracts a string argument.
func argString(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

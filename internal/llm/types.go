// Package llm provides the LM Studio (OpenAI-compatible) client for
// ShiroutoCode: request assembly, SSE streaming, a hybrid tool-calling
// strategy (native function calling with a single-JSON fallback), retries, and
// a classified error model. It is front-agnostic and depends only on the
// standard library (plus U1 config/log).
package llm

import "context"

// Role is an OpenAI-compatible chat role.
type Role = string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// Message is one element of the conversation.
type Message struct {
	Role       Role
	Content    string
	ToolCallID string     // when Role == tool
	ToolCalls  []ToolCall // when Role == assistant requested tools (function mode)
}

// ToolSpec describes a tool offered to the model.
type ToolSpec struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON schema
}

// ToolCall is a tool invocation requested by the model (normalized across
// function-calling and JSON-fallback modes).
type ToolCall struct {
	ID   string
	Name string
	Args map[string]any
}

// ToolMode controls the hybrid tool-calling strategy.
type ToolMode string

const (
	ToolModeAuto     ToolMode = "auto"     // try function calling, fall back to JSON
	ToolModeFunction ToolMode = "function" // always native function calling
	ToolModeJSON     ToolMode = "json"     // always single-JSON prompt protocol
)

// Request is a completion request.
type Request struct {
	Messages    []Message
	Tools       []ToolSpec
	Stream      bool
	ToolMode    ToolMode
	Temperature *float64
	MaxTokens   *int
}

// ChunkKind discriminates streaming chunk payloads.
type ChunkKind int

const (
	ChunkText ChunkKind = iota
	ChunkToolCall
	ChunkDone
)

// Chunk is a single streaming event surfaced to callers.
type Chunk struct {
	Kind          ChunkKind
	Text          string         // ChunkText
	ToolCallDelta *ToolCallDelta // ChunkToolCall
	FinishReason  string         // ChunkDone
}

// ToolCallDelta is a partial tool-call fragment from a streaming response.
type ToolCallDelta struct {
	Index        int
	ID           string
	Name         string
	ArgsFragment string
}

// Stream yields chunks until io.EOF.
type Stream interface {
	Recv() (Chunk, error)
	Close() error
	// Mode reports the resolved tool mode (function|json) for this stream, so
	// callers know whether to apply JSON-fallback parsing on the collected text.
	Mode() ToolMode
}

// Caps records detected model capabilities.
type Caps struct {
	Known                   bool
	SupportsFunctionCalling bool
}

// CompletionResult is the aggregated result of consuming a Stream.
type CompletionResult struct {
	Text         string
	ToolCalls    []ToolCall
	FinishReason string
}

// LLMClient is the LLM connectivity surface used by the agent (U4).
type LLMClient interface {
	Complete(ctx context.Context, req Request) (Stream, error)
}

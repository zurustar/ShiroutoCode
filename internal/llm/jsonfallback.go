package llm

import (
	"encoding/json"
	"strings"
)

// jsonToolProtocol is the single-JSON fallback shape (Functional R3): the model
// must emit exactly one of {"tool","args"} or {"final"}.
type jsonToolProtocol struct {
	Tool  string         `json:"tool"`
	Args  map[string]any `json:"args"`
	Final *string        `json:"final"`
}

// fallbackSystemPrompt builds the instruction appended in JSON mode telling the
// model to respond with a single JSON object selecting a tool or finishing.
func fallbackSystemPrompt(tools []ToolSpec) string {
	var b strings.Builder
	b.WriteString("You can use tools. Respond with EXACTLY ONE JSON object and nothing else.\n")
	b.WriteString("To call a tool: {\"tool\":\"<name>\",\"args\":{...}}\n")
	b.WriteString("To finish: {\"final\":\"<your answer>\"}\n")
	if len(tools) > 0 {
		b.WriteString("Available tools:\n")
		for _, t := range tools {
			b.WriteString("- ")
			b.WriteString(t.Name)
			if t.Description != "" {
				b.WriteString(": ")
				b.WriteString(t.Description)
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

// parseJSONTool extracts the single-JSON object from a model response and
// normalizes it to a ToolCall or final text. Returns (nil, "", err) when the
// text cannot be decoded into the protocol (fail-closed; do not execute).
func parseJSONTool(text string) (*ToolCall, string, error) {
	raw := extractJSONObject(text)
	if raw == "" {
		return nil, "", newDecodeError(errNoJSON)
	}
	var p jsonToolProtocol
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return nil, "", newDecodeError(err)
	}
	if p.Tool != "" {
		return &ToolCall{Name: p.Tool, Args: p.Args}, "", nil
	}
	if p.Final != nil {
		return nil, *p.Final, nil
	}
	return nil, "", newDecodeError(errNoToolOrFinal)
}

// extractJSONObject finds the outermost {...} object in text, tolerating code
// fences and surrounding prose.
func extractJSONObject(text string) string {
	s := strings.TrimSpace(text)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end < 0 || end < start {
		return ""
	}
	return s[start : end+1]
}

var (
	errNoJSON        = &simpleErr{"no JSON object in response"}
	errNoToolOrFinal = &simpleErr{"JSON has neither tool nor final"}
)

type simpleErr struct{ s string }

func (e *simpleErr) Error() string { return e.s }

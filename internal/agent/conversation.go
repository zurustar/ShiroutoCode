package agent

import (
	"fmt"

	"github.com/zurustar/shiroutocode/internal/llm"
	"github.com/zurustar/shiroutocode/internal/tools"
)

// specs builds LLM tool specs from the registry (F3). Parameters are a
// permissive object schema; argument validation happens at execution time.
func (r *Runner) specs() []llm.ToolSpec {
	var out []llm.ToolSpec
	for _, name := range r.reg.Names() {
		t, ok := r.reg.Get(name)
		if !ok {
			continue
		}
		out = append(out, llm.ToolSpec{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  map[string]any{"type": "object"},
		})
	}
	return out
}

// appendAssistant records the assistant turn that requested tools (R6).
func appendAssistant(msgs *[]llm.Message, res llm.CompletionResult, mode llm.ToolMode) {
	m := llm.Message{Role: llm.RoleAssistant, Content: res.Text}
	if mode == llm.ToolModeFunction {
		m.ToolCalls = res.ToolCalls
	}
	*msgs = append(*msgs, m)
}

// appendObservation records a tool result (or block/error) as an observation,
// shaped per the active mode (R6).
func appendObservation(msgs *[]llm.Message, tc llm.ToolCall, out tools.ToolResult, err error, mode llm.ToolMode) {
	content := out.Output
	if err != nil {
		content = "操作はブロックまたは失敗しました: " + err.Error()
	}
	if mode == llm.ToolModeFunction {
		*msgs = append(*msgs, llm.Message{Role: llm.RoleTool, ToolCallID: tc.ID, Content: content})
		return
	}
	*msgs = append(*msgs, llm.Message{
		Role:    llm.RoleUser,
		Content: fmt.Sprintf("Tool %s result:\n%s", tc.Name, content),
	})
}

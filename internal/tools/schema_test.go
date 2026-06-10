package tools

import "testing"

// Every tool must expose a JSON Schema with a "properties" object — LM Studio /
// OpenAI reject function.parameters without it (regression from live E2E).
func TestToolSchemasHaveProperties(t *testing.T) {
	all := []Tool{
		NewReadFileTool("/ws"),
		NewFileTool("/ws"),
		NewTerminalTool("/ws", 0, nil),
		NewGitTool("/ws"),
		NewWebTool(0),
	}
	for _, tool := range all {
		sp, ok := tool.(SchemaProvider)
		if !ok {
			t.Errorf("%s: missing SchemaProvider", tool.Name())
			continue
		}
		s := sp.ParametersSchema()
		if s["type"] != "object" {
			t.Errorf("%s: type != object", tool.Name())
		}
		props, ok := s["properties"].(map[string]any)
		if !ok || len(props) == 0 {
			t.Errorf("%s: properties missing/empty", tool.Name())
		}
	}
}

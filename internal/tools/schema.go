package tools

// SchemaProvider is implemented by tools that expose a JSON Schema for their
// arguments. LM Studio (and OpenAI) require function.parameters to be a valid
// schema object with a "properties" field, so every tool provides one.
type SchemaProvider interface {
	ParametersSchema() map[string]any
}

func obj(props map[string]any, required ...string) map[string]any {
	m := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		m["required"] = required
	}
	return m
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }

func (t *ReadFileTool) ParametersSchema() map[string]any {
	return obj(map[string]any{"path": str("workspace-relative file path to read")}, "path")
}

func (t *FileTool) ParametersSchema() map[string]any {
	return obj(map[string]any{
		"path": str("workspace-relative file path"),
		"mode": map[string]any{
			"type":        "string",
			"enum":        []string{"create", "overwrite", "edit", "delete"},
			"description": "operation: create/overwrite write full content; edit replaces old_string with new_string; delete removes the file",
		},
		"content":    str("full file content for create/overwrite"),
		"old_string": str("exact unique text to replace (edit mode)"),
		"new_string": str("replacement text (edit mode)"),
	}, "path", "mode")
}

func (t *TerminalTool) ParametersSchema() map[string]any {
	return obj(map[string]any{
		"command":      str("executable name (use with args)"),
		"args":         map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "arguments for command"},
		"command_line": str("full shell command line (alternative to command+args)"),
	})
}

func (t *GitTool) ParametersSchema() map[string]any {
	return obj(map[string]any{
		"op":   str("git subcommand, e.g. status, add, commit, diff, log, branch"),
		"args": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "arguments for the git subcommand"},
	}, "op")
}

func (t *WebTool) ParametersSchema() map[string]any {
	return obj(map[string]any{"url": str("http(s) URL to fetch with GET")}, "url")
}

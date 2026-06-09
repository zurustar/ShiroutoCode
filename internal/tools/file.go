package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileTool reads and mutates files within the workspace. Scope enforcement is
// performed by the guardrail; this tool resolves paths relative to the
// workspace root and writes atomically (NFR design P4).
type FileTool struct {
	root string
}

func NewFileTool(workspaceRoot string) *FileTool { return &FileTool{root: workspaceRoot} }

func (t *FileTool) Name() string { return "write_file" }
func (t *FileTool) Description() string {
	return "Read, create, overwrite, edit, or delete a workspace file."
}

func (t *FileTool) abs(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(t.root, path))
}

func (t *FileTool) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	mode := argString(args, "mode")
	path := argString(args, "path")
	if path == "" {
		return ToolResult{}, fmt.Errorf("path is required")
	}
	abs := t.abs(path)

	switch mode {
	case "create", "overwrite", "":
		content := argString(args, "content")
		if mode == "create" {
			if _, err := os.Stat(abs); err == nil {
				return ToolResult{}, fmt.Errorf("file already exists: %s", path)
			}
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return ToolResult{}, err
		}
		if err := writeAtomic(abs, []byte(content)); err != nil {
			return ToolResult{}, err
		}
		return ToolResult{Output: fmt.Sprintf("wrote %s (%d bytes)", path, len(content)), Changed: []string{path}}, nil

	case "edit":
		old := argString(args, "old_string")
		nw := argString(args, "new_string")
		if old == "" {
			return ToolResult{}, fmt.Errorf("edit requires old_string")
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return ToolResult{}, err
		}
		n := strings.Count(string(data), old)
		if n == 0 {
			return ToolResult{}, fmt.Errorf("old_string not found in %s", path)
		}
		if n > 1 {
			return ToolResult{}, fmt.Errorf("old_string is not unique in %s (%d matches)", path, n)
		}
		updated := strings.Replace(string(data), old, nw, 1)
		if err := writeAtomic(abs, []byte(updated)); err != nil {
			return ToolResult{}, err
		}
		return ToolResult{Output: fmt.Sprintf("edited %s", path), Changed: []string{path}}, nil

	case "delete":
		if err := os.Remove(abs); err != nil {
			return ToolResult{}, err
		}
		return ToolResult{Output: fmt.Sprintf("deleted %s", path), Changed: []string{path}}, nil

	default:
		return ToolResult{}, fmt.Errorf("unknown mode: %q", mode)
	}
}

// ReadFileTool reads a workspace file (separate tool name for clarity).
type ReadFileTool struct{ root string }

func NewReadFileTool(workspaceRoot string) *ReadFileTool { return &ReadFileTool{root: workspaceRoot} }

func (t *ReadFileTool) Name() string        { return "read_file" }
func (t *ReadFileTool) Description() string { return "Read the contents of a workspace file." }

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	path := argString(args, "path")
	if path == "" {
		return ToolResult{}, fmt.Errorf("path is required")
	}
	abs := path
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(t.root, abs)
	}
	data, err := os.ReadFile(filepath.Clean(abs))
	if err != nil {
		return ToolResult{}, err
	}
	return ToolResult{Output: string(data)}, nil
}

// writeAtomic writes data to a temp file in the same directory and renames it
// into place, leaving no partial file on failure (R9/P4).
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".shirouto-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // no-op if renamed

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

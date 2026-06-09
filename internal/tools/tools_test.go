package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileCreateReadEditDelete(t *testing.T) {
	root := t.TempDir()
	ctx := context.Background()
	ft := NewFileTool(root)
	rt := NewReadFileTool(root)

	// create
	if _, err := ft.Execute(ctx, map[string]any{"path": "a.txt", "mode": "create", "content": "hello world"}); err != nil {
		t.Fatalf("create: %v", err)
	}
	// create again should fail
	if _, err := ft.Execute(ctx, map[string]any{"path": "a.txt", "mode": "create", "content": "x"}); err == nil {
		t.Errorf("create existing should fail")
	}
	// read
	res, err := rt.Execute(ctx, map[string]any{"path": "a.txt"})
	if err != nil || res.Output != "hello world" {
		t.Fatalf("read: %v %q", err, res.Output)
	}
	// edit unique
	if _, err := ft.Execute(ctx, map[string]any{"path": "a.txt", "mode": "edit", "old_string": "world", "new_string": "gophers"}); err != nil {
		t.Fatalf("edit: %v", err)
	}
	res, _ = rt.Execute(ctx, map[string]any{"path": "a.txt"})
	if res.Output != "hello gophers" {
		t.Errorf("after edit: %q", res.Output)
	}
	// delete
	if _, err := ft.Execute(ctx, map[string]any{"path": "a.txt", "mode": "delete"}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "a.txt")); !os.IsNotExist(err) {
		t.Errorf("file should be deleted")
	}
}

func TestFileEditNonUniqueFails(t *testing.T) {
	root := t.TempDir()
	ft := NewFileTool(root)
	ctx := context.Background()
	_, _ = ft.Execute(ctx, map[string]any{"path": "b.txt", "mode": "overwrite", "content": "x x x"})
	if _, err := ft.Execute(ctx, map[string]any{"path": "b.txt", "mode": "edit", "old_string": "x", "new_string": "y"}); err == nil {
		t.Errorf("non-unique edit should fail")
	}
}

func TestTerminalEcho(t *testing.T) {
	tt := NewTerminalTool(t.TempDir(), 10*time.Second, nil)
	res, err := tt.Execute(context.Background(), map[string]any{"command": "echo", "args": []any{"hello"}})
	if err != nil {
		t.Fatalf("echo: %v", err)
	}
	if res.ExitCode != 0 || !strings.Contains(res.Output, "hello") {
		t.Errorf("echo result: code=%d out=%q", res.ExitCode, res.Output)
	}
}

func TestTerminalTimeoutKills(t *testing.T) {
	tt := NewTerminalTool(t.TempDir(), 100*time.Millisecond, nil)
	start := time.Now()
	_, err := tt.Execute(context.Background(), map[string]any{"command_line": "sleep 5"})
	if err == nil {
		t.Errorf("expected timeout error")
	}
	if time.Since(start) > 3*time.Second {
		t.Errorf("timeout did not kill promptly: %s", time.Since(start))
	}
}

func TestTerminalNonZeroExit(t *testing.T) {
	tt := NewTerminalTool(t.TempDir(), 10*time.Second, nil)
	res, err := tt.Execute(context.Background(), map[string]any{"command_line": "exit 3"})
	if err != nil {
		t.Fatalf("unexpected go error: %v", err)
	}
	if res.ExitCode != 3 {
		t.Errorf("exit code = %d, want 3", res.ExitCode)
	}
}

func TestGitStatus(t *testing.T) {
	if _, err := os.Stat("/usr/bin/git"); err != nil {
		// best-effort: skip if git missing in PATH check below
	}
	root := t.TempDir()
	gt := NewGitTool(root)
	ctx := context.Background()
	// init a repo
	init := NewTerminalTool(root, 10*time.Second, nil)
	if _, err := init.Execute(ctx, map[string]any{"command_line": "git init -q"}); err != nil {
		t.Skipf("git not available: %v", err)
	}
	res, err := gt.Execute(ctx, map[string]any{"op": "status", "args": []any{"--porcelain"}})
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("git status exit = %d, out=%q", res.ExitCode, res.Output)
	}
}

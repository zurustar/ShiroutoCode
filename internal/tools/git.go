package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// GitTool runs git operations in the workspace via the git CLI. Dangerous
// operations are gated by the guardrail before reaching here.
type GitTool struct {
	root string
}

func NewGitTool(workspaceRoot string) *GitTool { return &GitTool{root: workspaceRoot} }

func (t *GitTool) Name() string { return "git" }
func (t *GitTool) Description() string {
	return "Run a git operation in the workspace (commit, branch, status, diff, ...)."
}

func (t *GitTool) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	op := argString(args, "op")
	if op == "" {
		return ToolResult{}, fmt.Errorf("op is required")
	}
	gitArgs := []string{op}
	if raw, ok := args["args"].([]any); ok {
		for _, a := range raw {
			if s, ok := a.(string); ok {
				gitArgs = append(gitArgs, s)
			}
		}
	}
	cmd := exec.CommandContext(ctx, "git", gitArgs...)
	cmd.Dir = t.root
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	res := ToolResult{Output: out.String()}
	if err != nil {
		var ee *exec.ExitError
		if errorsAsExit(err, &ee) {
			res.ExitCode = ee.ExitCode()
			return res, nil
		}
		res.ExitCode = -1
		return res, err
	}
	return res, nil
}

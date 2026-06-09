package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"syscall"
	"time"
)

// TerminalTool runs shell commands inside the workspace. The process is started
// in its own process group so the whole tree can be killed on timeout/cancel
// (NFR design P3). Output is captured up to a cap and optionally streamed.
type TerminalTool struct {
	root    string
	timeout time.Duration
	maxOut  int
	sink    io.Writer // optional: live streaming (U5)
}

const defaultMaxOutput = 1 << 20 // 1 MiB

func NewTerminalTool(workspaceRoot string, timeout time.Duration, sink io.Writer) *TerminalTool {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &TerminalTool{root: workspaceRoot, timeout: timeout, maxOut: defaultMaxOutput, sink: sink}
}

func (t *TerminalTool) Name() string { return "run_command" }
func (t *TerminalTool) Description() string {
	return "Run a shell command in the workspace and capture its output."
}

func (t *TerminalTool) Execute(ctx context.Context, args map[string]any) (ToolResult, error) {
	cmdLine := argString(args, "command_line")
	command := argString(args, "command")
	if cmdLine == "" && command == "" {
		return ToolResult{}, fmt.Errorf("command_line or command is required")
	}

	cctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	var cmd *exec.Cmd
	if cmdLine != "" {
		cmd = exec.CommandContext(cctx, "sh", "-c", cmdLine)
	} else {
		var cargs []string
		if raw, ok := args["args"].([]any); ok {
			for _, a := range raw {
				if s, ok := a.(string); ok {
					cargs = append(cargs, s)
				}
			}
		}
		cmd = exec.CommandContext(cctx, command, cargs...)
	}
	cmd.Dir = t.root
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			// Kill the whole process group.
			return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		return nil
	}
	cmd.WaitDelay = 2 * time.Second

	cw := &capWriter{max: t.maxOut, sink: t.sink}
	cmd.Stdout = cw
	cmd.Stderr = cw

	err := cmd.Run()
	res := ToolResult{Output: cw.buf.String(), Truncated: cw.truncated}
	if cctx.Err() == context.DeadlineExceeded {
		res.ExitCode = -1
		return res, fmt.Errorf("command timed out after %s", t.timeout)
	}
	if err != nil {
		var ee *exec.ExitError
		if errorsAsExit(err, &ee) {
			res.ExitCode = ee.ExitCode()
			return res, nil // non-zero exit is reported, not a Go error
		}
		res.ExitCode = -1
		return res, err
	}
	return res, nil
}

// capWriter captures up to max bytes and optionally mirrors to a sink.
type capWriter struct {
	buf       bytes.Buffer
	max       int
	truncated bool
	sink      io.Writer
}

func (w *capWriter) Write(p []byte) (int, error) {
	if w.sink != nil {
		_, _ = w.sink.Write(p)
	}
	remaining := w.max - w.buf.Len()
	if remaining <= 0 {
		w.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		w.buf.Write(p[:remaining])
		w.truncated = true
		return len(p), nil
	}
	w.buf.Write(p)
	return len(p), nil
}

func errorsAsExit(err error, target **exec.ExitError) bool {
	for err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			*target = ee
			return true
		}
		type unwrap interface{ Unwrap() error }
		if u, ok := err.(unwrap); ok {
			err = u.Unwrap()
		} else {
			break
		}
	}
	return false
}

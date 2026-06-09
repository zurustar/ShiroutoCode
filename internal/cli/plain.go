package cli

import (
	"fmt"
	"io"
)

// plainFrontend renders agent events as plain labeled text to a writer
// (single-shot / non-TTY). Implements agent.Frontend.
type plainFrontend struct {
	w io.Writer
}

func (f *plainFrontend) OnAssistantText(delta string) { fmt.Fprint(f.w, delta) }

func (f *plainFrontend) OnToolCall(name string, args map[string]any) {
	fmt.Fprintf(f.w, "\n[tool] %s %v\n", name, args)
}

func (f *plainFrontend) OnToolResult(name, output string, err error) {
	if err != nil {
		fmt.Fprintf(f.w, "[tool:%s] error: %s\n", name, err)
		return
	}
	fmt.Fprintf(f.w, "[tool:%s] %s\n", name, truncate(output, 2000))
}

func (f *plainFrontend) OnStep(current, max int) {
	fmt.Fprintf(f.w, "\n[step %d/%d]\n", current, max)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…(truncated)"
}

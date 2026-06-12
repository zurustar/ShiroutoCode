package cli

import (
	"fmt"
	"io"
)

// plainFrontend renders agent events as plain labeled text to a writer
// (single-shot / non-TTY / REPL). Implements agent.Frontend.
//
// It tracks whether the assistant produced any text this run so the caller can
// tell a genuinely empty response apart from one that was somehow dropped.
type plainFrontend struct {
	w         io.Writer
	wroteText bool // any assistant text emitted since the last reset
	started   bool // assistant-text prefix already printed for the current block
}

// reset clears per-run state (call before each Run).
func (f *plainFrontend) reset() {
	f.wroteText = false
	f.started = false
}

func (f *plainFrontend) OnAssistantText(delta string) {
	if delta == "" {
		return
	}
	if !f.started {
		fmt.Fprint(f.w, "\n🤖 ")
		f.started = true
	}
	f.wroteText = true
	fmt.Fprint(f.w, delta)
}

func (f *plainFrontend) OnToolCall(name string, args map[string]any) {
	f.started = false // a new assistant-text block will start after the tool
	fmt.Fprintf(f.w, "\n🔧 %s %v\n", name, args)
}

func (f *plainFrontend) OnToolResult(name, output string, err error) {
	if err != nil {
		fmt.Fprintf(f.w, "   ⚠ [%s] エラー: %s\n", name, err)
		return
	}
	fmt.Fprintf(f.w, "   → [%s] %s\n", name, truncate(output, 2000))
}

func (f *plainFrontend) OnStep(current, max int) {
	fmt.Fprintf(f.w, "\n— step %d/%d —\n", current, max)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…(truncated)"
}

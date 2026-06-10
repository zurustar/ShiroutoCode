package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/zurustar/shiroutocode/internal/guardrail"
)

// promptConfirmer asks the user to approve a guarded action via a TTY prompt.
// Implements guardrail.Confirmer.
type promptConfirmer struct {
	in  *bufio.Reader
	out io.Writer
}

func newPromptConfirmer(in io.Reader, out io.Writer) *promptConfirmer {
	return &promptConfirmer{in: bufio.NewReader(in), out: out}
}

func (c *promptConfirmer) Confirm(ctx context.Context, a guardrail.Action, reason string) (bool, error) {
	fmt.Fprintf(c.out, "\n⚠ 確認が必要です: %s\n  ツール: %s\n実行しますか? [y/N]: ", reason, a.Tool)
	line, err := c.in.ReadString('\n')
	if err != nil {
		// Any read error (EOF, broken pipe, partial line) is a decline — never
		// approve on a confirmation we could not fully read (F-09, fail-safe).
		return false, nil
	}
	return parseYes(line), nil
}

func parseYes(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "y" || s == "yes"
}

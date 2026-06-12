package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

// errInterrupted is returned by a line reader when the user presses Ctrl+C.
var errInterrupted = errors.New("interrupted")

// lineReader reads one prompt line. Implementations: a raw-mode editor on a
// real terminal, or a plain cooked fallback for pipes/tests.
type lineReader func(prompt string) (string, error)

// newLineReader returns a line reader for fd. On a real terminal it edits in
// raw mode with UTF-8 + display-width awareness, so one backspace deletes a
// whole (possibly full-width, e.g. Japanese) character and Ctrl+C exits at
// once. When fd is not a terminal (tests, pipes) it falls back to a cooked
// line read so behavior stays simple and testable.
func newLineReader(fd int, r *bufio.Reader, out io.Writer) lineReader {
	if fd < 0 || !term.IsTerminal(fd) {
		return func(prompt string) (string, error) {
			fmt.Fprint(out, prompt)
			line, err := r.ReadString('\n')
			if err != nil {
				return "", io.EOF
			}
			return line, nil
		}
	}
	return func(prompt string) (string, error) {
		return readLineRaw(fd, r, out, prompt)
	}
}

// readLineRaw reads and edits a single line in raw mode.
func readLineRaw(fd int, r *bufio.Reader, out io.Writer, prompt string) (string, error) {
	state, err := term.MakeRaw(fd)
	if err != nil {
		// Not a raw-capable terminal: degrade to a cooked read.
		fmt.Fprint(out, prompt)
		line, rerr := r.ReadString('\n')
		if rerr != nil {
			return "", io.EOF
		}
		return line, nil
	}
	defer term.Restore(fd, state)

	fmt.Fprint(out, prompt)
	var buf []rune
	for {
		c, _, rerr := r.ReadRune()
		if rerr != nil {
			if len(buf) > 0 {
				return string(buf), nil
			}
			return "", io.EOF
		}
		switch c {
		case '\r', '\n':
			fmt.Fprint(out, "\r\n")
			return string(buf), nil
		case 0x03: // Ctrl+C
			fmt.Fprint(out, "\r\n")
			return "", errInterrupted
		case 0x04: // Ctrl+D
			if len(buf) == 0 {
				fmt.Fprint(out, "\r\n")
				return "", io.EOF
			}
		case 0x7f, 0x08: // Backspace / DEL
			if len(buf) > 0 {
				last := buf[len(buf)-1]
				buf = buf[:len(buf)-1]
				eraseCells(out, runewidth.RuneWidth(last))
			}
		case 0x15: // Ctrl+U: clear the line
			eraseCells(out, lineWidth(buf))
			buf = buf[:0]
		case 0x1b: // escape sequence (arrow keys, etc.): swallow and ignore
			swallowEscape(r)
		default:
			if c >= 0x20 { // printable (includes multibyte / CJK)
				buf = append(buf, c)
				fmt.Fprint(out, string(c))
			}
		}
	}
}

// eraseCells erases n character cells to the left of the cursor.
func eraseCells(out io.Writer, n int) {
	if n <= 0 {
		return
	}
	fmt.Fprint(out, strings.Repeat("\b", n)+strings.Repeat(" ", n)+strings.Repeat("\b", n))
}

func lineWidth(runes []rune) int {
	w := 0
	for _, r := range runes {
		w += runewidth.RuneWidth(r)
	}
	return w
}

// swallowEscape consumes a CSI/SS3 escape sequence so navigation keys do not
// corrupt the line. Only bytes already buffered are read, so a lone ESC does
// not block.
func swallowEscape(r *bufio.Reader) {
	if r.Buffered() == 0 {
		return
	}
	b, err := r.ReadByte()
	if err != nil {
		return
	}
	if b != '[' && b != 'O' {
		return
	}
	for r.Buffered() > 0 {
		fb, err := r.ReadByte()
		if err != nil {
			return
		}
		if fb >= 0x40 && fb <= 0x7e { // final byte of the sequence
			return
		}
	}
}

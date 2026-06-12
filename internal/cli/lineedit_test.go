package cli

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestLineWidthCountsFullWidth(t *testing.T) {
	// ASCII is 1 cell each; Japanese kana/kanji are 2 cells each.
	if w := lineWidth([]rune("ab")); w != 2 {
		t.Errorf("lineWidth(ab) = %d, want 2", w)
	}
	if w := lineWidth([]rune("あ")); w != 2 {
		t.Errorf("lineWidth(あ) = %d, want 2", w)
	}
	if w := lineWidth([]rune("aあ")); w != 3 {
		t.Errorf("lineWidth(aあ) = %d, want 3", w)
	}
}

func TestEraseCellsEmitsBalancedSequence(t *testing.T) {
	var b bytes.Buffer
	eraseCells(&b, 2) // erasing a full-width char: back 2, 2 spaces, back 2
	want := "\b\b  \b\b"
	if b.String() != want {
		t.Errorf("eraseCells(2) = %q, want %q", b.String(), want)
	}
	b.Reset()
	eraseCells(&b, 0)
	if b.String() != "" {
		t.Errorf("eraseCells(0) should emit nothing, got %q", b.String())
	}
}

// The non-terminal fallback reader writes the prompt and returns the line.
func TestNewLineReaderFallback(t *testing.T) {
	in := bufio.NewReader(strings.NewReader("こんにちは\n"))
	var out bytes.Buffer
	read := newLineReader(-1, in, &out)
	line, err := read("> ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(line) != "こんにちは" {
		t.Errorf("line = %q, want こんにちは", line)
	}
	if !strings.Contains(out.String(), "> ") {
		t.Errorf("prompt not written: %q", out.String())
	}
}

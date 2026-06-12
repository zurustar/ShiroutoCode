package cli

import (
	"context"
	"io"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/zurustar/shiroutocode/internal/llm"
)

func TestPickerNavigationClamps(t *testing.T) {
	p := newPicker([]string{"a", "b", "c"}, "")
	if p.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", p.cursor)
	}
	p.up() // already at top, stays
	if p.cursor != 0 {
		t.Errorf("up at top moved to %d", p.cursor)
	}
	p.down()
	p.down()
	if p.cursor != 2 || p.selected() != "c" {
		t.Errorf("cursor=%d selected=%q, want 2/c", p.cursor, p.selected())
	}
	p.down() // at bottom, stays
	if p.cursor != 2 {
		t.Errorf("down at bottom moved to %d", p.cursor)
	}
}

func TestPickerPreselect(t *testing.T) {
	p := newPicker([]string{"a", "b", "c"}, "b")
	if p.selected() != "b" {
		t.Errorf("preselect: selected = %q, want b", p.selected())
	}
	// Unknown preselect falls back to the first entry.
	p2 := newPicker([]string{"a", "b"}, "zzz")
	if p2.cursor != 0 {
		t.Errorf("unknown preselect cursor = %d, want 0", p2.cursor)
	}
}

func TestPickerEmptySelectedIsBlank(t *testing.T) {
	p := newPicker(nil, "")
	if p.selected() != "" {
		t.Errorf("empty picker selected = %q, want blank", p.selected())
	}
}

func TestPickerViewMarksCursor(t *testing.T) {
	p := newPicker([]string{"alpha", "beta"}, "beta")
	out := p.view("選択")
	if !strings.Contains(out, "beta") || !strings.Contains(out, "alpha") {
		t.Errorf("view missing entries: %q", out)
	}
	// The selected line must carry the cursor marker.
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "beta") && !strings.Contains(line, ">") {
			t.Errorf("cursor not on selected line: %q", line)
		}
	}
}

// Cursor stays within bounds under any sequence of moves (PBT).
func TestPickerCursorAlwaysInBoundsPBT(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 8).Draw(t, "n")
		models := make([]string, n)
		for i := range models {
			models[i] = string(rune('a' + i))
		}
		p := newPicker(models, "")
		moves := rapid.SliceOfN(rapid.IntRange(0, 1), 0, 30).Draw(t, "moves")
		for _, mv := range moves {
			if mv == 0 {
				p.up()
			} else {
				p.down()
			}
			if p.cursor < 0 || p.cursor >= len(models) {
				t.Fatalf("cursor out of bounds: %d (n=%d)", p.cursor, n)
			}
		}
	})
}

// selectModelStandalone surfaces a fetch error before launching any UI.
func TestSelectModelStandaloneListError(t *testing.T) {
	lister := &fakeClient{listErr: &llm.LLMError{Kind: llm.ErrUnreachable, UserMessage: "接続できません"}}
	_, err := selectModelStandalone(context.Background(), lister, "", io.Discard)
	if err == nil {
		t.Fatal("expected error from failed fetch")
	}
}

// An empty model list is reported rather than opening an empty picker.
func TestSelectModelStandaloneEmpty(t *testing.T) {
	lister := &fakeClient{models: nil}
	_, err := selectModelStandalone(context.Background(), lister, "", io.Discard)
	if err == nil {
		t.Fatal("expected error for empty model list")
	}
	if !strings.Contains(err.Error(), "モデル") {
		t.Errorf("unexpected message: %v", err)
	}
}

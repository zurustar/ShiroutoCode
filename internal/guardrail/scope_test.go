package guardrail

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// R3 (PBT): paths under the workspace resolve inside; escaping paths resolve
// outside.
func TestScopeContainmentPBT(t *testing.T) {
	root := t.TempDir()
	rapid.Check(t, func(rt *rapid.T) {
		segs := rapid.SliceOfN(rapid.StringMatching(`[a-z]{1,5}`), 1, 4).Draw(rt, "segs")
		rel := filepath.Join(segs...)
		_, within, err := resolveWithin(root, rel)
		if err != nil || !within {
			rt.Fatalf("expected %q within root, within=%v err=%v", rel, within, err)
		}

		// escape with enough ../
		escape := strings.Repeat("../", len(segs)+3) + filepath.Join(segs...)
		_, within2, err2 := resolveWithin(root, escape)
		if err2 == nil && within2 {
			rt.Fatalf("expected %q outside root", escape)
		}
	})
}

func TestScopeSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	// create a symlink inside root pointing outside
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	_, within, err := resolveWithin(root, "link/secret.txt")
	if err == nil && within {
		t.Errorf("symlink escape should not be within root")
	}
}

func TestScopeAbsoluteInsideAndOutside(t *testing.T) {
	root := t.TempDir()
	_, within, err := resolveWithin(root, filepath.Join(root, "x", "y.txt"))
	if err != nil || !within {
		t.Errorf("absolute inside should be within: within=%v err=%v", within, err)
	}
	_, within2, _ := resolveWithin(root, "/etc/passwd")
	if within2 {
		t.Errorf("/etc/passwd should be outside")
	}
}

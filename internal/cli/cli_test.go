package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/zurustar/shiroutocode/internal/guardrail"
)

// R3 (PBT): only y/yes (case-insensitive, trimmed) confirm.
func TestParseYesPBT(t *testing.T) {
	yes := map[string]bool{"y": true, "Y": true, "yes": true, "YES": true, " y ": true, "Yes\n": true}
	for s, want := range yes {
		if parseYes(s) != want {
			t.Errorf("parseYes(%q)=%v want %v", s, parseYes(s), want)
		}
	}
	rapid.Check(t, func(rt *rapid.T) {
		s := rapid.String().Draw(rt, "s")
		norm := strings.ToLower(strings.TrimSpace(s))
		want := norm == "y" || norm == "yes"
		if parseYes(s) != want {
			rt.Fatalf("parseYes(%q)=%v want %v", s, parseYes(s), want)
		}
	})
}

func TestPromptConfirmer(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{{"y\n", true}, {"yes\n", true}, {"n\n", false}, {"\n", false}, {"maybe\n", false}}
	for _, c := range cases {
		var out bytes.Buffer
		pc := newPromptConfirmer(strings.NewReader(c.in), &out)
		ok, err := pc.Confirm(context.Background(), guardrail.Action{Tool: "run_command"}, "危険")
		if err != nil || ok != c.want {
			t.Errorf("in=%q -> ok=%v err=%v want=%v", c.in, ok, err, c.want)
		}
		if !strings.Contains(out.String(), "確認が必要") {
			t.Errorf("prompt not shown for %q", c.in)
		}
	}
}

func TestPlainFrontend(t *testing.T) {
	var buf bytes.Buffer
	f := &plainFrontend{w: &buf}
	f.OnAssistantText("hello")
	f.OnToolCall("read_file", map[string]any{"path": "a"})
	f.OnToolResult("read_file", "content", nil)
	f.OnToolResult("run_command", "", &guardrail.BlockedError{Reason: "blocked"})
	f.OnStep(1, 5)
	s := buf.String()
	for _, want := range []string{"hello", "[tool] read_file", "[tool:read_file] content", "error: blocked", "[step 1/5]"} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q in:\n%s", want, s)
		}
	}
}

func TestExtractPrompt(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"do", "the", "thing"}, "do the thing"},
		{[]string{"-model", "m1", "fix bug"}, "fix bug"},
		{[]string{"-model", "m1"}, ""},
		{[]string{}, ""},
	}
	var errOut bytes.Buffer
	for _, c := range cases {
		got, err := extractPrompt(c.args, &errOut)
		if err != nil {
			t.Fatalf("args=%v err=%v", c.args, err)
		}
		if got != c.want {
			t.Errorf("args=%v -> %q want %q", c.args, got, c.want)
		}
	}
}

// Non-TTY with no prompt -> usage error exit code.
func TestRunNoPromptNonTTY(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run(context.Background(), nil, &out, &errb, strings.NewReader(""),
		Env{"SHIROUTO_MODEL": "m", "SHIROUTO_WORKSPACE": t.TempDir()}, false)
	if code != exitUsage {
		t.Errorf("exit = %d, want %d", code, exitUsage)
	}
	if !strings.Contains(errb.String(), "使い方") {
		t.Errorf("usage not shown: %s", errb.String())
	}
}

// Missing required config (model) -> usage error.
func TestRunMissingModel(t *testing.T) {
	var out, errb bytes.Buffer
	code := Run(context.Background(), []string{"hello"}, &out, &errb, strings.NewReader(""),
		Env{"SHIROUTO_WORKSPACE": t.TempDir()}, false)
	if code != exitUsage {
		t.Errorf("exit = %d, want %d", code, exitUsage)
	}
	if !strings.Contains(errb.String(), "設定エラー") {
		t.Errorf("config error not shown: %s", errb.String())
	}
}

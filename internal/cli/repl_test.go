package cli

import (
	"bufio"
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/zurustar/shiroutocode/internal/agent"
	"github.com/zurustar/shiroutocode/internal/llm"
)

// A prompt runs the agent (streaming feedback + summary), then /exit ends it.
func TestREPLRunsThenExits(t *testing.T) {
	core := testCore(t, &fakeClient{
		model: "m",
		chunks: []llm.Chunk{
			{Kind: llm.ChunkText, Text: "やりました"},
			{Kind: llm.ChunkDone, FinishReason: "stop"},
		},
	})
	in := bufio.NewReader(strings.NewReader("テストを書いて\n/exit\n"))
	var out, errb bytes.Buffer

	code := replLoop(context.Background(), core, &out, &errb, in)
	if code != exitOK {
		t.Fatalf("exit = %d, want %d; stderr=%s", code, exitOK, errb.String())
	}
	s := out.String()
	for _, want := range []string{"> ", "▶ 実行中…", "やりました", "完了"} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q:\n%s", want, s)
		}
	}
}

// EOF (Ctrl+D) at the prompt ends the session cleanly.
func TestREPLEOFExits(t *testing.T) {
	core := testCore(t, &fakeClient{model: "m"})
	in := bufio.NewReader(strings.NewReader(""))
	var out, errb bytes.Buffer
	if code := replLoop(context.Background(), core, &out, &errb, in); code != exitOK {
		t.Errorf("exit = %d, want %d", code, exitOK)
	}
}

// Empty lines are ignored; /help prints guidance; /exit quits.
func TestREPLHelpAndEmptyLines(t *testing.T) {
	core := testCore(t, &fakeClient{model: "m"})
	in := bufio.NewReader(strings.NewReader("\n   \n/help\n/exit\n"))
	var out, errb bytes.Buffer
	if code := replLoop(context.Background(), core, &out, &errb, in); code != exitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "/model") {
		t.Errorf("/help did not print guidance:\n%s", out.String())
	}
}

// A canceled context stops the loop with the aborted exit code.
func TestREPLContextCanceled(t *testing.T) {
	core := testCore(t, &fakeClient{model: "m"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	in := bufio.NewReader(strings.NewReader("anything\n"))
	var out, errb bytes.Buffer
	if code := replLoop(ctx, core, &out, &errb, in); code != exitAborted {
		t.Errorf("exit = %d, want %d", code, exitAborted)
	}
}

func TestDoneSummary(t *testing.T) {
	cases := []struct {
		res  agent.Result
		want string
	}{
		{agent.Result{Status: agent.Completed, Steps: 3, ChangedFiles: []string{"a.go"}}, "完了"},
		{agent.Result{Status: agent.StoppedMaxSteps}, "最大ステップ"},
		{agent.Result{Status: agent.Aborted}, "中断"},
		{agent.Result{Status: agent.Failed}, "❌"},
	}
	for _, c := range cases {
		if got := doneSummary(c.res); !strings.Contains(got, c.want) {
			t.Errorf("doneSummary(%v) = %q, want substring %q", c.res.Status, got, c.want)
		}
	}
}

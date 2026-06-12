package agent

import (
	"context"
	"io"
	"sync/atomic"
	"testing"

	"pgregory.net/rapid"

	"github.com/zurustar/shiroutocode/internal/llm"
	"github.com/zurustar/shiroutocode/internal/tools"
)

// --- fakes ---

// fakeStream yields a fixed list of chunks then io.EOF.
type fakeStream struct {
	chunks []llm.Chunk
	i      int
	mode   llm.ToolMode
}

func (s *fakeStream) Recv() (llm.Chunk, error) {
	if s.i >= len(s.chunks) {
		return llm.Chunk{}, io.EOF
	}
	c := s.chunks[s.i]
	s.i++
	return c, nil
}
func (s *fakeStream) Close() error       { return nil }
func (s *fakeStream) Mode() llm.ToolMode { return s.mode }

func textStream(text string) *fakeStream {
	return &fakeStream{mode: llm.ToolModeFunction, chunks: []llm.Chunk{
		{Kind: llm.ChunkText, Text: text},
		{Kind: llm.ChunkDone, FinishReason: "stop"},
	}}
}

func toolStream(name, id string) *fakeStream {
	return &fakeStream{mode: llm.ToolModeFunction, chunks: []llm.Chunk{
		{Kind: llm.ChunkToolCall, ToolCallDelta: &llm.ToolCallDelta{Index: 0, ID: id, Name: name, ArgsFragment: "{}"}},
		{Kind: llm.ChunkDone, FinishReason: "tool_calls"},
	}}
}

// fakeLLM returns scripted streams in order; the last is reused if exhausted.
type fakeLLM struct {
	streams  []*fakeStream
	calls    int32
	requests [][]llm.Message // messages seen on each Complete call
}

func (f *fakeLLM) Complete(ctx context.Context, req llm.Request) (llm.Stream, error) {
	f.requests = append(f.requests, append([]llm.Message(nil), req.Messages...))
	n := int(atomic.AddInt32(&f.calls, 1)) - 1
	if n >= len(f.streams) {
		n = len(f.streams) - 1
	}
	s := *f.streams[n]
	return &s, nil
}

// fakeDispatcher records calls and returns a canned result.
type fakeDispatcher struct {
	calls   int32
	changed []string
	err     error
}

func (d *fakeDispatcher) Dispatch(ctx context.Context, call tools.ToolCall) (tools.ToolResult, error) {
	atomic.AddInt32(&d.calls, 1)
	if d.err != nil {
		return tools.ToolResult{}, d.err
	}
	return tools.ToolResult{Output: "ok", Changed: d.changed}, nil
}

type recordFrontend struct {
	text     string
	steps    int
	toolCall int
}

func (f *recordFrontend) OnAssistantText(s string)           { f.text += s }
func (f *recordFrontend) OnToolCall(string, map[string]any)  { f.toolCall++ }
func (f *recordFrontend) OnToolResult(string, string, error) {}
func (f *recordFrontend) OnStep(int, int)                    { f.steps++ }

func reg() *tools.Registry { return tools.NewRegistry() }

// R2: single response with no tool calls completes immediately.
func TestSingleShotCompletes(t *testing.T) {
	fe := &recordFrontend{}
	r := NewRunner(&fakeLLM{streams: []*fakeStream{textStream("all done")}}, &fakeDispatcher{}, reg(), WithFrontend(fe))
	res, err := r.Run(context.Background(), Task{Prompt: "do it"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Status != Completed || res.Summary != "all done" || res.Steps != 1 {
		t.Errorf("res = %+v", res)
	}
	if fe.text != "all done" || fe.steps != 1 {
		t.Errorf("frontend text=%q steps=%d", fe.text, fe.steps)
	}
}

// R1/R5: tool call then completion runs two steps via the dispatcher.
func TestToolThenComplete(t *testing.T) {
	fe := &recordFrontend{}
	disp := &fakeDispatcher{changed: []string{"a.go"}}
	llmc := &fakeLLM{streams: []*fakeStream{toolStream("write_file", "c1"), textStream("finished")}}
	r := NewRunner(llmc, disp, reg(), WithFrontend(fe))
	res, err := r.Run(context.Background(), Task{Prompt: "edit"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Status != Completed || res.Steps != 2 {
		t.Errorf("res = %+v", res)
	}
	if atomic.LoadInt32(&disp.calls) != 1 {
		t.Errorf("dispatch calls = %d", disp.calls)
	}
	if len(res.ChangedFiles) != 1 || res.ChangedFiles[0] != "a.go" {
		t.Errorf("changed = %v", res.ChangedFiles)
	}
	if fe.toolCall != 1 {
		t.Errorf("OnToolCall = %d", fe.toolCall)
	}
}

// R3 (PBT): a model that always calls a tool stops at exactly maxSteps.
func TestMaxStepsTerminationPBT(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		max := rapid.IntRange(1, 12).Draw(rt, "max")
		disp := &fakeDispatcher{}
		llmc := &fakeLLM{streams: []*fakeStream{toolStream("run_command", "c1")}} // reused forever
		r := NewRunner(llmc, disp, reg(), WithMaxSteps(max))
		res, err := r.Run(context.Background(), Task{Prompt: "loop"})
		if err != nil {
			rt.Fatalf("run: %v", err)
		}
		if res.Status != StoppedMaxSteps || res.Steps != max {
			rt.Fatalf("max=%d: status=%v steps=%d", max, res.Status, res.Steps)
		}
		if int(disp.calls) != max {
			rt.Fatalf("max=%d: dispatch calls=%d", max, disp.calls)
		}
	})
}

// R4: cancellation aborts promptly.
func TestCancelAborts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled
	disp := &fakeDispatcher{}
	llmc := &fakeLLM{streams: []*fakeStream{toolStream("run_command", "c1")}}
	r := NewRunner(llmc, disp, reg(), WithMaxSteps(10))
	res, err := r.Run(ctx, Task{Prompt: "x"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Status != Aborted {
		t.Errorf("status = %v, want Aborted", res.Status)
	}
	if disp.calls != 0 {
		t.Errorf("should not dispatch after cancel")
	}
}

// R6: a blocked/failed tool becomes an observation and the loop continues.
func TestBlockedToolContinues(t *testing.T) {
	disp := &fakeDispatcher{err: &blockedErr{}}
	llmc := &fakeLLM{streams: []*fakeStream{toolStream("run_command", "c1"), textStream("recovered")}}
	r := NewRunner(llmc, disp, reg(), WithMaxSteps(5))
	res, err := r.Run(context.Background(), Task{Prompt: "x"})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Status != Completed || res.Summary != "recovered" {
		t.Errorf("res = %+v", res)
	}
}

type blockedErr struct{}

func (*blockedErr) Error() string { return "blocked" }

// R7: empty prompt is rejected.
func TestEmptyPromptFails(t *testing.T) {
	r := NewRunner(&fakeLLM{streams: []*fakeStream{textStream("x")}}, &fakeDispatcher{}, reg())
	if _, err := r.Run(context.Background(), Task{Prompt: "  "}); err == nil {
		t.Error("empty prompt should error")
	}
}

// Multi-turn memory: a second Run on the same Runner must include the first
// turn's user prompt and assistant reply in the conversation it sends.
func TestConversationPersistsAcrossRuns(t *testing.T) {
	llmc := &fakeLLM{streams: []*fakeStream{textStream("first answer"), textStream("second answer")}}
	r := NewRunner(llmc, &fakeDispatcher{}, reg())

	if _, err := r.Run(context.Background(), Task{Prompt: "remember X"}); err != nil {
		t.Fatalf("run 1: %v", err)
	}
	if _, err := r.Run(context.Background(), Task{Prompt: "what was X?"}); err != nil {
		t.Fatalf("run 2: %v", err)
	}

	if len(llmc.requests) != 2 {
		t.Fatalf("expected 2 LLM calls, got %d", len(llmc.requests))
	}
	second := llmc.requests[1]
	var sawFirstPrompt, sawFirstAnswer, sawSecondPrompt bool
	for _, m := range second {
		switch m.Content {
		case "remember X":
			sawFirstPrompt = true
		case "first answer":
			sawFirstAnswer = true
		case "what was X?":
			sawSecondPrompt = true
		}
	}
	if !sawFirstPrompt || !sawFirstAnswer || !sawSecondPrompt {
		t.Errorf("second turn missing history: firstPrompt=%v firstAnswer=%v secondPrompt=%v\nmsgs=%+v",
			sawFirstPrompt, sawFirstAnswer, sawSecondPrompt, second)
	}
	// Exactly one system message across the whole conversation.
	sys := 0
	for _, m := range second {
		if m.Role == llm.RoleSystem {
			sys++
		}
	}
	if sys != 1 {
		t.Errorf("expected exactly 1 system message, got %d", sys)
	}
}

// errLLM always fails on Complete (simulates an HTTP 500 after retries).
type errLLM struct{ err error }

func (e *errLLM) Complete(ctx context.Context, req llm.Request) (llm.Stream, error) {
	return nil, e.err
}

// A failed run rolls the conversation back so the next turn starts clean
// (the failed prompt is not resent and can't keep poisoning requests).
func TestFailedRunRollsBackHistory(t *testing.T) {
	r := NewRunner(&errLLM{err: &llm.LLMError{Kind: llm.ErrHTTPStatus, StatusCode: 500, UserMessage: "boom"}},
		&fakeDispatcher{}, reg())

	res, err := r.Run(context.Background(), Task{Prompt: "do X"})
	if err != nil {
		t.Fatalf("run err: %v", err)
	}
	if res.Status != Failed {
		t.Fatalf("status = %v, want Failed", res.Status)
	}
	// Only the system message should remain; the failed user turn is gone.
	if len(r.history) != 1 || r.history[0].Role != llm.RoleSystem {
		t.Errorf("history not rolled back: %+v", r.history)
	}
}

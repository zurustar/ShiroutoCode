package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func writeSSE(w http.ResponseWriter, lines ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	fl, _ := w.(http.Flusher)
	for _, l := range lines {
		io.WriteString(w, l)
		if fl != nil {
			fl.Flush()
		}
	}
}

func textSSELines(parts ...string) []string {
	var lines []string
	for _, p := range parts {
		lines = append(lines, "data: "+mustJSON(streamChunkOut(p, "", "", "", 0, ""))+"\n\n")
	}
	lines = append(lines, "data: "+mustJSON(streamChunkOut("", "", "", "", 0, "stop"))+"\n\n")
	lines = append(lines, "data: [DONE]\n\n")
	return lines
}

func newTestClient(url string, opts ...Option) *Client {
	base := []Option{WithBackoffBase(time.Millisecond), WithIdleTimeout(0)}
	return New(url, "test-model", append(base, opts...)...)
}

// US-2.1 / US-2.2: request reaches endpoint/model and streams text deltas.
func TestCompleteStreamsText(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		writeSSE(w, textSSELines("Hel", "lo", " there")...)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, WithHTTPClient(srv.Client()))
	// Function mode: plain assistant text passes through unchanged.
	s, err := c.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "hi"}}, ToolMode: ToolModeFunction})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	res, err := Collect(s)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if res.Text != "Hello there" {
		t.Errorf("text = %q", res.Text)
	}
	var sent chatRequestJSON
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatalf("bad request body: %v", err)
	}
	if sent.Model != "test-model" || !sent.Stream {
		t.Errorf("model/stream = %q/%v", sent.Model, sent.Stream)
	}
}

// R1: tools sent only in function mode; temperature/max_tokens omitted when nil.
func TestRequestToolsOnlyInFunctionMode(t *testing.T) {
	capture := func(req Request, mode ToolMode) chatRequestJSON {
		var got chatRequestJSON
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &got)
			writeSSE(w, textSSELines("ok")...)
		}))
		defer srv.Close()
		req.ToolMode = mode
		c := newTestClient(srv.URL, WithHTTPClient(srv.Client()))
		s, err := c.Complete(context.Background(), req)
		if err != nil {
			t.Fatalf("complete: %v", err)
		}
		_, _ = Collect(s)
		return got
	}

	req := Request{
		Messages: []Message{{Role: RoleUser, Content: "x"}},
		Tools:    []ToolSpec{{Name: "read_file", Description: "read", Parameters: map[string]any{"type": "object"}}},
	}
	fn := capture(req, ToolModeFunction)
	if len(fn.Tools) != 1 || fn.Tools[0].Function.Name != "read_file" {
		t.Errorf("function mode should send tools, got %+v", fn.Tools)
	}
	if fn.Temperature != nil || fn.MaxTokens != nil {
		t.Errorf("nil params must be omitted")
	}
	js := capture(req, ToolModeJSON)
	if len(js.Tools) != 0 {
		t.Errorf("json mode must not send tools, got %+v", js.Tools)
	}
	// json mode injects a system instruction message
	if js.Messages[0].Role != RoleSystem || !strings.Contains(js.Messages[0].Content, "JSON object") {
		t.Errorf("json mode should inject system prompt, got %+v", js.Messages[0])
	}
}

// R7 / P2: retry on 5xx up to maxRetries, then succeed; count attempts.
func TestRetryOn5xxThenSuccess(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			io.WriteString(w, "unavailable")
			return
		}
		writeSSE(w, textSSELines("ok")...)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, WithHTTPClient(srv.Client()), WithMaxRetries(3))
	s, err := c.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "x"}}, ToolMode: ToolModeJSON})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	_, _ = Collect(s)
	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Errorf("attempts = %d, want 3", got)
	}
}

// R7: non-retryable 4xx fails immediately (single attempt).
func TestNoRetryOn4xx(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, "nope")
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, WithHTTPClient(srv.Client()), WithMaxRetries(3))
	_, err := c.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "x"}}, ToolMode: ToolModeJSON})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Errorf("attempts = %d, want 1 (no retry on 4xx)", got)
	}
}

// P5 / R2: auto mode falls back to JSON when function mode is rejected (400).
func TestAutoFallbackToJSON(t *testing.T) {
	var mu sync.Mutex
	var modes []string // record whether each request carried tools
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var req chatRequestJSON
		_ = json.Unmarshal(b, &req)
		mu.Lock()
		hadTools := len(req.Tools) > 0
		modes = append(modes, fmt.Sprintf("tools=%v", hadTools))
		mu.Unlock()
		if hadTools {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, "tools not supported by this model")
			return
		}
		writeSSE(w, textSSELines(`{"final":"done"}`)...)
	}))
	defer srv.Close()

	c := newTestClient(srv.URL, WithHTTPClient(srv.Client()))
	req := Request{
		Messages: []Message{{Role: RoleUser, Content: "x"}},
		Tools:    []ToolSpec{{Name: "t"}},
		ToolMode: ToolModeAuto,
	}
	s, err := c.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if s.Mode() != ToolModeJSON {
		t.Errorf("resolved mode = %v, want json after fallback", s.Mode())
	}
	res, err := Collect(s)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if res.Text != "done" {
		t.Errorf("final text = %q", res.Text)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(modes) != 2 || modes[0] != "tools=true" || modes[1] != "tools=false" {
		t.Errorf("expected function-then-json, got %v", modes)
	}
}

// US-6.1: connection failure surfaces an Unreachable LLMError.
func TestUnreachableError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // now nothing is listening

	c := newTestClient(url, WithMaxRetries(0))
	_, err := c.Complete(context.Background(), Request{Messages: []Message{{Role: RoleUser, Content: "x"}}, ToolMode: ToolModeJSON})
	var le *LLMError
	if err == nil {
		t.Fatal("expected error")
	}
	if asLLM(err, &le); le == nil || le.Kind != ErrUnreachable {
		t.Errorf("want Unreachable, got %v", err)
	}
}

// R2: ctx cancellation aborts a stalled request without hanging.
func TestContextCancelAborts(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block // never respond until test ends
	}))
	defer srv.Close()
	defer close(block)

	c := newTestClient(srv.URL, WithHTTPClient(srv.Client()), WithMaxRetries(0))
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()

	done := make(chan error, 1)
	go func() {
		_, err := c.Complete(ctx, Request{Messages: []Message{{Role: RoleUser, Content: "x"}}, ToolMode: ToolModeJSON})
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Error("expected error on cancel")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Complete did not return after cancel")
	}
}

func asLLM(err error, target **LLMError) {
	for err != nil {
		if le, ok := err.(*LLMError); ok {
			*target = le
			return
		}
		type unwrapper interface{ Unwrap() error }
		if u, ok := err.(unwrapper); ok {
			err = u.Unwrap()
		} else {
			return
		}
	}
}

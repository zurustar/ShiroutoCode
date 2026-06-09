package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"sort"
	"strings"
	"time"
)

// sseReader reads Server-Sent Events from a stream, returning the data payload
// of each event. Comment lines (":") and blank separators are handled; multiple
// "data:" lines within one event are joined with newlines (NFR design P3).
type sseReader struct {
	sc *bufio.Scanner
}

func newSSEReader(r io.Reader) *sseReader {
	sc := bufio.NewScanner(r)
	// Allow long lines (model outputs can be large): up to 4 MiB.
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	return &sseReader{sc: sc}
}

// next returns the next event's data, done=true at "[DONE]", or io.EOF.
func (s *sseReader) next() (data string, done bool, err error) {
	var dataLines []string
	flush := func() (string, bool, bool) {
		if len(dataLines) == 0 {
			return "", false, false
		}
		joined := strings.Join(dataLines, "\n")
		if strings.TrimSpace(joined) == "[DONE]" {
			return "", true, true
		}
		return joined, false, true
	}

	for s.sc.Scan() {
		line := s.sc.Text()
		if line == "" {
			if d, done, ok := flush(); ok {
				return d, done, nil
			}
			continue // skip leading/duplicate blank lines
		}
		if strings.HasPrefix(line, ":") {
			continue // comment
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
			continue
		}
		// Other SSE fields (event:, id:, retry:) are not used here.
	}
	if err := s.sc.Err(); err != nil {
		return "", false, err
	}
	if d, done, ok := flush(); ok { // event not terminated by a blank line
		return d, done, nil
	}
	return "", false, io.EOF
}

// streamChunkJSON mirrors the OpenAI streaming chunk shape we consume.
type streamChunkJSON struct {
	Choices []struct {
		Delta struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// parseStreamData converts one SSE data payload into domain chunks plus an
// optional finish reason.
func parseStreamData(data string) ([]Chunk, string, error) {
	var sc streamChunkJSON
	if err := json.Unmarshal([]byte(data), &sc); err != nil {
		return nil, "", err
	}
	var chunks []Chunk
	finish := ""
	for _, ch := range sc.Choices {
		if ch.Delta.Content != "" {
			chunks = append(chunks, Chunk{Kind: ChunkText, Text: ch.Delta.Content})
		}
		for _, tc := range ch.Delta.ToolCalls {
			chunks = append(chunks, Chunk{Kind: ChunkToolCall, ToolCallDelta: &ToolCallDelta{
				Index: tc.Index, ID: tc.ID, Name: tc.Function.Name, ArgsFragment: tc.Function.Arguments,
			}})
		}
		if ch.FinishReason != nil && *ch.FinishReason != "" {
			finish = *ch.FinishReason
		}
	}
	return chunks, finish, nil
}

// assembleToolCalls concatenates streamed tool-call fragments (by index) into
// complete ToolCalls, parsing accumulated arguments JSON (Functional R5).
func assembleToolCalls(deltas []ToolCallDelta) ([]ToolCall, error) {
	calls := map[int]*ToolCall{}
	rawArgs := map[int]*strings.Builder{}
	var order []int
	for _, d := range deltas {
		tc, ok := calls[d.Index]
		if !ok {
			tc = &ToolCall{}
			calls[d.Index] = tc
			rawArgs[d.Index] = &strings.Builder{}
			order = append(order, d.Index)
		}
		if d.ID != "" {
			tc.ID = d.ID
		}
		if d.Name != "" {
			tc.Name = d.Name
		}
		rawArgs[d.Index].WriteString(d.ArgsFragment)
	}
	sort.Ints(order)
	var out []ToolCall
	for _, i := range order {
		tc := calls[i]
		raw := rawArgs[i].String()
		if strings.TrimSpace(raw) != "" {
			var args map[string]any
			if err := json.Unmarshal([]byte(raw), &args); err != nil {
				return nil, newDecodeError(err)
			}
			tc.Args = args
		}
		out = append(out, *tc)
	}
	return out, nil
}

// streamImpl adapts an SSE body into a Stream of domain Chunks with an optional
// idle timeout (NFR design P1/P3). Reading happens in a goroutine so Recv can
// race the idle timer and context cancellation.
type streamImpl struct {
	ctx    context.Context
	closer io.Closer
	events chan sseEvent
	idle   time.Duration
	mode   ToolMode

	pending []Chunk
	finish  string
	done    bool
}

type sseEvent struct {
	data string
	done bool
	err  error
}

func newStream(ctx context.Context, body io.ReadCloser, mode ToolMode, idle time.Duration) *streamImpl {
	s := &streamImpl{ctx: ctx, closer: body, events: make(chan sseEvent, 16), idle: idle, mode: mode}
	go s.readLoop(body)
	return s
}

func (s *streamImpl) readLoop(body io.Reader) {
	defer close(s.events)
	r := newSSEReader(body)
	for {
		data, done, err := r.next()
		select {
		case s.events <- sseEvent{data: data, done: done, err: err}:
		case <-s.ctx.Done():
			return
		}
		if err != nil || done {
			return
		}
	}
}

func (s *streamImpl) Mode() ToolMode { return s.mode }

func (s *streamImpl) Close() error {
	if s.closer != nil {
		return s.closer.Close()
	}
	return nil
}

func (s *streamImpl) Recv() (Chunk, error) {
	for len(s.pending) == 0 {
		if s.done {
			return Chunk{}, io.EOF
		}
		var idleC <-chan time.Time
		if s.idle > 0 {
			t := time.NewTimer(s.idle)
			defer t.Stop()
			idleC = t.C
		}
		select {
		case <-s.ctx.Done():
			return Chunk{}, classifyCtx(s.ctx.Err())
		case <-idleC:
			return Chunk{}, &LLMError{Kind: ErrTimeout,
				UserMessage: "応答が途中で止まりました（アイドルタイムアウト）。モデルの状態を確認してください。", Retryable: false}
		case ev, ok := <-s.events:
			if !ok {
				s.done = true
				return Chunk{Kind: ChunkDone, FinishReason: orDefault(s.finish, "stop")}, nil
			}
			if ev.err != nil {
				if ev.err == io.EOF {
					s.done = true
					return Chunk{Kind: ChunkDone, FinishReason: orDefault(s.finish, "stop")}, nil
				}
				return Chunk{}, newBadStream(ev.err)
			}
			if ev.done {
				s.done = true
				return Chunk{Kind: ChunkDone, FinishReason: orDefault(s.finish, "stop")}, nil
			}
			chunks, fr, perr := parseStreamData(ev.data)
			if perr != nil {
				return Chunk{}, newBadStream(perr)
			}
			if fr != "" {
				s.finish = fr
			}
			s.pending = append(s.pending, chunks...)
		}
	}
	c := s.pending[0]
	s.pending = s.pending[1:]
	return c, nil
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// Collect consumes a Stream fully into a CompletionResult, assembling tool
// calls and, in JSON mode, parsing the single-JSON fallback protocol.
func Collect(s Stream) (CompletionResult, error) {
	var res CompletionResult
	var text strings.Builder
	var deltas []ToolCallDelta
	for {
		c, err := s.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return res, err
		}
		switch c.Kind {
		case ChunkText:
			text.WriteString(c.Text)
		case ChunkToolCall:
			if c.ToolCallDelta != nil {
				deltas = append(deltas, *c.ToolCallDelta)
			}
		case ChunkDone:
			res.FinishReason = c.FinishReason
		}
	}
	res.Text = text.String()
	if len(deltas) > 0 {
		calls, err := assembleToolCalls(deltas)
		if err != nil {
			return res, err
		}
		res.ToolCalls = calls
	}
	if s.Mode() == ToolModeJSON && len(res.ToolCalls) == 0 {
		call, final, err := parseJSONTool(res.Text)
		if err != nil {
			return res, err
		}
		if call != nil {
			res.ToolCalls = []ToolCall{*call}
			res.Text = ""
		} else {
			res.Text = final
		}
	}
	return res, nil
}

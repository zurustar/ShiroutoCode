package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// buildSSE renders text deltas as an OpenAI-style SSE body, terminated by
// finish_reason and [DONE].
func buildTextSSE(deltas []string) string {
	var b strings.Builder
	for _, d := range deltas {
		payload, _ := json.Marshal(streamChunkOut(d, "", "", "", 0, ""))
		fmt.Fprintf(&b, "data: %s\n\n", payload)
	}
	fin, _ := json.Marshal(streamChunkOut("", "", "", "", 0, "stop"))
	fmt.Fprintf(&b, "data: %s\n\n", fin)
	b.WriteString("data: [DONE]\n\n")
	return b.String()
}

// streamChunkOut builds a minimal OpenAI streaming chunk for tests.
func streamChunkOut(content, tcID, tcName, tcArgs string, tcIndex int, finish string) map[string]any {
	delta := map[string]any{}
	if content != "" {
		delta["content"] = content
	}
	if tcID != "" || tcName != "" || tcArgs != "" {
		fn := map[string]any{}
		if tcName != "" {
			fn["name"] = tcName
		}
		fn["arguments"] = tcArgs
		delta["tool_calls"] = []any{map[string]any{"index": tcIndex, "id": tcID, "function": fn}}
	}
	choice := map[string]any{"index": 0, "delta": delta}
	if finish != "" {
		choice["finish_reason"] = finish
	} else {
		choice["finish_reason"] = nil
	}
	return map[string]any{"choices": []any{choice}}
}

func collectText(t *testing.T, s Stream) string {
	t.Helper()
	res, err := Collect(s)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	return res.Text
}

// R4 (PBT): splitting text into arbitrary SSE deltas reconstructs the original.
func TestStreamTextReconstructionPBT(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		full := rapid.String().Draw(rt, "text")
		// split into random pieces
		pieces := splitString(rt, full)
		body := buildTextSSE(pieces)
		s := newStream(context.Background(), io.NopCloser(strings.NewReader(body)), ToolModeFunction, 0)
		got := collectText(t, s)
		if got != full {
			rt.Fatalf("reconstructed %q != original %q", got, full)
		}
	})
}

func splitString(rt *rapid.T, s string) []string {
	if s == "" {
		return nil
	}
	runes := []rune(s)
	var out []string
	i := 0
	for i < len(runes) {
		n := rapid.IntRange(1, len(runes)-i).Draw(rt, "chunklen")
		out = append(out, string(runes[i:i+n]))
		i += n
	}
	return out
}

// R5 (PBT): tool-call argument fragments reassemble into the original args.
func TestToolCallAssemblyPBT(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		keys := rapid.SliceOfDistinct(rapid.StringMatching(`[a-z]{1,6}`), func(s string) string { return s }).Draw(rt, "keys")
		argsObj := map[string]string{}
		for _, k := range keys {
			argsObj[k] = rapid.StringMatching(`[a-zA-Z0-9 ]{0,8}`).Draw(rt, "v_"+k)
		}
		argsJSON, _ := json.Marshal(argsObj)
		frags := splitString(rt, string(argsJSON))

		var deltas []ToolCallDelta
		for i, f := range frags {
			d := ToolCallDelta{Index: 0, ArgsFragment: f}
			if i == 0 {
				d.ID = "call_1"
				d.Name = "read_file"
			}
			deltas = append(deltas, d)
		}
		calls, err := assembleToolCalls(deltas)
		if err != nil {
			rt.Fatalf("assemble: %v", err)
		}
		if len(calls) != 1 {
			rt.Fatalf("want 1 call, got %d", len(calls))
		}
		c := calls[0]
		if c.ID != "call_1" || c.Name != "read_file" {
			rt.Fatalf("bad id/name: %+v", c)
		}
		for k, v := range argsObj {
			if got, _ := c.Args[k].(string); got != v {
				rt.Fatalf("arg %q = %q, want %q", k, got, v)
			}
		}
	})
}

func TestStreamCommentsBlanksAndDone(t *testing.T) {
	body := ": this is a comment\n\n" +
		"data: " + mustJSON(streamChunkOut("Hello", "", "", "", 0, "")) + "\n\n" +
		"\n" + // stray blank
		"data: " + mustJSON(streamChunkOut(" world", "", "", "", 0, "stop")) + "\n\n" +
		"data: [DONE]\n\n"
	s := newStream(context.Background(), io.NopCloser(strings.NewReader(body)), ToolModeFunction, 0)
	res, err := Collect(s)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if res.Text != "Hello world" {
		t.Errorf("text = %q", res.Text)
	}
	if res.FinishReason != "stop" {
		t.Errorf("finish = %q", res.FinishReason)
	}
}

func TestStreamBadJSONIsBadStream(t *testing.T) {
	body := "data: {not json}\n\n"
	s := newStream(context.Background(), io.NopCloser(strings.NewReader(body)), ToolModeFunction, 0)
	_, err := s.Recv()
	var le *LLMError
	if !errors.As(err, &le) || le.Kind != ErrBadStream {
		t.Fatalf("want BadStream, got %v", err)
	}
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

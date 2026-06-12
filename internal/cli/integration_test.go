package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/zurustar/shiroutocode/internal/config"
	"github.com/zurustar/shiroutocode/internal/guardrail"
	"github.com/zurustar/shiroutocode/internal/llm"
	"github.com/zurustar/shiroutocode/internal/log"
	"github.com/zurustar/shiroutocode/internal/tools"
)

type fakeStream struct {
	chunks []llm.Chunk
	i      int
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
func (s *fakeStream) Mode() llm.ToolMode { return llm.ToolModeFunction }

type fakeClient struct {
	chunks  []llm.Chunk
	err     error
	model   string
	models  []string
	listErr error
}

func (f *fakeClient) Complete(ctx context.Context, req llm.Request) (llm.Stream, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &fakeStream{chunks: append([]llm.Chunk(nil), f.chunks...)}, nil
}

func (f *fakeClient) Model() string     { return f.model }
func (f *fakeClient) SetModel(m string) { f.model = m }
func (f *fakeClient) ListModels(ctx context.Context) ([]string, error) {
	return f.models, f.listErr
}

func testCore(t *testing.T, client llmClient) *Core {
	t.Helper()
	return &Core{
		cfg:    config.Config{MaxSteps: 5, Workspace: t.TempDir()},
		logger: log.New("error", log.FormatText, io.Discard),
		client: client,
		reg:    tools.NewRegistry(),
		policy: guardrail.Policy{WorkspaceRoot: t.TempDir()},
	}
}

// Single-shot completion prints summary and exits 0.
func TestSingleShotCompletes(t *testing.T) {
	client := &fakeClient{chunks: []llm.Chunk{
		{Kind: llm.ChunkText, Text: "done thing"},
		{Kind: llm.ChunkDone, FinishReason: "stop"},
	}}
	var out, errb bytes.Buffer
	code := runSingleShot(context.Background(), testCore(t, client), "do it", &out, &errb, strings.NewReader(""), false)
	if code != exitOK {
		t.Errorf("exit = %d, want 0; stderr=%s", code, errb.String())
	}
	if !strings.Contains(out.String(), "done thing") || !strings.Contains(out.String(), "完了") {
		t.Errorf("stdout missing summary:\n%s", out.String())
	}
}

// US-6.1: connection failure surfaces guidance and a non-zero exit code.
func TestSingleShotConnectionError(t *testing.T) {
	client := &fakeClient{err: &llm.LLMError{
		Kind:        llm.ErrUnreachable,
		UserMessage: "LM Studio に接続できません。起動状態と Endpoint を確認してください。",
	}}
	var out, errb bytes.Buffer
	code := runSingleShot(context.Background(), testCore(t, client), "do it", &out, &errb, strings.NewReader(""), false)
	if code != exitFailed {
		t.Errorf("exit = %d, want %d", code, exitFailed)
	}
	if !strings.Contains(errb.String(), "接続できません") {
		t.Errorf("connection guidance missing:\n%s", errb.String())
	}
	if strings.Contains(errb.String(), "ErrUnreachable") {
		t.Errorf("internal details leaked:\n%s", errb.String())
	}
}

package guardrail

import (
	"context"
	"errors"
	"testing"

	"pgregory.net/rapid"

	"github.com/zurustar/shiroutocode/internal/tools"
)

// fakeTool records whether it was executed.
type fakeTool struct {
	name     string
	executed *bool
}

func (f *fakeTool) Name() string        { return f.name }
func (f *fakeTool) Description() string { return "fake" }
func (f *fakeTool) Execute(ctx context.Context, args map[string]any) (tools.ToolResult, error) {
	*f.executed = true
	return tools.ToolResult{Output: "ran"}, nil
}

// fakeEvaluator returns a fixed decision.
type fakeEvaluator struct{ d Decision }

func (e fakeEvaluator) Evaluate(a Action) Decision { return e.d }

// fakeConfirmer returns a fixed (ok, err).
type fakeConfirmer struct {
	ok  bool
	err error
}

func (c fakeConfirmer) Confirm(ctx context.Context, a Action, reason string) (bool, error) {
	return c.ok, c.err
}

// R2/R8 (PBT): execution happens iff Allow, or Confirm with an interactive
// confirmer returning yes.
func TestDispatchExecutionInvariantPBT(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		decKind := rapid.SampledFrom([]DecisionKind{Allow, Confirm, Deny}).Draw(rt, "dec")
		interactive := rapid.Bool().Draw(rt, "interactive")
		confirmYes := rapid.Bool().Draw(rt, "yes")

		executed := false
		reg := tools.NewRegistry()
		reg.Register(&fakeTool{name: "t", executed: &executed})

		var confirmer Confirmer
		if interactive {
			confirmer = fakeConfirmer{ok: confirmYes}
		} // else nil (non-interactive)

		d := NewToolDispatcher(reg, fakeEvaluator{Decision{Kind: decKind}}, confirmer, nil)
		_, _ = d.Dispatch(context.Background(), tools.ToolCall{Name: "t"})

		want := decKind == Allow || (decKind == Confirm && interactive && confirmYes)
		if executed != want {
			rt.Fatalf("dec=%v interactive=%v yes=%v: executed=%v want=%v",
				decKind, interactive, confirmYes, executed, want)
		}
	})
}

func TestDispatchConfirmerErrorFailsClosed(t *testing.T) {
	executed := false
	reg := tools.NewRegistry()
	reg.Register(&fakeTool{name: "t", executed: &executed})
	d := NewToolDispatcher(reg, fakeEvaluator{Decision{Kind: Confirm}}, fakeConfirmer{err: errors.New("io")}, nil)
	_, err := d.Dispatch(context.Background(), tools.ToolCall{Name: "t"})
	if executed {
		t.Errorf("must not execute when confirmer errors")
	}
	var be *BlockedError
	if !errors.As(err, &be) {
		t.Errorf("expected BlockedError, got %v", err)
	}
}

func TestDispatchDenyReturnsBlocked(t *testing.T) {
	executed := false
	reg := tools.NewRegistry()
	reg.Register(&fakeTool{name: "t", executed: &executed})
	d := NewToolDispatcher(reg, fakeEvaluator{Decision{Kind: Deny, Reason: "nope"}}, fakeConfirmer{ok: true}, nil)
	_, err := d.Dispatch(context.Background(), tools.ToolCall{Name: "t"})
	if executed {
		t.Errorf("deny must not execute")
	}
	var be *BlockedError
	if !errors.As(err, &be) {
		t.Fatalf("expected BlockedError, got %v", err)
	}
}

// Integration: real evaluator denies workspace escape via the dispatcher.
func TestDispatchRealEvaluatorScopeDeny(t *testing.T) {
	root := t.TempDir()
	executed := false
	reg := tools.NewRegistry()
	reg.Register(&fakeTool{name: "write_file", executed: &executed})
	ev := NewEvaluator(Policy{WorkspaceRoot: root})
	d := NewToolDispatcher(reg, ev, fakeConfirmer{ok: true}, nil)

	_, err := d.Dispatch(context.Background(), tools.ToolCall{
		Name: "write_file",
		Args: map[string]any{"path": "../escape.txt", "mode": "overwrite", "content": "x"},
	})
	if executed {
		t.Errorf("workspace escape must not execute")
	}
	var be *BlockedError
	if !errors.As(err, &be) {
		t.Errorf("expected BlockedError, got %v", err)
	}
}

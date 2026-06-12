package cli

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/zurustar/shiroutocode/internal/agent"
	"github.com/zurustar/shiroutocode/internal/llm"
)

func newTestModel() *tuiModel {
	m := newModel(context.Background(), nil, make(chan tea.Msg, 16))
	// give it a size so the viewport is ready
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return mm.(*tuiModel)
}

func TestTUIAppendsAssistantAndStep(t *testing.T) {
	m := newTestModel()
	mm, _ := m.Update(assistantTextMsg("hello "))
	m = mm.(*tuiModel)
	mm, _ = m.Update(assistantTextMsg("world"))
	m = mm.(*tuiModel)
	mm, _ = m.Update(stepMsg{cur: 2, max: 5})
	m = mm.(*tuiModel)
	h := m.history.String()
	if !strings.Contains(h, "hello world") || !strings.Contains(h, "step 2/5") {
		t.Errorf("history = %q", h)
	}
}

func TestTUIConfirmFlow(t *testing.T) {
	m := newTestModel()
	reply := make(chan bool, 1)
	mm, _ := m.Update(confirmReqMsg{reason: "危険", tool: "run_command", reply: reply})
	m = mm.(*tuiModel)
	if !m.confirming {
		t.Fatal("should be confirming")
	}
	// press 'y'
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	m = mm.(*tuiModel)
	if m.confirming {
		t.Error("should have left confirming state")
	}
	select {
	case ok := <-reply:
		if !ok {
			t.Error("expected approval true")
		}
	default:
		t.Error("reply not sent")
	}
}

func TestTUIConfirmDeny(t *testing.T) {
	m := newTestModel()
	reply := make(chan bool, 1)
	mm, _ := m.Update(confirmReqMsg{reason: "危険", tool: "x", reply: reply})
	m = mm.(*tuiModel)
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	m = mm.(*tuiModel)
	if got := <-reply; got {
		t.Error("expected denial false")
	}
}

func TestTUIDoneSummary(t *testing.T) {
	m := newTestModel()
	mm, _ := m.Update(doneMsg{res: agent.Result{Status: agent.Completed, Steps: 3, ChangedFiles: []string{"a.go"}}})
	m = mm.(*tuiModel)
	if m.running {
		t.Error("should not be running after done")
	}
	if !strings.Contains(m.history.String(), "完了") || !strings.Contains(m.history.String(), "a.go") {
		t.Errorf("summary missing: %q", m.history.String())
	}
}

func TestTUICtrlCQuitsWhenIdle(t *testing.T) {
	m := newTestModel()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected quit command")
	}
	// tea.Quit is a func returning tea.QuitMsg
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("ctrl+c when idle should quit")
	}
}

// modelsMsg opens the picker; navigating and pressing Enter selects a model.
func TestTUIModelSelectFlow(t *testing.T) {
	core := testCore(t, &fakeClient{models: []string{"a", "b", "c"}})
	m := newModel(context.Background(), core, make(chan tea.Msg, 16))
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = mm.(*tuiModel)

	mm, _ = m.Update(modelsMsg{models: []string{"a", "b", "c"}})
	m = mm.(*tuiModel)
	if !m.selecting {
		t.Fatal("expected picker to open on modelsMsg")
	}
	if !strings.Contains(m.View(), "モデルを選択") {
		t.Errorf("picker view not shown:\n%s", m.View())
	}

	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = mm.(*tuiModel)
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(*tuiModel)
	if m.selecting {
		t.Error("should exit selecting after Enter")
	}
	if core.Model() != "b" {
		t.Errorf("selected model = %q, want b", core.Model())
	}
}

// /model re-opens the picker; a prompt before any model is selected is refused.
func TestTUIModelCommandAndGuard(t *testing.T) {
	core := testCore(t, &fakeClient{models: []string{"x"}})
	m := newModel(context.Background(), core, make(chan tea.Msg, 16))
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = mm.(*tuiModel)

	// No model yet: a prompt is refused with guidance.
	m.ti.SetValue("do something")
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(*tuiModel)
	if m.running {
		t.Error("must not run without a model")
	}
	if !strings.Contains(m.history.String(), "先にモデルを選択") {
		t.Errorf("missing model guidance:\n%s", m.history.String())
	}

	// /model returns a fetch command (re-opens the picker).
	m.ti.SetValue("/model")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("/model should trigger a fetch command")
	}
}

// A fetch error leaves the picker closed and surfaces guidance.
func TestTUIModelFetchError(t *testing.T) {
	core := testCore(t, &fakeClient{})
	m := newModel(context.Background(), core, make(chan tea.Msg, 16))
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = mm.(*tuiModel)

	mm, _ = m.Update(modelsMsg{err: &llm.LLMError{Kind: llm.ErrUnreachable, UserMessage: "接続できません"}})
	m = mm.(*tuiModel)
	if m.selecting {
		t.Error("picker should stay closed on fetch error")
	}
	if !strings.Contains(m.history.String(), "接続できません") {
		t.Errorf("fetch error not surfaced:\n%s", m.history.String())
	}
}

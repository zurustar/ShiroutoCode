package cli

import (
	"context"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/zurustar/shiroutocode/internal/agent"
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

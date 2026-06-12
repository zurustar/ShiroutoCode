package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/zurustar/shiroutocode/internal/agent"
	"github.com/zurustar/shiroutocode/internal/guardrail"
)

// --- agent -> TUI event messages ---

type assistantTextMsg string
type toolCallMsg struct{ name string }
type toolResultMsg struct {
	name, output string
	err          error
}
type stepMsg struct{ cur, max int }
type confirmReqMsg struct {
	reason, tool string
	reply        chan bool
}
type doneMsg struct{ res agent.Result }

// teaFrontend forwards agent events to the TUI via a channel.
type teaFrontend struct{ ch chan tea.Msg }

func (f teaFrontend) OnAssistantText(s string)                 { f.ch <- assistantTextMsg(s) }
func (f teaFrontend) OnToolCall(name string, _ map[string]any) { f.ch <- toolCallMsg{name} }
func (f teaFrontend) OnToolResult(name, out string, err error) { f.ch <- toolResultMsg{name, out, err} }
func (f teaFrontend) OnStep(cur, max int)                      { f.ch <- stepMsg{cur, max} }

// teaConfirmer asks the TUI for confirmation and blocks until the user answers.
type teaConfirmer struct{ ch chan tea.Msg }

func (c teaConfirmer) Confirm(ctx context.Context, a guardrail.Action, reason string) (bool, error) {
	reply := make(chan bool, 1)
	c.ch <- confirmReqMsg{reason: reason, tool: a.Tool, reply: reply}
	select {
	case ok := <-reply:
		return ok, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

// tuiModel is the bubbletea REPL model.
type tuiModel struct {
	core      *Core
	parentCtx context.Context

	ti viewportInput
	vp viewport.Model
	ch chan tea.Msg

	history strings.Builder
	ready   bool

	running    bool
	cancel     context.CancelFunc
	confirming bool
	reply      chan bool

	selecting bool   // model picker is open (startup or /model)
	pk        picker // picker state while selecting
}

// modelsMsg carries the result of an async model-list fetch.
type modelsMsg struct {
	models []string
	err    error
}

func fetchModelsCmd(ctx context.Context, lister modelLister) tea.Cmd {
	return func() tea.Msg {
		ms, err := lister.ListModels(ctx)
		return modelsMsg{models: ms, err: err}
	}
}

// viewportInput aliases textinput.Model (kept separate for clarity).
type viewportInput = textinput.Model

func newModel(ctx context.Context, core *Core, ch chan tea.Msg) *tuiModel {
	ti := textinput.New()
	ti.Placeholder = "指示を入力（Enterで送信 / Ctrl+Cで中断・終了）"
	ti.Focus()
	ti.CharLimit = 0
	return &tuiModel{core: core, parentCtx: ctx, ti: ti, ch: ch}
}

func waitForEvent(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg { return <-ch }
}

func (m *tuiModel) Init() tea.Cmd {
	// Present the model picker at startup by fetching the server's model list.
	return tea.Batch(textinput.Blink, waitForEvent(m.ch), fetchModelsCmd(m.parentCtx, m.core))
}

func (m *tuiModel) appendLine(s string) {
	m.history.WriteString(s)
	if !strings.HasSuffix(s, "\n") {
		m.history.WriteString("\n")
	}
	m.vp.SetContent(m.history.String())
	m.vp.GotoBottom()
}

func (m *tuiModel) appendText(s string) {
	m.history.WriteString(s)
	m.vp.SetContent(m.history.String())
	m.vp.GotoBottom()
}

func (m *tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.ready {
			m.vp = viewport.New(msg.Width, msg.Height-2)
			m.ready = true
		} else {
			m.vp.Width = msg.Width
			m.vp.Height = msg.Height - 2
		}
		m.ti.Width = msg.Width - 4
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case modelsMsg:
		if msg.err != nil {
			m.appendLine("\n⚠ モデル一覧を取得できません: " + modelSelectError(msg.err))
			m.appendLine("LM Studio を起動後、/model で再試行できます。")
			return m, nil
		}
		if len(msg.models) == 0 {
			m.appendLine("\n⚠ 利用可能なモデルがありません。LM Studio でモデルをロードしてください。")
			return m, nil
		}
		m.pk = newPicker(msg.models, m.core.Model())
		m.selecting = true
		return m, nil

	case assistantTextMsg:
		m.appendText(string(msg))
		return m, waitForEvent(m.ch)
	case toolCallMsg:
		m.appendLine("\n🔧 " + msg.name)
		return m, waitForEvent(m.ch)
	case toolResultMsg:
		if msg.err != nil {
			m.appendLine("   ⚠ " + msg.err.Error())
		} else {
			m.appendLine("   → " + truncate(strings.TrimSpace(msg.output), 500))
		}
		return m, waitForEvent(m.ch)
	case stepMsg:
		m.appendLine(fmt.Sprintf("— step %d/%d —", msg.cur, msg.max))
		return m, waitForEvent(m.ch)
	case confirmReqMsg:
		m.confirming = true
		m.reply = msg.reply
		m.appendLine(fmt.Sprintf("\n⚠ 確認: %s（ツール: %s） [y/N]", msg.reason, msg.tool))
		return m, waitForEvent(m.ch)
	case doneMsg:
		m.running = false
		m.cancel = nil
		m.appendLine("\n" + doneSummary(msg.res))
		return m, waitForEvent(m.ch)
	}
	return m, nil
}

func (m *tuiModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.selecting {
		return m.handleSelectKey(msg)
	}
	if m.confirming {
		ok := parseYes(msg.String())
		m.reply <- ok
		m.confirming = false
		m.reply = nil
		if ok {
			m.appendLine("→ 承認")
		} else {
			m.appendLine("→ 拒否")
		}
		return m, nil
	}

	switch msg.Type {
	case tea.KeyCtrlC:
		if m.running && m.cancel != nil {
			m.cancel()
			m.appendLine("\n⏹ 中断要求...")
			return m, nil
		}
		return m, tea.Quit
	case tea.KeyEnter:
		if m.running {
			return m, nil
		}
		prompt := strings.TrimSpace(m.ti.Value())
		if prompt == "" {
			return m, nil
		}
		if prompt == "/model" {
			m.ti.Reset()
			m.appendLine("> /model")
			return m, fetchModelsCmd(m.parentCtx, m.core)
		}
		if m.core.Model() == "" {
			m.ti.Reset()
			m.appendLine("> " + prompt)
			m.appendLine("⚠ 先にモデルを選択してください（/model）。")
			return m, nil
		}
		m.ti.Reset()
		m.appendLine("> " + prompt)
		return m, m.startRun(prompt)
	}

	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

// handleSelectKey drives the model picker while it is open.
func (m *tuiModel) handleSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		m.pk.up()
	case tea.KeyDown:
		m.pk.down()
	case tea.KeyEnter:
		chosen := m.pk.selected()
		m.selecting = false
		if chosen != "" {
			m.core.SetModel(chosen)
			m.appendLine("モデル: " + chosen)
		}
		return m, nil
	case tea.KeyCtrlC, tea.KeyEsc:
		return m.cancelSelect()
	case tea.KeyRunes:
		switch msg.String() {
		case "k":
			m.pk.up()
		case "j":
			m.pk.down()
		case "q":
			return m.cancelSelect()
		}
	}
	return m, nil
}

// cancelSelect closes the picker. With no model selected yet there is nothing
// to run, so the program quits; otherwise it keeps the current model.
func (m *tuiModel) cancelSelect() (tea.Model, tea.Cmd) {
	m.selecting = false
	if m.core.Model() == "" {
		return m, tea.Quit
	}
	m.appendLine("（モデル選択を中止）")
	return m, nil
}

func (m *tuiModel) startRun(prompt string) tea.Cmd {
	m.running = true
	ctx, cancel := context.WithCancel(m.parentCtx)
	m.cancel = cancel
	runner := m.core.newRunner(teaFrontend{m.ch}, teaConfirmer{m.ch})
	go func() {
		res, err := runner.Run(ctx, agent.Task{Prompt: prompt})
		if err != nil {
			res = agent.Result{Status: agent.Failed, Err: err}
		}
		m.ch <- doneMsg{res}
	}()
	return nil
}

func (m *tuiModel) View() string {
	if m.selecting {
		return m.pk.view("使用するモデルを選択してください（/model で変更可）:")
	}
	if !m.ready {
		return "起動中..."
	}
	footer := m.ti.View()
	if m.confirming {
		footer = "確認: y / N を入力"
	}
	return m.vp.View() + "\n" + footer
}

func doneSummary(res agent.Result) string {
	switch res.Status {
	case agent.Completed:
		s := fmt.Sprintf("✅ 完了（%dステップ）", res.Steps)
		if len(res.ChangedFiles) > 0 {
			s += "  変更: " + strings.Join(res.ChangedFiles, ", ")
		}
		return s
	case agent.StoppedMaxSteps:
		return "⏹ 最大ステップに到達（未完）"
	case agent.Aborted:
		return "⏹ 中断されました"
	default:
		return "❌ " + failureMessage(res.Err)
	}
}

// runREPL launches the interactive bubbletea program.
func runREPL(ctx context.Context, core *Core, stdout, stderr io.Writer) int {
	ch := make(chan tea.Msg, 64)
	m := newModel(ctx, core, ch)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithContext(ctx))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(stderr, "TUIエラー: %s\n", err)
		return exitFailed
	}
	return exitOK
}

package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// modelLister fetches the available model ids from the LLM server. *Core
// satisfies it; tests can inject a stub.
type modelLister interface {
	ListModels(ctx context.Context) ([]string, error)
}

// picker is the pure selection state shared by the standalone startup picker
// and the in-REPL /model picker. Navigation is bounds-clamped so the cursor is
// always a valid index (or the list is empty).
type picker struct {
	models []string
	cursor int
}

func newPicker(models []string, preselect string) picker {
	p := picker{models: models}
	for i, m := range models {
		if m == preselect {
			p.cursor = i
			break
		}
	}
	return p
}

func (p *picker) up() {
	if p.cursor > 0 {
		p.cursor--
	}
}

func (p *picker) down() {
	if p.cursor < len(p.models)-1 {
		p.cursor++
	}
}

func (p picker) selected() string {
	if len(p.models) == 0 || p.cursor < 0 || p.cursor >= len(p.models) {
		return ""
	}
	return p.models[p.cursor]
}

func (p picker) view(title string) string {
	var b strings.Builder
	b.WriteString(title + "\n\n")
	for i, m := range p.models {
		if i == p.cursor {
			b.WriteString("  > " + m + "\n")
		} else {
			b.WriteString("    " + m + "\n")
		}
	}
	b.WriteString("\n[↑/↓ 移動  Enter 決定  q/Esc 中止]")
	return b.String()
}

// --- standalone bubbletea picker (used by single-shot before a run) ---

type pickerModel struct {
	picker
	title    string
	chosen   string
	canceled bool
}

func (m *pickerModel) Init() tea.Cmd { return nil }

func (m *pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.Type {
		case tea.KeyUp:
			m.up()
		case tea.KeyDown:
			m.down()
		case tea.KeyEnter:
			m.chosen = m.selected()
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.canceled = true
			return m, tea.Quit
		case tea.KeyRunes:
			switch key.String() {
			case "k":
				m.up()
			case "j":
				m.down()
			case "q":
				m.canceled = true
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m *pickerModel) View() string { return m.view(m.title) + "\n" }

// errModelSelectionCanceled is returned when the user aborts the picker.
var errModelSelectionCanceled = fmt.Errorf("モデル選択が中止されました")

// selectModelStandalone fetches the model list and runs a blocking picker.
// Used by single-shot runs on a TTY when no model was configured.
func selectModelStandalone(ctx context.Context, lister modelLister, preselect string, stdout io.Writer) (string, error) {
	models, err := lister.ListModels(ctx)
	if err != nil {
		return "", err
	}
	if len(models) == 0 {
		return "", fmt.Errorf("利用可能なモデルがありません。LM Studio でモデルをロードしてください")
	}
	m := &pickerModel{picker: newPicker(models, preselect), title: "使用するモデルを選択してください:"}
	p := tea.NewProgram(m, tea.WithContext(ctx), tea.WithOutput(stdout))
	if _, err := p.Run(); err != nil {
		return "", err
	}
	if m.canceled {
		return "", errModelSelectionCanceled
	}
	return m.chosen, nil
}

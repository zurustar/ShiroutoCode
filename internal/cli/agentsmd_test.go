package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zurustar/shiroutocode/internal/agent"
	"github.com/zurustar/shiroutocode/internal/config"
	"github.com/zurustar/shiroutocode/internal/log"
)

func TestLoadAgentsDocPresent(t *testing.T) {
	ws := t.TempDir()
	want := "# Project\nUse tabs. Run `make test` before finishing."
	if err := os.WriteFile(filepath.Join(ws, "AGENTS.md"), []byte("\n"+want+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, ok := loadAgentsDoc(ws)
	if !ok {
		t.Fatal("expected AGENTS.md to be found")
	}
	if doc != want {
		t.Errorf("doc = %q, want %q", doc, want)
	}
}

func TestLoadAgentsDocAbsentOrEmpty(t *testing.T) {
	ws := t.TempDir()
	if _, ok := loadAgentsDoc(ws); ok {
		t.Error("absent AGENTS.md should return ok=false")
	}
	if err := os.WriteFile(filepath.Join(ws, "AGENTS.md"), []byte("   \n\t\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := loadAgentsDoc(ws); ok {
		t.Error("blank AGENTS.md should return ok=false")
	}
}

func TestLoadAgentsDocTruncates(t *testing.T) {
	ws := t.TempDir()
	big := strings.Repeat("x", maxAgentsDocBytes+5000)
	if err := os.WriteFile(filepath.Join(ws, "AGENTS.md"), []byte(big), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, ok := loadAgentsDoc(ws)
	if !ok {
		t.Fatal("expected found")
	}
	if len(doc) <= maxAgentsDocBytes || !strings.Contains(doc, "truncated") {
		t.Errorf("expected truncation marker; len=%d", len(doc))
	}
}

func TestComposeSystemPrompt(t *testing.T) {
	base := agent.DefaultSystemPrompt
	if got := composeSystemPrompt(base, ""); got != base {
		t.Error("empty doc should leave the base prompt unchanged")
	}
	got := composeSystemPrompt(base, "Always write Go.")
	if !strings.Contains(got, base) || !strings.Contains(got, "AGENTS.md") || !strings.Contains(got, "Always write Go.") {
		t.Errorf("composed prompt missing parts:\n%s", got)
	}
}

// BuildCore applies AGENTS.md when present.
func TestBuildCoreAppliesAgentsMd(t *testing.T) {
	ws := t.TempDir()
	if err := os.WriteFile(filepath.Join(ws, "AGENTS.md"), []byte("Prefer small PRs."), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{Endpoint: "http://localhost:1234/v1", Model: "m", MaxSteps: 5, Workspace: ws, ToolMode: "auto"}
	core := BuildCore(cfg, log.New("error", log.FormatText, io.Discard))
	if !core.AgentsLoaded() {
		t.Fatal("AgentsLoaded should be true")
	}
	if !strings.Contains(core.system, "Prefer small PRs.") {
		t.Errorf("system prompt missing AGENTS.md content:\n%s", core.system)
	}
}

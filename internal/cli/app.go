// Package cli is the U5 frontend: it wires U1-U4 together and provides the
// interactive TUI (REPL) and plain single-shot interfaces. The agent core is
// front-agnostic; this package supplies the Frontend and Confirmer.
package cli

import (
	"github.com/zurustar/shiroutocode/internal/agent"
	"github.com/zurustar/shiroutocode/internal/config"
	"github.com/zurustar/shiroutocode/internal/guardrail"
	"github.com/zurustar/shiroutocode/internal/llm"
	"github.com/zurustar/shiroutocode/internal/log"
	"github.com/zurustar/shiroutocode/internal/tools"
)

// Core holds the front-agnostic wiring built from config (U1-U3 + LLM).
type Core struct {
	cfg    config.Config
	logger log.Logger
	client llm.LLMClient
	reg    *tools.Registry
	policy guardrail.Policy
}

// BuildCore assembles the LLM client, tool registry and guardrail policy.
func BuildCore(cfg config.Config, logger log.Logger) *Core {
	client := llm.New(cfg.Endpoint, cfg.Model, llm.WithLogger(logger))

	reg := tools.NewRegistry()
	reg.Register(tools.NewReadFileTool(cfg.Workspace))
	reg.Register(tools.NewFileTool(cfg.Workspace))
	reg.Register(tools.NewTerminalTool(cfg.Workspace, 0, nil))
	reg.Register(tools.NewGitTool(cfg.Workspace))
	reg.Register(tools.NewWebTool(0))

	policy := guardrail.Policy{
		WorkspaceRoot:     cfg.Workspace,
		ConfirmMode:       cfg.Guardrail.ConfirmMode,
		ExtraDenyPatterns: cfg.Guardrail.ExtraDenyPatterns,
	}
	return &Core{cfg: cfg, logger: logger, client: client, reg: reg, policy: policy}
}

// newRunner builds an agent Runner bound to the given frontend and confirmer.
// A nil confirmer means non-interactive (the guardrail will block Confirm
// actions — fail-closed).
func (c *Core) newRunner(fe agent.Frontend, confirmer guardrail.Confirmer) *agent.Runner {
	ev := guardrail.NewEvaluator(c.policy)
	disp := guardrail.NewToolDispatcher(c.reg, ev, confirmer, c.logger)
	return agent.NewRunner(c.client, disp, c.reg,
		agent.WithFrontend(fe),
		agent.WithLogger(c.logger),
		agent.WithMaxSteps(c.cfg.MaxSteps),
		agent.WithToolMode(llm.ToolModeAuto),
	)
}

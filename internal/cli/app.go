// Package cli is the U5 frontend: it wires U1-U4 together and provides the
// interactive TUI (REPL) and plain single-shot interfaces. The agent core is
// front-agnostic; this package supplies the Frontend and Confirmer.
package cli

import (
	"net"
	"net/url"

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
	warnInsecureEndpoint(cfg.Endpoint, logger)

	// Surface deny patterns that will not compile rather than letting them
	// silently fail open (F-06).
	if bad := guardrail.InvalidDenyPatterns(cfg.Guardrail.ExtraDenyPatterns); len(bad) > 0 {
		logger.Warn("guardrail: ignoring invalid deny patterns (no protection from these)", "patterns", bad)
	}

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

// warnInsecureEndpoint warns when the LLM endpoint is cleartext http to a
// non-loopback host, since prompts and file contents would travel unencrypted
// (F-08). Loopback/localhost (the default) is silent.
func warnInsecureEndpoint(endpoint string, logger log.Logger) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "http" {
		return
	}
	host := u.Hostname()
	if host == "localhost" {
		return
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return
	}
	logger.Warn("llm endpoint uses cleartext http to a non-loopback host; prompts and file contents are sent unencrypted — prefer https", "endpoint", endpoint)
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
		agent.WithToolMode(llm.ToolMode(c.cfg.ToolMode)),
	)
}

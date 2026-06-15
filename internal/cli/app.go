// Package cli is the U5 frontend: it wires U1-U4 together and provides the
// interactive TUI (REPL) and plain single-shot interfaces. The agent core is
// front-agnostic; this package supplies the Frontend and Confirmer.
package cli

import (
	"context"
	"net"
	"net/url"

	"github.com/zurustar/shiroutocode/internal/agent"
	"github.com/zurustar/shiroutocode/internal/config"
	"github.com/zurustar/shiroutocode/internal/guardrail"
	"github.com/zurustar/shiroutocode/internal/llm"
	"github.com/zurustar/shiroutocode/internal/log"
	"github.com/zurustar/shiroutocode/internal/tools"
)

// llmClient is the LLM surface the CLI needs: completions (for the agent) plus
// model management (for the interactive picker and the /model command).
// *llm.Client satisfies it.
type llmClient interface {
	llm.LLMClient
	Model() string
	SetModel(string)
	ListModels(ctx context.Context) ([]string, error)
}

// Core holds the front-agnostic wiring built from config (U1-U3 + LLM).
type Core struct {
	cfg          config.Config
	logger       log.Logger
	client       llmClient
	reg          *tools.Registry
	policy       guardrail.Policy
	system       string // composed system prompt (default + AGENTS.md if present)
	agentsLoaded bool   // AGENTS.md was found and applied
}

// AgentsLoaded reports whether project instructions from AGENTS.md are applied.
func (c *Core) AgentsLoaded() bool { return c.agentsLoaded }

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

	// Follow the agents.md convention: a workspace-root AGENTS.md provides
	// project-specific instructions that are folded into the system prompt.
	doc, ok := loadAgentsDoc(cfg.Workspace)
	if ok {
		logger.Debug("loaded project instructions", "file", agentsFileName)
	}
	system := composeSystemPrompt(agent.DefaultSystemPrompt, doc)

	return &Core{cfg: cfg, logger: logger, client: client, reg: reg, policy: policy, system: system, agentsLoaded: ok}
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

// Model reports the model the core currently targets ("" if unselected).
func (c *Core) Model() string { return c.client.Model() }

// SetModel switches the target model (interactive picker / REPL /model).
func (c *Core) SetModel(m string) { c.client.SetModel(m) }

// ListModels returns the model ids the server exposes (for the picker).
func (c *Core) ListModels(ctx context.Context) ([]string, error) {
	return c.client.ListModels(ctx)
}

// newRunner builds an agent Runner bound to the given frontend and confirmer.
// A nil confirmer means non-interactive (the guardrail will block Confirm
// actions — fail-closed).
func (c *Core) newRunner(fe agent.Frontend, confirmer guardrail.Confirmer) *agent.Runner {
	ev := guardrail.NewEvaluator(c.policy)
	disp := guardrail.NewToolDispatcher(c.reg, ev, confirmer, c.logger)
	opts := []agent.Option{
		agent.WithFrontend(fe),
		agent.WithLogger(c.logger),
		agent.WithMaxSteps(c.cfg.MaxSteps),
		agent.WithToolMode(llm.ToolMode(c.cfg.ToolMode)),
	}
	if c.system != "" {
		opts = append(opts, agent.WithSystemPrompt(c.system))
	}
	return agent.NewRunner(c.client, disp, c.reg, opts...)
}

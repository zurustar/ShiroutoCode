package guardrail

import (
	"context"
	"fmt"
	"strings"

	"github.com/zurustar/shiroutocode/internal/log"
	"github.com/zurustar/shiroutocode/internal/tools"
)

// evaluator is the decision surface (interface for testability).
type evaluator interface {
	Evaluate(a Action) Decision
}

// ToolDispatcher is the single entry point for all tool execution (App Q5/R1).
type ToolDispatcher struct {
	reg       *tools.Registry
	ev        evaluator
	confirmer Confirmer
	logger    log.Logger
}

// NewToolDispatcher wires the registry, evaluator, confirmer and logger.
func NewToolDispatcher(reg *tools.Registry, ev evaluator, confirmer Confirmer, logger log.Logger) *ToolDispatcher {
	if logger == nil {
		logger = log.New("error", log.FormatText, discard{})
	}
	return &ToolDispatcher{reg: reg, ev: ev, confirmer: confirmer, logger: logger}
}

// Dispatch evaluates and (if permitted) executes a tool call. It is the only
// path to Tool.Execute.
func (d *ToolDispatcher) Dispatch(ctx context.Context, call tools.ToolCall) (tools.ToolResult, error) {
	action := toAction(call)
	dec := d.ev.Evaluate(action)

	switch dec.Kind {
	case Deny:
		d.logger.Warn("guardrail: denied", "tool", call.Name, "reason", dec.Reason)
		return tools.ToolResult{}, &BlockedError{Reason: "拒否: " + dec.Reason}
	case Confirm:
		if d.confirmer == nil {
			d.logger.Warn("guardrail: confirm required but non-interactive", "tool", call.Name)
			return tools.ToolResult{}, &BlockedError{Reason: "確認が必要ですが非対話環境のため実行できません: " + dec.Reason}
		}
		ok, err := d.confirmer.Confirm(ctx, action, dec.Reason)
		if err != nil {
			// Fail-closed: cannot obtain confirmation -> do not execute (R8/R9).
			return tools.ToolResult{}, &BlockedError{Reason: "確認を取得できませんでした: " + dec.Reason}
		}
		if !ok {
			return tools.ToolResult{}, &BlockedError{Reason: "ユーザーが操作を拒否しました"}
		}
	}

	tool, ok := d.reg.Get(call.Name)
	if !ok {
		return tools.ToolResult{}, fmt.Errorf("unknown tool: %s", call.Name)
	}
	d.logger.Info("guardrail: executing", "tool", call.Name, "decision", int(dec.Kind))
	return tool.Execute(ctx, call.Args)
}

// toAction maps a tool call to a guardrail Action using each tool's argument
// conventions.
func toAction(call tools.ToolCall) Action {
	a := Action{Tool: call.Name}
	get := func(k string) string {
		if v, ok := call.Args[k].(string); ok {
			return v
		}
		return ""
	}
	switch call.Name {
	case "read_file":
		a.Kind = FileRead
		a.Paths = []string{get("path")}
	case "write_file":
		if get("mode") == "delete" {
			a.Kind = FileDelete
		} else {
			a.Kind = FileWrite
		}
		a.Paths = []string{get("path")}
	case "run_command":
		a.Kind = Command
		if cl := get("command_line"); cl != "" {
			a.CommandLine = cl
		} else {
			a.CommandLine = strings.TrimSpace(get("command") + " " + joinArgs(call.Args["args"]))
		}
	case "git":
		a.Kind = GitOp
		a.CommandLine = strings.TrimSpace("git " + get("op") + " " + joinArgs(call.Args["args"]))
	case "web_fetch":
		a.Kind = WebFetch
		a.URL = get("url")
	default:
		a.Kind = Unknown
	}
	return a
}

func joinArgs(v any) string {
	raw, ok := v.([]any)
	if !ok {
		return ""
	}
	var parts []string
	for _, x := range raw {
		if s, ok := x.(string); ok {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " ")
}

// discard is an io.Writer that drops everything (default logger sink).
type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

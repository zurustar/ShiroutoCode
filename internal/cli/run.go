package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/zurustar/shiroutocode/internal/agent"
	"github.com/zurustar/shiroutocode/internal/config"
	"github.com/zurustar/shiroutocode/internal/guardrail"
	"github.com/zurustar/shiroutocode/internal/llm"
	"github.com/zurustar/shiroutocode/internal/log"
)

// Exit codes.
const (
	exitOK      = 0
	exitFailed  = 1
	exitUsage   = 2
	exitAborted = 130
)

// Env is the environment lookup (injected for testability).
type Env = map[string]string

// Run is the CLI entry point. args are the program arguments (excluding argv[0]).
// isTTY reports whether the session is interactive. It returns a process exit
// code.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer, stdin io.Reader, env Env, isTTY bool) int {
	prompt, err := extractPrompt(args, stderr)
	if err != nil {
		return exitUsage
	}

	cfg, err := config.Load(config.Options{Args: args, Env: env})
	if err != nil {
		fmt.Fprintf(stderr, "設定エラー:\n%s\n", err)
		return exitUsage
	}

	logger := log.New(cfg.LogLevel, log.Format(cfg.LogFormat), stderr)
	core := BuildCore(cfg, logger)

	switch {
	case prompt != "":
		return runSingleShot(ctx, core, prompt, stdout, stderr, stdin, isTTY)
	case isTTY:
		return runREPL(ctx, core, stdout, stderr)
	default:
		fmt.Fprintln(stderr, `使い方: shiroutocode "<指示>"   （または端末で対話モード）`)
		return exitUsage
	}
}

// extractPrompt parses flags (so positional args are isolated) and returns the
// joined positional prompt. Flag values themselves are consumed by config.Load.
func extractPrompt(args []string, stderr io.Writer) (string, error) {
	fs := flag.NewFlagSet("shiroutocode", flag.ContinueOnError)
	fs.SetOutput(stderr)
	for _, name := range []string{"endpoint", "model", "workspace", "log-level", "log-format", "log-file", "max-steps", "confirm-mode"} {
		fs.String(name, "", "")
	}
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	return strings.TrimSpace(strings.Join(fs.Args(), " ")), nil
}

func runSingleShot(ctx context.Context, core *Core, prompt string, stdout, stderr io.Writer, stdin io.Reader, isTTY bool) int {
	fe := &plainFrontend{w: stdout}
	var confirmer guardrail.Confirmer
	if isTTY {
		confirmer = newPromptConfirmer(stdin, stdout)
	} // else nil -> non-interactive, guardrail blocks Confirm actions
	runner := core.newRunner(fe, confirmer)

	res, err := runner.Run(ctx, agent.Task{Prompt: prompt})
	if err != nil {
		fmt.Fprintf(stderr, "エラー: %s\n", err)
		return exitFailed
	}
	return reportResult(res, stdout, stderr)
}

// reportResult prints a final summary and maps status to an exit code.
func reportResult(res agent.Result, stdout, stderr io.Writer) int {
	switch res.Status {
	case agent.Completed:
		fmt.Fprintf(stdout, "\n\n✅ 完了（%dステップ）\n", res.Steps)
		if len(res.ChangedFiles) > 0 {
			fmt.Fprintf(stdout, "変更ファイル: %s\n", strings.Join(res.ChangedFiles, ", "))
		}
		return exitOK
	case agent.StoppedMaxSteps:
		fmt.Fprintf(stderr, "\n⏹ 最大ステップに到達（未完）。指示を分割するか上限を上げてください。\n")
		return exitFailed
	case agent.Aborted:
		fmt.Fprintf(stderr, "\n⏹ 中断されました。\n")
		return exitAborted
	default: // Failed
		fmt.Fprintf(stderr, "\n❌ %s\n", failureMessage(res.Err))
		return exitFailed
	}
}

// failureMessage produces a user-facing message, surfacing connection guidance
// for LLM errors (US-6.1) without leaking internals (SECURITY-09).
func failureMessage(err error) string {
	if err == nil {
		return "処理に失敗しました。"
	}
	var le *llm.LLMError
	if errors.As(err, &le) {
		return le.UserMessage
	}
	return "処理に失敗しました。"
}

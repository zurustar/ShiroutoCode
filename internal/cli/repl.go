package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/zurustar/shiroutocode/internal/agent"
)

// runREPL is the interactive read-eval loop. It uses plain streaming output
// (not a full-screen TUI): the terminal's native line editing is used for the
// prompt — so Japanese/IME input works correctly — and agent events stream to
// stdout as they happen, so progress is visible while the LLM works.
func runREPL(ctx context.Context, core *Core, stdout, stderr io.Writer, stdin io.Reader) int {
	// Ensure the terminal edits multibyte (Japanese) input correctly: canonical
	// mode + echo + IUTF8 so one backspace erases a whole character. No-op when
	// stdin is not a real terminal (tests, pipes).
	if f, ok := stdin.(*os.File); ok {
		restore := enableCookedUTF8(int(f.Fd()))
		defer restore()
	}

	r := bufio.NewReader(stdin)

	// Choose the model after launch (per design). Pre-select any configured one.
	if err := replSelectModel(ctx, core, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "%s\n", modelSelectError(err))
		return exitUsage
	}

	fmt.Fprintf(stdout, "\nShiroutoCode 対話モード — モデル: %s\n%s\n", core.Model(), replHelp)

	return replLoop(ctx, core, stdout, stderr, r)
}

const replHelp = "指示を入力して Enter。コマンド: /model（モデル変更） /help /exit（Ctrl+D でも終了）"

// replLoop reads prompts and runs the agent until EOF, /exit, or cancellation.
// A model must already be selected.
func replLoop(ctx context.Context, core *Core, stdout, stderr io.Writer, r *bufio.Reader) int {
	for {
		if ctx.Err() != nil {
			return exitAborted
		}
		fmt.Fprint(stdout, "\n> ")
		line, err := r.ReadString('\n')
		if err != nil {
			// EOF (Ctrl+D) or read error: end the session cleanly.
			fmt.Fprintln(stdout)
			return exitOK
		}
		prompt := strings.TrimSpace(line)
		switch prompt {
		case "":
			continue
		case "/exit", "/quit":
			return exitOK
		case "/help":
			fmt.Fprintln(stdout, replHelp)
			continue
		case "/model":
			if err := replSelectModel(ctx, core, stdout, stderr); err != nil {
				fmt.Fprintf(stderr, "%s\n", modelSelectError(err))
				continue
			}
			fmt.Fprintf(stdout, "モデル: %s\n", core.Model())
			continue
		}
		runOnce(ctx, core, prompt, stdout, stderr, r)
	}
}

// runOnce executes a single instruction, streaming events to stdout and using
// the shared reader for confirmation prompts (so buffered input is not lost).
func runOnce(ctx context.Context, core *Core, prompt string, stdout, stderr io.Writer, r *bufio.Reader) {
	fe := &plainFrontend{w: stdout}
	confirmer := &promptConfirmer{in: r, out: stdout}
	runner := core.newRunner(fe, confirmer)

	fmt.Fprintln(stdout, "▶ 実行中…")
	res, err := runner.Run(ctx, agent.Task{Prompt: prompt})
	if err != nil {
		res = agent.Result{Status: agent.Failed, Err: err}
	}
	fmt.Fprintln(stdout, "\n"+doneSummary(res))
}

// replSelectModel opens the model picker, pre-selecting the current model, and
// applies the choice. Canceling keeps the current model (an error is returned
// only when none is selected and none was set, so the caller can abort).
func replSelectModel(ctx context.Context, core *Core, stdout, stderr io.Writer) error {
	chosen, err := selectModelStandalone(ctx, core, core.Model(), stdout)
	if err != nil {
		if err == errModelSelectionCanceled && core.Model() != "" {
			return nil // keep the current model
		}
		return err
	}
	core.SetModel(chosen)
	return nil
}

// doneSummary renders a one-line outcome for a completed run.
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

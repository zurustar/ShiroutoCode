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
// (not a full-screen TUI): agent events stream to stdout as they happen so
// progress is visible while the LLM works, and prompts are read with a
// raw-mode line editor that handles UTF-8 / full-width (Japanese) input —
// backspace deletes a whole character and Ctrl+C exits immediately.
func runREPL(ctx context.Context, core *Core, stdout, stderr io.Writer, stdin io.Reader) int {
	r := bufio.NewReader(stdin)

	fd := -1
	if f, ok := stdin.(*os.File); ok {
		fd = int(f.Fd())
	}
	readLine := newLineReader(fd, r, stdout)

	// Choose the model after launch (per design). Pre-select any configured one.
	if err := replSelectModel(ctx, core, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "%s\n", modelSelectError(err))
		return exitUsage
	}

	fmt.Fprintf(stdout, "\nShiroutoCode 対話モード — モデル: %s\n%s\n", core.Model(), replHelp)

	return replLoop(ctx, core, stdout, stderr, r, readLine)
}

const replHelp = "指示を入力して Enter。コマンド: /model（モデル変更） /reset（会話クリア） /help /exit（Ctrl+C / Ctrl+D でも終了）"

// replLoop reads prompts and runs the agent until EOF, /exit, or cancellation.
// A model must already be selected.
func replLoop(ctx context.Context, core *Core, stdout, stderr io.Writer, r *bufio.Reader, readLine lineReader) int {
	// One runner for the whole session: its conversation persists across turns,
	// so follow-up prompts remember earlier instructions, files, and results.
	fe := &plainFrontend{w: stdout}
	runner := core.newRunner(fe, &promptConfirmer{in: r, out: stdout})

	for {
		if ctx.Err() != nil {
			return exitAborted
		}
		line, err := readLine("\n> ")
		if err != nil {
			if err == errInterrupted {
				return exitAborted // Ctrl+C
			}
			return exitOK // EOF (Ctrl+D) or read error: end cleanly
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
		case "/reset", "/new":
			runner.Reset()
			fmt.Fprintln(stdout, "新しい会話を開始しました（履歴をクリア）。")
			continue
		case "/model":
			if err := replSelectModel(ctx, core, stdout, stderr); err != nil {
				fmt.Fprintf(stderr, "%s\n", modelSelectError(err))
				continue
			}
			fmt.Fprintf(stdout, "モデル: %s\n", core.Model())
			continue
		}
		runOnce(ctx, runner, fe, prompt, stdout)
	}
}

// runOnce executes a single instruction on the session runner, streaming events
// to stdout. The runner carries conversation history across calls. When a run
// completes without the model emitting any text, that is stated explicitly so
// "no reply" is never ambiguous with a dropped reply.
func runOnce(ctx context.Context, runner *agent.Runner, fe *plainFrontend, prompt string, stdout io.Writer) {
	fe.reset()
	fmt.Fprintln(stdout, "▶ 実行中…")
	res, err := runner.Run(ctx, agent.Task{Prompt: prompt})
	if err != nil {
		res = agent.Result{Status: agent.Failed, Err: err}
	}
	if res.Status == agent.Completed && !fe.wroteText {
		fmt.Fprintln(stdout, "\n（モデルはテキスト応答を返しませんでした。ツール実行のみ、または空の応答です。）")
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

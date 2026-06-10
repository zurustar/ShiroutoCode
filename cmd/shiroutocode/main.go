// Command shiroutocode is the CLI entry point for the ShiroutoCode agent.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"

	"github.com/zurustar/shiroutocode/internal/cli"
)

// Build information, injected at release time via -ldflags by GoReleaser.
// Defaults apply to `go build`/`go install` builds without ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-version", "version":
			fmt.Printf("shiroutocode %s (commit %s, built %s)\n", version, commit, date)
			os.Exit(0)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	env := map[string]string{}
	for _, kv := range os.Environ() {
		for i := 0; i < len(kv); i++ {
			if kv[i] == '=' {
				env[kv[:i]] = kv[i+1:]
				break
			}
		}
	}

	isTTY := term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))

	code := cli.Run(ctx, os.Args[1:], os.Stdout, os.Stderr, os.Stdin, env, isTTY)
	os.Exit(code)
}

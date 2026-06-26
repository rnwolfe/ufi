// Command ufi is an agent-friendly CLI for Ubiquiti UniFi Network over the official API
// (local Network Integration API + Site Manager cloud). It implements the agent-CLI contract
// (read-only by default, --json, schema --json, structured errors, exit codes, embedded
// SKILL.md). See AGENTS.md for build/test and spec.md for the design.
package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rnwolfe/ufi/internal/cli"
	"github.com/rnwolfe/ufi/internal/errs"
)

func main() {
	// SIGINT/SIGTERM → exit 130 (CANCELLED), not a panic/stack trace.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		os.Exit(errs.ExitCancelled)
	}()

	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

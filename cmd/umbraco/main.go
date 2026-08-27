package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"umbraco-cli/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cli.NewRootCommand().ExecuteContext(ctx); err != nil {
		// Quiet-exit errors carry only an exit code: the command already
		// printed its report, and machine consumers that merge streams must
		// not have it corrupted by a redundant summary line.
		var quiet interface{ QuietExit() bool }
		if !errors.As(err, &quiet) || !quiet.QuietExit() {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(cli.ExitCode(err))
	}
}

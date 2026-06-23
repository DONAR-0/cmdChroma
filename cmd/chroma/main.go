package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/DONAR-0/cmdChroma/cmd/chroma/output"
	"github.com/DONAR-0/cmdChroma/internal/version"
)

var (
	// AppVersion is the semantic version of the CLI.
	// It defaults to "dev" and can be overridden via go generate / ldflags.
	AppVersion  = version.Version
	ExitSuccess = 0
	ExitError   = 1

	// printer is the global console printer for user-facing output.
	printer *output.ConsolePrinter
)

func main() {
	app := createApp()

	// Apply panic recovery to all commands
	ApplyRecovery(app.Commands)

	var exitCode int

	defer func() {
		if r := recover(); r != nil {
			slog.Error("CLI application Panicked", "panic", r)
			fmt.Printf("Error: An Unexpected error occurred: %v\n", r)

			exitCode = ExitError
		}

		os.Exit(exitCode)
	}()

	if err := app.Run(context.Background(), os.Args); err != nil {
		slog.Error("CLI execution failed", "error", err)
		fmt.Printf("Error: %v\n", err)

		exitCode = ExitError
	}
}

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/urfave/cli/v3"
)

// withRecovery wraps a handler function with panic recovery.
// If a panic occurs, it logs the stack trace and returns a friendly error.
func withRecovery(fn func(context.Context, *cli.Command) error) func(context.Context, *cli.Command) error {
	return func(ctx context.Context, c *cli.Command) error {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Handler panicked",
					"panic", r,
					"command", c.Name,
					"stack", string(debug.Stack()),
				)
				fmt.Fprintf(os.Stderr, "Internal error: command '%s' failed unexpectedly\n", c.Name)
			}
		}()

		return fn(ctx, c)
	}
}

// ApplyRecovery wraps all command actions with panic recovery.
func ApplyRecovery(cmds []*cli.Command) {
	for _, cmd := range cmds {
		if cmd.Action != nil {
			originalAction := cmd.Action
			cmd.Action = withRecovery(func(ctx context.Context, c *cli.Command) error {
				return originalAction(ctx, c)
			})
		}

		// Recursively apply to subcommands
		if len(cmd.Commands) > 0 {
			ApplyRecovery(cmd.Commands)
		}
	}
}

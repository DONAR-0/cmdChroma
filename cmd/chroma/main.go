package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	client "github.com/DONAR-0/cmdChroma/internal/client"
	"github.com/DONAR-0/cmdChroma/internal/config"
	"github.com/DONAR-0/cmdChroma/internal/version"
	"github.com/urfave/cli/v3"
)

var (
	// AppVersion is the semantic version of the CLI.
	// It defaults to "dev" and can be overridden via go generate / ldflags.
	AppVersion  = version.Version
	ExitSuccess = 0
	ExitError   = 1
)

func main() {
	app := createApp()

	// Recover from panic gracefully
	defer func() {
		if r := recover(); r != nil {
			slog.Error("CLI application Panicked", "panic", r)
			fmt.Printf("Error: An Unexpected error occurred: %v\n", r)
			os.Exit(ExitError)
		}
	}()

	if err := app.Run(context.Background(), os.Args); err != nil {
		slog.Error("CLI execution failed", "error", err)
		fmt.Printf("Error: %v\n", err)
		os.Exit(ExitError)
	}
}

// createChromaClient creates a Chroma client based on CLI context
func createChromaClient(c *cli.Command) (*client.ChromaClient, error) {
	cfg, err := config.LoadConfig(c)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return client.NewChromaDBClient(cfg.GetChromaURL(), cfg.GetTenant(), cfg.GetDatabase()), nil
}

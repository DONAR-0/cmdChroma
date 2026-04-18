package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	client "github.com/donar0/cmdChroma/internal/client"
	"github.com/donar0/cmdChroma/internal/version"
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
	// Initialize Logger first
	InitLogger()

	// Recover from panic gracefully
	defer func() {
		if r := recover(); r != nil {
			slog.Error("CLI application Panicked", "panic", r)
			fmt.Printf("Error: An Unexpected error occurred: %v\n", r)
			os.Exit(ExitError)
		}
	}()

	app := createApp()

	if err := app.Run(context.Background(), os.Args); err != nil {
		slog.Error("CLI execution failed", "error", err)
		fmt.Printf("Error: %v\n", err)
		os.Exit(ExitError)
	}
}

// createChromaClient creates a Chroma client based on CLI context
func createChromaClient(c *cli.Command) (*client.ChromaClient, error) {
	// For now, use the default client
	// In the future, this could be enhanced to use host/port from flags
	return client.NewChromaDBClient(fmt.Sprintf("http://%s:%s", c.String("host"), c.String("port")), c.String("tenant"), c.String("database")), nil
}

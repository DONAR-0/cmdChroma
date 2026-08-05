package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/server"
)

func RunStdio(ctx context.Context, s *server.MCPServer) error {
	slog.Info("starting MCP server in stdio mode")

	srv := server.NewStdioServer(s)

	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.Listen(ctx, os.Stdin, os.Stdout)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("stdio server shutting down")
		return ctx.Err()
	}
}

func RunHTTP(ctx context.Context, s *server.MCPServer, addr string) error {
	slog.Info("starting MCP server in HTTP mode", "addr", addr)

	srv := server.NewStreamableHTTPServer(s)

	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.Start(addr)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("http server error: %w", err)
	case <-ctx.Done():
		slog.Info("http server shutting down")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http shutdown: %w", err)
		}

		return ctx.Err()
	}
}

func WaitForSignal() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		slog.Info("received signal, shutting down", "signal", sig)
		cancel()
	}()

	return ctx
}

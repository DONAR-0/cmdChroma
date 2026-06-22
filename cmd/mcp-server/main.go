package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/DONAR-0/cmdChroma/internal/client"
	"github.com/DONAR-0/cmdChroma/internal/onnx"
)

func main() {
	cfg := parseFlagsAndConfig()
	run(cfg)
}

func parseFlagsAndConfig() *Config {
	mode := flag.String("mode", "", `Server mode: "" (generic) or "memory"`)
	transport := flag.String("transport", "", `Transport: "stdio" or "http"`)
	port := flag.Int("port", 0, "HTTP port (default: 9090)")
	collection := flag.String("collection", "", "Default collection name")
	logLevel := flag.String("log-level", "", "Log level (debug, info, warn, error)")
	chromaURL := flag.String("chroma-url", "", "ChromaDB server URL")
	tenant := flag.String("tenant", "", "ChromaDB tenant")
	database := flag.String("database", "", "ChromaDB database")
	modelPath := flag.String("model-path", "", "ONNX model path")
	libraryPath := flag.String("library-path", "", "ONNX runtime library path")
	configPath := flag.String("config", "", "Path to config file")

	flag.Parse()

	clr := CLIOverrides{
		Mode:           *mode,
		Transport:      *transport,
		Port:           *port,
		Collection:     *collection,
		LogLevel:       *logLevel,
		ChromaURL:      *chromaURL,
		ChromaTenant:   *tenant,
		ChromaDatabase: *database,
		ModelPath:      *modelPath,
		LibraryPath:    *libraryPath,
	}

	var (
		cfg *Config
		err error
	)

	if *configPath != "" {
		cfg, err = Load(*configPath)
	} else {
		cfg, err = LoadAuto()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "config load: %v\n", err)
		os.Exit(1)
	}

	cfg.ApplyOverrides(clr)

	setupLogging(cfg.Server.LogLevel)

	return cfg
}

func run(cfg *Config) {
	slog.Info("initializing MCP server",
		"mode", cfg.Server.Mode,
		"transport", cfg.Server.Transport,
		"collection", cfg.Server.Collection,
	)

	embedderPath := cfg.Embedder.ModelPath
	tokenizerPath := deriveTokenizerPath(embedderPath)

	slog.Info("loading embedder",
		"model", embedderPath,
		"tokenizer", tokenizerPath,
		"library", cfg.Embedder.LibraryPath,
	)

	embedder, err := onnx.NewEmbedder(embedderPath, tokenizerPath, cfg.Embedder.LibraryPath)
	if err != nil {
		slog.Error("failed to init embedder", "error", err)
		os.Exit(1)
	}
	defer embedder.Close()

	slog.Info("connecting to ChromaDB",
		"url", cfg.Chroma.URL,
		"tenant", cfg.Chroma.Tenant,
		"database", cfg.Chroma.Database,
	)

	chromaClient := client.NewChromaDBClient(cfg.Chroma.URL, cfg.Chroma.Tenant, cfg.Chroma.Database)
	chromaClient.SetEmbedder(embedder)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := chromaClient.TestConnection(ctx); err != nil {
		slog.Error("ChromaDB connection failed", "error", err)
		os.Exit(1)
	}

	srv := buildServer(chromaClient, embedder, cfg.Server.Mode)
	slog.Info("server built",
		"tools", len(srv.ListTools()),
		"mode", cfg.Server.Mode,
	)

	ctx = WaitForSignal()

	switch cfg.Server.Transport {
	case "stdio":
		if err := RunStdio(ctx, srv); err != nil && err != context.Canceled {
			slog.Error("stdio server error", "error", err)
			os.Exit(1)
		}
	case "http":
		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		if err := RunHTTP(ctx, srv, addr); err != nil && err != context.Canceled {
			slog.Error("http server error", "error", err)
			os.Exit(1)
		}
	default:
		slog.Error("unsupported transport", "transport", cfg.Server.Transport)
		os.Exit(1)
	}

	slog.Info("server stopped")
}

func setupLogging(level string) {
	var lvl slog.Level

	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: lvl,
	}

	// In stdio mode, logs MUST go to stderr so JSON-RPC on stdout stays clean.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, opts)))
}

func deriveTokenizerPath(modelPath string) string {
	dir := filepath.Dir(modelPath)
	return filepath.Join(dir, "tokenizer.json")
}

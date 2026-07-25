package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DONAR-0/cmdChroma/cmd/chat-server/api"
	"github.com/DONAR-0/cmdChroma/cmd/chat-server/api/handler"
	"github.com/DONAR-0/cmdChroma/cmd/chat-server/config"
	"github.com/DONAR-0/cmdChroma/cmd/chat-server/service"
	"github.com/DONAR-0/cmdChroma/cmd/chat-server/storage"
)

func main() {
	var exitCode int
	defer func() { os.Exit(exitCode) }()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load("chatting_server.yaml")
	if err != nil {
		logger.Warn("config not found, using defaults", "err", err)

		c := config.Default()
		cfg = c
	}

	logger.Info("config loaded",
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
		"api_key_set", cfg.Server.APIKey != "",
		"collections", len(cfg.Collections),
	)

	// Initialize integrations (ChromaDB + ONNX embedder)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	embedder, chromaClient, err := service.InitIntegrations(ctx, &cfg.Chroma, &cfg.Embedder)
	if err != nil {
		logger.Error("failed to initialize integrations", "err", err)

		exitCode = 1

		return
	}

	// Build service layer
	chatService := service.NewChatService(logger, chromaClient, embedder, &cfg.LLM)
	sessionStore := storage.NewSessionStore()

	modelMgr := service.NewModelManager(logger, "cmdChroma/models")
	if err := modelMgr.LoadInstalled(); err != nil {
		logger.Warn("failed to load installed models", "err", err)
	}

	// Build handlers
	chatHandler := handler.NewChatHandler(chatService, sessionStore, cfg.LLM.DefaultModel)
	queryHandler := handler.NewQueryHandler(chatService)
	sessionHandler := handler.NewSessionHandler(sessionStore)
	healthHandler := handler.NewHealthHandler(chromaClient, embedder)
	collectionHandler := handler.NewCollectionHandler(chatService)
	importHandler := handler.NewImportHandler(chatService)
	modelHandler := handler.NewModelHandler(modelMgr)

	// Wire up router
	router := api.NewRouter(api.RouterDeps{
		APIKey:            cfg.Server.APIKey,
		ChatHandler:       chatHandler,
		QueryHandler:      queryHandler,
		SessionHandler:    sessionHandler,
		HealthHandler:     healthHandler,
		CollectionHandler: collectionHandler,
		ImportHandler:     importHandler,
		ModelHandler:      modelHandler,
		Collections:       cfg.Collections,
	})

	// HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 120 * time.Second, // long timeout for streaming
		IdleTimeout:  60 * time.Second,
	}

	// Start
	go func() {
		logger.Info("chatting_server starting", "addr", addr)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "err", err)
	}

	logger.Info("server stopped")
}

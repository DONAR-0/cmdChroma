package factory

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	client "github.com/DONAR-0/cmdChroma/internal/client"
	"github.com/DONAR-0/cmdChroma/internal/onnx"
	"github.com/DONAR-0/cmdChroma/internal/service"
	"github.com/urfave/cli/v3"
)

// chromaClient is the subset of *client.ChromaClient needed by ServiceFactory.
type chromaClient interface {
	ResolveCollectionID(ctx context.Context, input string) (string, error)
	AddBatch(ctx context.Context, collectionID string, docs []string, ids []string) error
	AddBatchGeneric(ctx context.Context, collectionID string, documents []string, ids []string, metadatas []map[string]any) error
	UpsertBatchGeneric(ctx context.Context, collectionID string, documents []string, ids []string, metadatas []map[string]any) error
	QueryBatch(ctx context.Context, collectionID string, queryTexts []string, nResults int) (*client.QueryResponse, error)
	ListCollections(ctx context.Context) ([]client.Collection, error)
	ListDocuments(ctx context.Context, collectionID string) (*client.GetRecordsResponse, error)
	CountDocuments(ctx context.Context, collectionID string) (int64, error)
	TestConnection(ctx context.Context) error
	GetTenant(ctx context.Context) (bool, error)
	ListDatabases(ctx context.Context) ([]client.Database, error)
	CreateDatabase(ctx context.Context, name string) error
	DeleteCollection(ctx context.Context, name string) error
	DeleteRecords(ctx context.Context, collectionID string, ids []string) error
	CreateCollection(ctx context.Context, name string) (string, error)
}

// embedder is the subset of *onnx.Embedder needed by ServiceFactory.
type embedder interface {
	Close()
}

// ServiceFactory creates and configures service dependencies.
// It centralizes client, embedder, and service initialization.
type ServiceFactory struct{}

// embedderConfig holds the resolved paths for AI components.
type embedderConfig struct {
	modelPath     string
	tokenizerPath string
	onnxLibPath   string
}

// NewServiceFactory creates a new ServiceFactory.
func NewServiceFactory() *ServiceFactory {
	return &ServiceFactory{}
}

// CreateChromaService creates a ChromaService with client and embedder.
// Returns the service and a cleanup function.
func (f *ServiceFactory) CreateChromaService(cmd *cli.Command) (*service.ChromaService, func(), error) {
	// Resolve AI paths
	cfg, err := f.resolveEmbedderConfig(cmd)
	if err != nil {
		return nil, nil, err
	}

	slog.Info("Loading AI embedding engine", "model", cfg.modelPath)

	// Create Chroma client
	chromaHost := fmt.Sprintf("http://%s:%s", cmd.String("host"), cmd.String("port"))
	chromaClient := client.NewChromaDBClient(chromaHost, cmd.String("tenant"), cmd.String("database"))

	// Create embedder
	embedder, err := onnx.NewEmbedder(cfg.modelPath, cfg.tokenizerPath, cfg.onnxLibPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize embedder: %w", err)
	}

	// Create service
	chromaClient.SetEmbedder(embedder)
	svc := service.NewChromaService(chromaClient, embedder)

	// Return cleanup function that closes the embedder
	cleanup := func() {
		if embedder != nil {
			embedder.Close()
		}
	}

	return svc, cleanup, nil
}

// CreateChromaClient creates just the ChromaDB client without embedder.
// Use this for operations that don't require embeddings (e.g., list, create, delete).
func (f *ServiceFactory) CreateChromaClient(cmd *cli.Command) (*client.ChromaClient, error) {
	chromaHost := fmt.Sprintf("http://%s:%s", cmd.String("host"), cmd.String("port"))
	return client.NewChromaDBClient(chromaHost, cmd.String("tenant"), cmd.String("database")), nil
}

// CreateChromaServiceWithEmbedder creates a ChromaService from pre-existing client and embedder.
// Use this when you need to inject custom instances for testing.
func (f *ServiceFactory) CreateChromaServiceWithEmbedder(
	c chromaClient,
	e embedder,
) *service.ChromaService {
	return service.NewChromaService(c, e)
}

// resolveEmbedderConfig determines the paths for model files with fallbacks.
func (f *ServiceFactory) resolveEmbedderConfig(cmd *cli.Command) (*embedderConfig, error) {
	// Determine base paths relative to executable
	ex, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve executable path: %w", err)
	}

	binDir := filepath.Dir(ex)
	projectRoot := filepath.Join(binDir, "..")

	// Resolve model path
	modelPath := cmd.String("model-path")
	if modelPath == "" {
		modelPath = filepath.Join(projectRoot, "models/all-MiniLM-L6-v2/model.onnx")
	}

	// Resolve tokenizer path
	tokenizerPath := cmd.String("tokenizer-path")
	if tokenizerPath == "" {
		tokenizerPath = filepath.Join(projectRoot, "models/all-MiniLM-L6-v2/tokenizer.json")
	}

	// Resolve ONNX lib path
	onnxLibPath := cmd.String("onnx-lib")
	if onnxLibPath == "" {
		onnxLibPath = filepath.Join(projectRoot, "models/onnx_runtime/lib/libonnxruntime.so")
	}

	// Validate model file exists if using default paths
	if _, err := os.Stat(modelPath); err != nil {
		slog.Debug("Model file not found", "path", modelPath, "error", err)
		return nil, fmt.Errorf("model file not found: %s\n\nHint: Use --model-path to specify the correct location or run the setup script", modelPath)
	}

	return &embedderConfig{
		modelPath:     modelPath,
		tokenizerPath: tokenizerPath,
		onnxLibPath:   onnxLibPath,
	}, nil
}

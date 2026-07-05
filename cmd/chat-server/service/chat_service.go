package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/DONAR-0/cmdChroma/cmd/chat-server/config"
	"github.com/DONAR-0/cmdChroma/cmd/chat-server/storage"
	"github.com/DONAR-0/cmdChroma/internal/client"
	"github.com/DONAR-0/cmdChroma/internal/ingest"
	"github.com/DONAR-0/cmdChroma/internal/llm"
	"github.com/DONAR-0/cmdChroma/internal/onnx"
	"github.com/DONAR-0/cmdChroma/internal/service"
)

// ChatService orchestrates: ChromaDB query → build RAG prompt → LLM stream.
type ChatService struct {
	logger       *slog.Logger
	chromaClient client.ChromaClientInterface
	embedder     onnx.EmbedderInterface
	chromaSvc    *service.ChromaService
	ollamaURL    string
	nimURL       string
	nimPrefixes  []string
}

// InitIntegrations boots the ONNX embedder and ChromaDB client.
func InitIntegrations(ctx context.Context, chromaCfg *config.ChromaConfig, embedderCfg *config.EmbedderConfig) (onnx.EmbedderInterface, client.ChromaClientInterface, error) {
	embedder, err := onnx.NewEmbedder(
		embedderCfg.ModelPath,
		strings.ReplaceAll(embedderCfg.ModelPath, "model.onnx", "tokenizer.json"),
		embedderCfg.LibraryPath,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load ONNX embedder: %w", err)
	}

	chromaClient := client.NewChromaDBClient(
		chromaCfg.URL,
		chromaCfg.Tenant,
		chromaCfg.Database,
	)
	chromaClient.SetEmbedder(embedder)

	return embedder, chromaClient, nil
}

// NewChatService creates a ChatService with injected dependencies.
func NewChatService(logger *slog.Logger, chromaClient client.ChromaClientInterface, embedder onnx.EmbedderInterface, llmCfg *config.LLMConfig) *ChatService {
	return &ChatService{
		logger:       logger,
		chromaClient: chromaClient,
		embedder:     embedder,
		chromaSvc:    service.NewChromaService(chromaClient, embedder),
		ollamaURL:    llmCfg.OllamaURL,
		nimURL:       llmCfg.NIMURL,
		nimPrefixes:  llmCfg.NIMPrefixes,
	}
}

// QueryResult holds a single retrieved document with its similarity score.
type QueryResult struct {
	ID       string         `json:"id"`
	Content  string         `json:"content"`
	Distance float32        `json:"distance"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Query runs a semantic search against ChromaDB and returns raw documents
// without LLM involvement (used by /api/query endpoint).
func (s *ChatService) Query(ctx context.Context, collectionName string, query string, nResults int, distanceThreshold float64) ([]QueryResult, error) {
	s.logger.Info("performing semantic search", "collection", collectionName, "n_results", nResults)

	collectionID, err := s.chromaClient.ResolveCollectionID(ctx, collectionName)
	if err != nil {
		s.logger.Error("collection not found", "collection", collectionName, "err", err)
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	resp, err := s.chromaClient.QueryBatch(ctx, collectionID, []string{query}, nResults)
	if err != nil {
		s.logger.Error("query failed", "collection_id", collectionID, "err", err)
		return nil, fmt.Errorf("query failed: %w", err)
	}

	results := make([]QueryResult, 0, nResults)

	for i := range resp.Documents[0] {
		distance := resp.Distances[0][i]
		if distanceThreshold > 0 && float64(distance) > distanceThreshold {
			continue
		}

		results = append(results, QueryResult{
			ID:       resp.IDs[0][i],
			Content:  resp.Documents[0][i],
			Distance: distance,
			Metadata: resp.Metadatas[0][i],
		})
	}

	s.logger.Info("query completed", "results_found", len(results))
	return results, nil
}

// CreateProvider returns the correct LLM provider for the model string.
// NIM models (google/*, meta/*, etc.) use NVIDIA NIM; everything else uses Ollama.
func (s *ChatService) CreateProvider(model, nimKey string) (llm.ProviderInterface, error) {
	model = strings.TrimPrefix(model, "nim://")

	for _, prefix := range s.nimPrefixes {
		if strings.HasPrefix(model, prefix) {
			s.logger.Debug("using NIM provider", "model", model)
			return llm.NewNIMProvider(s.nimURL, nimKey)
		}
	}

	s.logger.Debug("using Ollama provider", "model", model)
	return llm.NewProvider(s.ollamaURL), nil
}

// GetOllamaURL returns the configured Ollama base URL.
func (s *ChatService) GetOllamaURL() string { return s.ollamaURL }

// GetNIMURL returns the configured NVIDIA NIM base URL.
func (s *ChatService) GetNIMURL() string { return s.nimURL }

// ImportFile ingests records from a JSONL/Parquet file into a collection with progress reporting.
func (s *ChatService) ImportFile(ctx context.Context, collectionName, filePath string, contentField, idField string, progressFn func(int)) error {
	s.logger.Info("starting file import", "collection", collectionName, "path", filePath)

	cfg := &ingest.Config{
		BatchSize:    100,
		ContentField: contentField,
		IDField:      idField,
		AllMetadata:  true,
	}
	if cfg.ContentField == "" {
		cfg.ContentField = "text"
	}

	if cfg.IDField == "" {
		cfg.IDField = "id"
	}

	err := s.chromaSvc.IngestRecords(ctx, collectionName, filePath, cfg, progressFn)
	if err != nil {
		s.logger.Error("import failed", "collection", collectionName, "path", filePath, "err", err)
		return err
	}

	s.logger.Info("import completed successfully", "collection", collectionName, "path", filePath)
	return nil
}

// CollectionWithCount extends a ChromaDB collection with its document count.
type CollectionWithCount struct {
	client.Collection
	Count int `json:"count"`
}

// ListCollections returns all collections from ChromaDB with metadata.
func (s *ChatService) ListCollections(ctx context.Context) ([]client.Collection, error) {
	return s.chromaClient.ListCollections(ctx)
}

// ListCollectionsWithCount returns all collections with their document counts.
func (s *ChatService) ListCollectionsWithCount(ctx context.Context) ([]CollectionWithCount, error) {
	s.logger.Debug("listing collections with counts")
	collections, err := s.chromaClient.ListCollections(ctx)
	if err != nil {
		s.logger.Error("failed to list collections", "err", err)
		return nil, err
	}

	result := make([]CollectionWithCount, len(collections))
	for i, col := range collections {
		result[i] = CollectionWithCount{Collection: col}

		// NOTE: ListDocuments can be expensive for large collections. 
		// In a high-scale production environment, a dedicated count API should be used.
		records, err := s.chromaClient.ListDocuments(ctx, col.ID)
		if err == nil && records != nil {
			result[i].Count = len(records.IDs)
		} else if err != nil {
			s.logger.Warn("could not retrieve document count", "collection_id", col.ID, "err", err)
		}
	}

	return result, nil
}

// CreateCollection creates a new collection and returns its ID.
func (s *ChatService) CreateCollection(ctx context.Context, name string) (string, error) {
	return s.chromaClient.CreateCollection(ctx, name)
}

// DeleteCollection removes a collection by name.
func (s *ChatService) DeleteCollection(ctx context.Context, name string) error {
	return s.chromaClient.DeleteCollection(ctx, name)
}

// ListDocuments returns all documents in a collection.
func (s *ChatService) ListDocuments(ctx context.Context, collectionName string) (*client.GetRecordsResponse, error) {
	collectionID, err := s.chromaClient.ResolveCollectionID(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	return s.chromaClient.ListDocuments(ctx, collectionID)
}

// AddDocuments adds documents with optional metadata to a collection.
func (s *ChatService) AddDocuments(ctx context.Context, collectionName string, documents []string, ids []string, metadatas []map[string]any) error {
	collectionID, err := s.chromaClient.ResolveCollectionID(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("collection not found: %s", collectionName)
	}

	return s.chromaClient.AddBatchGeneric(ctx, collectionID, documents, ids, metadatas)
}

// DeleteDocuments removes specific documents from a collection by their IDs.
func (s *ChatService) DeleteDocuments(ctx context.Context, collectionName string, ids []string) error {
	collectionID, err := s.chromaClient.ResolveCollectionID(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("collection not found: %s", collectionName)
	}

	return s.chromaClient.DeleteRecords(ctx, collectionID, ids)
}

// BuildPromptWithHistory builds a RAG prompt with full session message history for multi-turn chat.
func (s *ChatService) BuildPromptWithHistory(history []storage.Message, query, context string, hasContext bool) string {
	var sb strings.Builder
	for _, m := range history {
		fmt.Fprintf(&sb, "%s: %s\n", m.Role, m.Content)
	}

	sb.WriteString("\n")

	if hasContext && context != "" {
		fmt.Fprintf(&sb, "Use the following context to answer the question.\n\nContext:\n%s\n\n", context)
	} else {
		sb.WriteString("No relevant documents were found in the knowledge base.\n\n")
	}

	fmt.Fprintf(&sb, "Question: %s\n\nProvide a clear, concise answer:", query)

	return sb.String()
}

// BuildRAGPrompt mirrors cmd/chroma/handlers.go buildRAGPrompt — used for single-turn chat.
func (s *ChatService) BuildRAGPrompt(question, context string, hasContext bool) string {
	return buildPrompt(question, context, hasContext)
}

func buildPrompt(question, context string, hasContext bool) string {
	if !hasContext || context == "" {
		return fmt.Sprintf(`The user asked: "%s"

No relevant documents were found in the knowledge base.

Instructions: Respond with a clear statement that the answer cannot be found in the provided context. Do not guess or provide information from general knowledge. Simply indicate that the query does not match any available context.`, question)
	}

	return fmt.Sprintf(`Use the following context to answer the question.
If the context doesn't contain relevant information, say so.

Context:
%s

Question: %s

Provide a clear, concise answer:`, context, question)
}

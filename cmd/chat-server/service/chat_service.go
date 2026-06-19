package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/DONAR-0/cmdChroma/cmd/chat-server/config"
	"github.com/DONAR-0/cmdChroma/cmd/chat-server/storage"
	"github.com/DONAR-0/cmdChroma/internal/client"
	"github.com/DONAR-0/cmdChroma/internal/llm"
	"github.com/DONAR-0/cmdChroma/internal/onnx"
)

// ChatService orchestrates: ChromaDB query → build RAG prompt → LLM stream.
type ChatService struct {
	chromaClient client.ChromaClientInterface
	embedder     onnx.EmbedderInterface
	ollamaURL    string
	nimURL       string
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
func NewChatService(chromaClient client.ChromaClientInterface, embedder onnx.EmbedderInterface, llmCfg *config.LLMConfig) *ChatService {
	return &ChatService{
		chromaClient: chromaClient,
		embedder:     embedder,
		ollamaURL:    llmCfg.OllamaURL,
		nimURL:       llmCfg.NIMURL,
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
	collectionID, err := s.chromaClient.ResolveCollectionID(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("collection not found: %s", collectionName)
	}

	resp, err := s.chromaClient.QueryBatch(ctx, collectionID, []string{query}, nResults)
	if err != nil {
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

	return results, nil
}

// CreateProvider returns the correct LLM provider for the model string.
// NIM models (google/*, meta/*, etc.) use NVIDIA NIM; everything else uses Ollama.
func (s *ChatService) CreateProvider(model, nimKey string) (llm.ProviderInterface, error) {
	model = strings.TrimPrefix(model, "nim://")

	nimPrefixes := []string{
		"google/", "meta/", "mistralai/", "nvidia/",
		"qwen/", "deepseek-", "minimaxai/", "snowflake/",
		"ibm/", "upstage/", "writer/", "z-ai/",
	}
	for _, prefix := range nimPrefixes {
		if strings.HasPrefix(model, prefix) {
			return llm.NewNIMProvider(s.nimURL, nimKey)
		}
	}

	return llm.NewProvider(s.ollamaURL), nil
}

// GetOllamaURL returns the configured Ollama base URL.
func (s *ChatService) GetOllamaURL() string { return s.ollamaURL }

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

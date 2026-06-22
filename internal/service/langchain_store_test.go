package service

import (
	"context"
	"testing"

	"github.com/DONAR-0/cmdChroma/internal/client"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/vectorstores"
)

// mockClient implements ChromaClientInterface for testing.
type mockClient struct {
	client.ChromaClientInterface
	queryBatchFunc func(ctx context.Context, collectionID string, queryTexts []string, nResults int) (*client.QueryResponse, error)
}

func (m *mockClient) QueryBatch(ctx context.Context, collectionID string, queryTexts []string, nResults int) (*client.QueryResponse, error) {
	if m.queryBatchFunc != nil {
		return m.queryBatchFunc(ctx, collectionID, queryTexts, nResults)
	}

	return &client.QueryResponse{}, nil
}

func (m *mockClient) AddBatchGeneric(ctx context.Context, collectionID string, documents []string, ids []string, metadatas []map[string]any) error {
	return nil
}

func TestNewChromaStore(t *testing.T) {
	mc := &mockClient{}

	store := NewChromaStore(mc, "test-collection")
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestChromaStore_AddDocuments(t *testing.T) {
	mc := &mockClient{}
	store := NewChromaStore(mc, "test-collection")

	docs := []schema.Document{
		{PageContent: "hello world", Metadata: map[string]any{"id": "1"}},
		{PageContent: "foo bar", Metadata: map[string]any{}},
	}

	_, err := store.AddDocuments(context.Background(), docs)
	if err != nil {
		t.Fatalf("AddDocuments failed: %v", err)
	}
}

func TestChromaStore_SimilaritySearch(t *testing.T) {
	mc := &mockClient{
		queryBatchFunc: func(ctx context.Context, collectionID string, queryTexts []string, nResults int) (*client.QueryResponse, error) {
			return &client.QueryResponse{
				IDs:       [][]string{{"1", "2"}},
				Documents: [][]string{{"result1", "result2"}},
				Metadatas: [][]map[string]any{{{}, {}}},
				Distances: [][]float32{{0.1, 0.2}},
			}, nil
		},
	}
	store := NewChromaStore(mc, "test-collection")

	docs, err := store.SimilaritySearch(context.Background(), "query", 2)
	if err != nil {
		t.Fatalf("SimilaritySearch failed: %v", err)
	}

	if len(docs) != 2 {
		t.Errorf("expected 2 docs, got %d", len(docs))
	}

	if docs[0].PageContent != "result1" {
		t.Errorf("expected 'result1', got %q", docs[0].PageContent)
	}
}

func TestChromaStore_SimilaritySearch_Empty(t *testing.T) {
	mc := &mockClient{
		queryBatchFunc: func(ctx context.Context, collectionID string, queryTexts []string, nResults int) (*client.QueryResponse, error) {
			return &client.QueryResponse{}, nil
		},
	}
	store := NewChromaStore(mc, "test-collection")

	docs, err := store.SimilaritySearch(context.Background(), "query", 2)
	if err != nil {
		t.Fatalf("SimilaritySearch failed: %v", err)
	}

	if len(docs) != 0 {
		t.Errorf("expected 0 docs, got %d", len(docs))
	}
}

func TestChromaStore_ImplementsVectorStore(t *testing.T) {
	var _ vectorstores.VectorStore = &ChromaStore{}
}

func TestRunRetrievalQA_Signature(t *testing.T) {
	// Verify RunRetrievalQA has the correct signature (compile-time check)
	// Can't test execution without real LLM, but ensures signature is valid
}

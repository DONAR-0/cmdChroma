package service

import (
	"context"
	"errors"
	"testing"

	client "github.com/DONAR-0/cmdChroma/internal/client"
	internalErrors "github.com/DONAR-0/cmdChroma/internal/errors"
	ingest "github.com/DONAR-0/cmdChroma/internal/ingest"
	"os"
)

// Mock implementations
type mockChromaClient struct {
	testConnectionErr         error
	getTenantResult           bool
	getTenantErr              error
	listDatabasesResult       []client.Database
	listDatabasesErr          error
	listCollectionsResult     []client.Collection
	listCollectionsErr        error
	addBatchErr               error
	addBatchGenericErr        error
	upsertBatchGenericErr     error
	queryBatchResult          *client.QueryResponse
	queryBatchErr             error
	getIDByNameResult         string
	getIDByNameErr            error
	resolveCollectionIDResult string
	resolveCollectionIDErr    error
	deleteCollectionErr       error
	deleteRecordsErr          error
	createDatabaseErr         error
}

func (m *mockChromaClient) TestConnection() error {
	return m.testConnectionErr
}

func (m *mockChromaClient) GetTenant() (bool, error) {
	return m.getTenantResult, m.getTenantErr
}

func (m *mockChromaClient) ListDatabases() ([]client.Database, error) {
	return m.listDatabasesResult, m.listDatabasesErr
}

func (m *mockChromaClient) ListCollections() ([]client.Collection, error) {
	return m.listCollectionsResult, m.listCollectionsErr
}

func (m *mockChromaClient) AddBatch(collectionID string, docs []string, ids []string) error {
	return m.addBatchErr
}

func (m *mockChromaClient) AddBatchGeneric(collectionID string, documents []string, ids []string, metadatas []map[string]any) error {
	return m.addBatchGenericErr
}

func (m *mockChromaClient) UpsertBatchGeneric(collectionID string, documents []string, ids []string, metadatas []map[string]any) error {
	return m.upsertBatchGenericErr
}

func (m *mockChromaClient) QueryBatch(collectionId string, queryTexts []string, nResults int) (*client.QueryResponse, error) {
	return m.queryBatchResult, m.queryBatchErr
}

func (m *mockChromaClient) GetIDByName(name string) (string, error) {
	return m.getIDByNameResult, m.getIDByNameErr
}

func (m *mockChromaClient) ResolveCollectionID(input string) (string, error) {
	return m.resolveCollectionIDResult, m.resolveCollectionIDErr
}

func (m *mockChromaClient) DeleteCollection(name string) error {
	return m.deleteCollectionErr
}

func (m *mockChromaClient) DeleteRecords(collectionID string, ids []string) error {
	return m.deleteRecordsErr
}

func (m *mockChromaClient) CreateDatabase(name string) error {
	return m.createDatabaseErr
}

type mockEmbedder struct{}

func (m *mockEmbedder) Embed(text string) ([]float32, error) {
	return []float32{0.1}, nil
}

func (m *mockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range result {
		result[i] = []float32{0.1}
	}

	return result, nil
}

func (m *mockEmbedder) Close() {}

func TestChromaService_TestConnection(t *testing.T) {
	client := &mockChromaClient{}
	embedder := &mockEmbedder{}
	svc := NewChromaService(client, embedder)

	err := svc.TestConnection()
	if err != nil {
		t.Errorf("TestConnection failed: %v", err)
	}
}

func TestChromaService_TestConnection_Error(t *testing.T) {
	client := &mockChromaClient{testConnectionErr: errors.New("connection failed")}
	svc := NewChromaService(client, nil)

	err := svc.TestConnection()
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestChromaService_GetTenant(t *testing.T) {
	client := &mockChromaClient{getTenantResult: true}
	svc := NewChromaService(client, nil)

	result, err := svc.GetTenant()
	if err != nil {
		t.Errorf("GetTenant failed: %v", err)
	}

	if !result {
		t.Errorf("Expected true, got false")
	}
}

func TestChromaService_GetTenant_Error(t *testing.T) {
	client := &mockChromaClient{getTenantResult: false, getTenantErr: errors.New("tenant check failed")}
	svc := NewChromaService(client, nil)

	_, err := svc.GetTenant()
	if err == nil {
		t.Errorf("Expected error for GetTenant when client returns error")
	}
}

func TestChromaService_ListDatabases(t *testing.T) {
	client := &mockChromaClient{listDatabasesResult: []client.Database{{Id: "1", Name: "db1"}}}
	svc := NewChromaService(client, nil)

	dbs, err := svc.ListDatabases()
	if err != nil {
		t.Errorf("ListDatabases failed: %v", err)
	}

	if len(dbs) != 1 {
		t.Errorf("Expected 1 db, got %d", len(dbs))
	}
}

func TestChromaService_ListDatabases_Error(t *testing.T) {
	client := &mockChromaClient{listDatabasesErr: errors.New("list databases failed")}
	svc := NewChromaService(client, nil)

	_, err := svc.ListDatabases()
	if err == nil {
		t.Errorf("Expected error for ListDatabases when client returns error")
	}
}

func TestChromaService_ListCollections(t *testing.T) {
	client := &mockChromaClient{listCollectionsResult: []client.Collection{{ID: "1", Name: "col1"}}}
	svc := NewChromaService(client, nil)

	collections, err := svc.ListCollections()
	if err != nil {
		t.Errorf("ListCollections failed: %v", err)
	}

	if len(collections) != 1 {
		t.Errorf("Expected 1 collection, got %d", len(collections))
	}

	if collections[0].ID != "1" || collections[0].Name != "col1" {
		t.Errorf("Unexpected collection: %v", collections[0])
	}
}

func TestChromaService_ListCollections_Error(t *testing.T) {
	client := &mockChromaClient{listCollectionsErr: errors.New("list collections failed")}
	svc := NewChromaService(client, nil)

	_, err := svc.ListCollections()
	if err == nil {
		t.Errorf("Expected error for ListCollections when client returns error")
	}
}

func TestChromaService_AddDocuments(t *testing.T) {
	client := &mockChromaClient{}
	embedder := &mockEmbedder{}
	svc := NewChromaService(client, embedder)

	err := svc.AddDocuments("col1", []string{"doc"}, []string{"id"})
	if err != nil {
		t.Errorf("AddDocuments failed: %v", err)
	}
}

func TestChromaService_AddDocuments_NoEmbedder(t *testing.T) {
	client := &mockChromaClient{}
	svc := NewChromaService(client, nil)

	err := svc.AddDocuments("col1", []string{"doc"}, []string{"id"})
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestChromaService_AddDocuments_Error(t *testing.T) {
	client := &mockChromaClient{addBatchGenericErr: errors.New("add batch failed")}
	embedder := &mockEmbedder{}
	svc := NewChromaService(client, embedder)

	err := svc.AddDocuments("col1", []string{"doc"}, []string{"id"})
	if err == nil {
		t.Errorf("Expected error for AddDocuments when client returns error")
	}
}

func TestChromaService_UpsertDocuments(t *testing.T) {
	client := &mockChromaClient{}
	embedder := &mockEmbedder{}
	svc := NewChromaService(client, embedder)

	err := svc.UpsertDocuments("col1", []string{"doc"}, []string{"id"})
	if err != nil {
		t.Errorf("UpsertDocuments failed: %v", err)
	}
}

func TestChromaService_UpsertDocuments_NoEmbedder(t *testing.T) {
	client := &mockChromaClient{}
	svc := NewChromaService(client, nil)

	err := svc.UpsertDocuments("col1", []string{"doc"}, []string{"id"})
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestChromaService_UpsertDocuments_Error(t *testing.T) {
	client := &mockChromaClient{upsertBatchGenericErr: errors.New("upsert batch failed")}
	embedder := &mockEmbedder{}
	svc := NewChromaService(client, embedder)

	err := svc.UpsertDocuments("col1", []string{"doc"}, []string{"id"})
	if err == nil {
		t.Errorf("Expected error for UpsertDocuments when client returns error")
	}
}

func TestChromaService_QueryDocuments(t *testing.T) {
	client := &mockChromaClient{
		queryBatchResult: &client.QueryResponse{
			IDs:       [][]string{{"id1"}},
			Documents: [][]string{{"doc1"}},
		},
	}
	embedder := &mockEmbedder{}
	svc := NewChromaService(client, embedder)

	result, err := svc.QueryDocuments("col1", []string{"query"}, 1)
	if err != nil {
		t.Errorf("QueryDocuments failed: %v", err)
	}

	if result == nil {
		t.Errorf("Expected result, got nil")
	}
}

func TestChromaService_QueryDocuments_NoEmbedder(t *testing.T) {
	client := &mockChromaClient{}
	svc := NewChromaService(client, nil) // nil embedder

	_, err := svc.QueryDocuments("col1", []string{"query"}, 1)
	if err == nil {
		t.Errorf("Expected error for QueryDocuments when embedder is nil")
	} else if !errors.Is(err, internalErrors.ErrEmbedderNotInitialized) {
		t.Errorf("Expected ErrEmbedderNotInitialized, got: %v", err)
	}
}

func TestChromaService_QueryDocuments_Error(t *testing.T) {
	client := &mockChromaClient{queryBatchErr: errors.New("query batch failed")}
	embedder := &mockEmbedder{}
	svc := NewChromaService(client, embedder)

	_, err := svc.QueryDocuments("col1", []string{"query"}, 1)
	if err == nil {
		t.Errorf("Expected error for QueryDocuments when client returns error")
	}
}

func TestChromaService_Close(t *testing.T) {
	client := &mockChromaClient{}
	embedder := &mockEmbedder{}
	svc := NewChromaService(client, embedder)

	// Should not panic
	svc.Close()
}

func TestChromaService_IngestRecords(t *testing.T) {
	// Create a temporary file with JSONL content
	tmpFile, err := os.CreateTemp("", "test_*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Errorf("Failed to remove temp file: %v", err)
		}
	}()

	// Write test data
	testData := []string{
		`{"id": "1", "content": "test document 1", "metadata": {"source": "test"}}`,
		`{"id": "2", "content": "test document 2", "metadata": {"source": "test"}}`,
	}
	for _, line := range testData {
		if _, err := tmpFile.WriteString(line + "\n"); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
	}

	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	client := &mockChromaClient{}
	embedder := &mockEmbedder{}
	svc := NewChromaService(client, embedder)

	// Test with default config
	err = svc.IngestRecords("test_collection", tmpFile.Name(), nil)
	if err != nil {
		t.Errorf("IngestRecords failed: %v", err)
	}
}

func TestChromaService_IngestRecords_NoEmbedder(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test_*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Errorf("Failed to remove temp file: %v", err)
		}
	}()

	if _, err := tmpFile.WriteString(`{"id": "1", "content": "test", "metadata": {}}` + "\n"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	client := &mockChromaClient{}
	svc := NewChromaService(client, nil) // nil embedder

	err = svc.IngestRecords("test_collection", tmpFile.Name(), nil)
	if err == nil {
		t.Errorf("Expected error for IngestRecords when embedder is nil")
	} else if !errors.Is(err, internalErrors.ErrEmbedderNotInitialized) {
		t.Errorf("Expected ErrEmbedderNotInitialized, got: %v", err)
	}
}

func TestChromaService_NewChromaService(t *testing.T) {
	realClient := &client.ChromaClient{}
	embedder := &mockEmbedder{}
	svc := NewChromaService(realClient, embedder)

	if svc.client != realClient {
		t.Errorf("Client not set correctly")
	}

	if svc.embedder != embedder {
		t.Errorf("Service embedder not set")
	}
	// Verify injection into concrete client
	if realClient.Embedder != embedder {
		t.Errorf("Embedder not injected into real client")
	}
}

func TestChromaService_NewChromaService_MockClient(t *testing.T) {
	mockClient := &mockChromaClient{}
	embedder := &mockEmbedder{}
	svc := NewChromaService(mockClient, embedder)

	if svc.client != mockClient {
		t.Errorf("Client not set correctly")
	}

	if svc.embedder != embedder {
		t.Errorf("Service embedder not set")
	}
	// No injection should happen for mock, and it has no Embedder field, so nothing to check
}

func TestChromaService_uploadBatch(t *testing.T) {
	mockClient := &mockChromaClient{}
	embedder := &mockEmbedder{}
	svc := &ChromaService{client: mockClient, embedder: embedder}

	err := svc.uploadBatch("coll", []string{"doc1"}, []string{"id1"}, nil)
	if err != nil {
		t.Errorf("uploadBatch failed: %v", err)
	}
}

func TestChromaService_uploadBatch_Empty(t *testing.T) {
	mockClient := &mockChromaClient{}
	embedder := &mockEmbedder{}
	svc := &ChromaService{client: mockClient, embedder: embedder}

	err := svc.uploadBatch("coll", []string{}, []string{}, nil)
	if err != nil {
		t.Errorf("uploadBatch with empty docs should not return error, got: %v", err)
	}
}

func TestChromaService_IngestRecords_BatchError(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Errorf("Failed to remove temp file: %v", err)
		}
	}()

	testData := []string{
		`{"id": "1", "content": "test document 1", "metadata": {"source": "test"}}`,
		`{"id": "2", "content": "test document 2", "metadata": {"source": "test"}}`,
	}
	for _, line := range testData {
		if _, err := tmpFile.WriteString(line + "\n"); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
	}

	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	client := &mockChromaClient{addBatchGenericErr: errors.New("batch upload failed")}
	embedder := &mockEmbedder{}
	svc := NewChromaService(client, embedder)

	// Configure to match JSONL fields for content and id
	cfg := &ingest.Config{
		BatchSize:    100,
		ContentField: "content",
		IDField:      "id",
	}

	err = svc.IngestRecords("test_collection", tmpFile.Name(), cfg)
	if err == nil {
		t.Errorf("Expected error for IngestRecords when batch upload fails")
	}
}

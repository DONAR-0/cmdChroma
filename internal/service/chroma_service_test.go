package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	client "github.com/DONAR-0/cmdChroma/internal/client"
	internalErrors "github.com/DONAR-0/cmdChroma/internal/errors"
	ingest "github.com/DONAR-0/cmdChroma/internal/ingest"
	"github.com/DONAR-0/cmdChroma/internal/onnx"
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
	listDocumentsResult       *client.GetRecordsResponse
	listDocumentsErr          error
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

func (m *mockChromaClient) TestConnection(_ context.Context) error {
	return m.testConnectionErr
}

func (m *mockChromaClient) GetTenant(_ context.Context) (bool, error) {
	return m.getTenantResult, m.getTenantErr
}

func (m *mockChromaClient) ListDatabases(_ context.Context) ([]client.Database, error) {
	return m.listDatabasesResult, m.listDatabasesErr
}

func (m *mockChromaClient) ListCollections(_ context.Context) ([]client.Collection, error) {
	return m.listCollectionsResult, m.listCollectionsErr
}

func (m *mockChromaClient) AddBatch(ctx context.Context, collectionID string, docs []string, ids []string) error {
	return m.addBatchErr
}

func (m *mockChromaClient) AddBatchGeneric(ctx context.Context, collectionID string, documents []string, ids []string, metadatas []map[string]any) error {
	return m.addBatchGenericErr
}

func (m *mockChromaClient) UpsertBatchGeneric(ctx context.Context, collectionID string, documents []string, ids []string, metadatas []map[string]any) error {
	return m.upsertBatchGenericErr
}

func (m *mockChromaClient) QueryBatch(ctx context.Context, collectionID string, queryTexts []string, nResults int) (*client.QueryResponse, error) {
	return m.queryBatchResult, m.queryBatchErr
}

func (m *mockChromaClient) GetIDByName(_ context.Context, _ string) (string, error) {
	return m.getIDByNameResult, m.getIDByNameErr
}

func (m *mockChromaClient) ResolveCollectionID(_ context.Context, _ string) (string, error) {
	return m.resolveCollectionIDResult, m.resolveCollectionIDErr
}

func (m *mockChromaClient) ListDocuments(_ context.Context, _ string) (*client.GetRecordsResponse, error) {
	return m.listDocumentsResult, m.listDocumentsErr
}

func (m *mockChromaClient) DeleteCollection(_ context.Context, _ string) error {
	return m.deleteCollectionErr
}

func (m *mockChromaClient) DeleteRecords(_ context.Context, _ string, ids []string) error {
	return m.deleteRecordsErr
}

func (m *mockChromaClient) CreateDatabase(_ context.Context, _ string) error {
	return m.createDatabaseErr
}

func (m *mockChromaClient) CreateCollection(_ context.Context, name string) (string, error) {
	return "test-collection-id", nil
}

func (m *mockChromaClient) SetEmbedder(_ onnx.EmbedderInterface) {}

type mockEmbedder struct{}

func (m *mockEmbedder) Embed(_ string) ([]float32, error) {
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

	err := svc.TestConnection(context.Background())
	if err != nil {
		t.Errorf("TestConnection failed: %v", err)
	}
}

func TestChromaService_TestConnection_Error(t *testing.T) {
	client := &mockChromaClient{testConnectionErr: errors.New("connection failed")}
	svc := NewChromaService(client, nil)

	err := svc.TestConnection(context.Background())
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestChromaService_GetTenant(t *testing.T) {
	client := &mockChromaClient{getTenantResult: true}
	svc := NewChromaService(client, nil)

	result, err := svc.GetTenant(context.Background())
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

	_, err := svc.GetTenant(context.Background())
	if err == nil {
		t.Errorf("Expected error for GetTenant when client returns error")
	}
}

func TestChromaService_ListDatabases(t *testing.T) {
	client := &mockChromaClient{listDatabasesResult: []client.Database{{ID: "1", Name: "db1"}}}
	svc := NewChromaService(client, nil)

	dbs, err := svc.ListDatabases(context.Background())
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

	_, err := svc.ListDatabases(context.Background())
	if err == nil {
		t.Errorf("Expected error for ListDatabases when client returns error")
	}
}

func TestChromaService_ListCollections(t *testing.T) {
	client := &mockChromaClient{listCollectionsResult: []client.Collection{{ID: "1", Name: "col1"}}}
	svc := NewChromaService(client, nil)

	collections, err := svc.ListCollections(context.Background())
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

	_, err := svc.ListCollections(context.Background())
	if err == nil {
		t.Errorf("Expected error for ListCollections when client returns error")
	}
}

func TestChromaService_AddDocuments(t *testing.T) {
	client := &mockChromaClient{}
	embedder := &mockEmbedder{}
	svc := NewChromaService(client, embedder)

	err := svc.AddDocuments(context.Background(), "col1", []string{"doc"}, []string{"id"})
	if err != nil {
		t.Errorf("AddDocuments failed: %v", err)
	}
}

func TestChromaService_AddDocuments_NoEmbedder(t *testing.T) {
	client := &mockChromaClient{}
	svc := NewChromaService(client, nil)

	err := svc.AddDocuments(context.Background(), "col1", []string{"doc"}, []string{"id"})
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestChromaService_AddDocuments_Error(t *testing.T) {
	client := &mockChromaClient{addBatchGenericErr: errors.New("add batch failed")}
	embedder := &mockEmbedder{}
	svc := NewChromaService(client, embedder)

	err := svc.AddDocuments(context.Background(), "col1", []string{"doc"}, []string{"id"})
	if err == nil {
		t.Errorf("Expected error for AddDocuments when client returns error")
	}
}

func TestChromaService_UpsertDocuments(t *testing.T) {
	client := &mockChromaClient{}
	embedder := &mockEmbedder{}
	svc := NewChromaService(client, embedder)

	err := svc.UpsertDocuments(context.Background(), "col1", []string{"doc"}, []string{"id"})
	if err != nil {
		t.Errorf("UpsertDocuments failed: %v", err)
	}
}

func TestChromaService_UpsertDocuments_NoEmbedder(t *testing.T) {
	client := &mockChromaClient{}
	svc := NewChromaService(client, nil)

	err := svc.UpsertDocuments(context.Background(), "col1", []string{"doc"}, []string{"id"})
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestChromaService_UpsertDocuments_Error(t *testing.T) {
	client := &mockChromaClient{upsertBatchGenericErr: errors.New("upsert batch failed")}
	embedder := &mockEmbedder{}
	svc := NewChromaService(client, embedder)

	err := svc.UpsertDocuments(context.Background(), "col1", []string{"doc"}, []string{"id"})
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

	result, err := svc.QueryDocuments(context.Background(), "col1", []string{"query"}, 1)
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

	_, err := svc.QueryDocuments(context.Background(), "col1", []string{"query"}, 1)
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

	_, err := svc.QueryDocuments(context.Background(), "col1", []string{"query"}, 1)
	if err == nil {
		t.Errorf("Expected error for QueryDocuments when client returns error")
	}
}

func TestChromaService_Close(_ *testing.T) {
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
	err = svc.IngestRecords(context.Background(), "test_collection", tmpFile.Name(), nil)
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

	err = svc.IngestRecords(context.Background(), "test_collection", tmpFile.Name(), nil)
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

	err := svc.uploadBatch(context.Background(), "coll", []string{"doc1"}, []string{"id1"}, nil, false)
	if err != nil {
		t.Errorf("uploadBatch failed: %v", err)
	}
}

func TestChromaService_uploadBatch_Empty(t *testing.T) {
	mockClient := &mockChromaClient{}
	embedder := &mockEmbedder{}
	svc := &ChromaService{client: mockClient, embedder: embedder}

	err := svc.uploadBatch(context.Background(), "coll", []string{}, []string{}, nil, false)
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

	err = svc.IngestRecords(context.Background(), "test_collection", tmpFile.Name(), cfg)
	if err == nil {
		t.Errorf("Expected error for IngestRecords when batch upload fails")
	}
}

func TestGetFileExt(t *testing.T) {
	tests := []struct {
		path   string
		expect string
	}{
		{"file.jsonl", ".jsonl"},
		{"file.parquet", ".parquet"},
		{"dir/file.jsonl", ".jsonl"},
		{"/abs/path/file.jsonl", ".jsonl"},
		{"file.tar.gz", ".gz"},
		{"noextension", ""},
		{".hidden", ".hidden"},
		{"file.", "."},
		{"", ""},
		{"file.JSONL", ".jsonl"},  // case insensitivity
		{"dir\\file", ""},         // backslash before dot, no extension
		{"dir\\file.txt", ".txt"}, // backslash with extension
		{"dir/file", ""},          // slash before dot, no extension
	}

	for _, tt := range tests {
		result := getFileExt(tt.path)
		if result != tt.expect {
			t.Errorf("getFileExt(%q) = %q, want %q", tt.path, result, tt.expect)
		}
	}
}

func TestChromaService_IngestRecords_UnsupportedFormat(t *testing.T) {
	// Create a temporary file with unsupported extension
	tmpFile, err := os.CreateTemp("", "test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	if _, err := tmpFile.WriteString("some content"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	client := &mockChromaClient{}
	embedder := &mockEmbedder{}
	svc := NewChromaService(client, embedder)

	err = svc.IngestRecords(context.Background(), "test_collection", tmpFile.Name(), nil)
	if err == nil {
		t.Error("Expected error for unsupported file format")
	}

	if !strings.Contains(err.Error(), "unsupported file format") {
		t.Errorf("Expected unsupported format error, got: %v", err)
	}
}

func TestChromaService_IngestRecords_Batching(t *testing.T) {
	// Create a temporary JSONL file with 5 records
	tmpFile, err := os.CreateTemp("", "test_*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	records := []string{
		`{"id":"1","content":"doc1","metadata":{}}`,
		`{"id":"2","content":"doc2","metadata":{}}`,
		`{"id":"3","content":"doc3","metadata":{}}`,
		`{"id":"4","content":"doc4","metadata":{}}`,
		`{"id":"5","content":"doc5","metadata":{}}`,
	}
	for _, line := range records {
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

	// Use a small batch size to trigger batch uploads
	cfg := &ingest.Config{BatchSize: 2}

	err = svc.IngestRecords(context.Background(), "test_collection", tmpFile.Name(), cfg)
	if err != nil {
		t.Fatalf("IngestRecords failed: %v", err)
	}
	// If we get here, batching logic executed successfully
}

func TestChromaService_IngestRecords_WithProgressCallback(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	for i := 0; i < 5; i++ {
		line := `{"id":"` + fmt.Sprintf("%d", i) + `","content":"doc ` + fmt.Sprintf("%d", i) + `"}`
		_, _ = tmpFile.WriteString(line + "\n")
	}

	_ = tmpFile.Close()

	var progressCalls []int

	client := &mockChromaClient{}
	svc := NewChromaService(client, &mockEmbedder{})

	cfg := &ingest.Config{
		BatchSize:    100,
		ContentField: "content",
		IDField:      "id",
		Total:        5,
		OnProgress: func(info ingest.ProgressInfo) {
			progressCalls = append(progressCalls, info.Processed)
		},
	}

	err = svc.IngestRecords(context.Background(), "test_collection", tmpFile.Name(), cfg)
	if err != nil {
		t.Fatalf("IngestRecords failed: %v", err)
	}

	if len(progressCalls) == 0 {
		t.Error("Expected at least one progress callback call")
	}

	if progressCalls[len(progressCalls)-1] != 5 {
		t.Errorf("Last progress call should have Processed=5, got %d", progressCalls[len(progressCalls)-1])
	}
}

func TestChromaService_IngestRecords_DedupSkip(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// Two records with the same ID
	_, _ = tmpFile.WriteString(`{"id":"dup","content":"first"}` + "\n")
	_, _ = tmpFile.WriteString(`{"id":"dup","content":"second"}` + "\n")
	_ = tmpFile.Close()

	client := &mockChromaClient{}
	svc := NewChromaService(client, &mockEmbedder{})

	cfg := &ingest.Config{
		BatchSize:    100,
		ContentField: "content",
		IDField:      "id",
		DedupMode:    "skip",
	}

	err = svc.IngestRecords(context.Background(), "test_collection", tmpFile.Name(), cfg)
	if err != nil {
		t.Fatalf("IngestRecords failed: %v", err)
	}
}

func TestChromaService_IngestRecords_DedupWarn(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, _ = tmpFile.WriteString(`{"id":"dup","content":"first"}` + "\n")
	_, _ = tmpFile.WriteString(`{"id":"dup","content":"second"}` + "\n")
	_ = tmpFile.Close()

	client := &mockChromaClient{}
	svc := NewChromaService(client, &mockEmbedder{})

	cfg := &ingest.Config{
		BatchSize:    100,
		ContentField: "content",
		IDField:      "id",
		DedupMode:    "warn",
	}

	err = svc.IngestRecords(context.Background(), "test_collection", tmpFile.Name(), cfg)
	if err != nil {
		t.Fatalf("IngestRecords failed: %v", err)
	}
}

func TestChromaService_IngestRecords_Upsert(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test_*.jsonl")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, _ = tmpFile.WriteString(`{"id":"1","content":"doc1"}` + "\n")
	_ = tmpFile.Close()

	client := &mockChromaClient{}
	svc := NewChromaService(client, &mockEmbedder{})

	cfg := &ingest.Config{
		BatchSize:    100,
		ContentField: "content",
		IDField:      "id",
		Upsert:       true,
	}

	err = svc.IngestRecords(context.Background(), "test_collection", tmpFile.Name(), cfg)
	if err != nil {
		t.Fatalf("IngestRecords with Upsert failed: %v", err)
	}
}

func TestChromaService_GetDocuments(t *testing.T) {
	client := &mockChromaClient{
		resolveCollectionIDResult: "resolved-id",
		listDocumentsResult: &client.GetRecordsResponse{
			IDs:       []string{"doc1", "doc2"},
			Documents: []string{"content1", "content2"},
		},
	}
	embedder := &mockEmbedder{}
	svc := NewChromaService(client, embedder)

	result, err := svc.GetDocuments(context.Background(), "my_collection")
	if err != nil {
		t.Errorf("GetDocuments failed: %v", err)
	}

	if result == nil {
		t.Fatalf("Expected result, got nil")
	}

	if len(result.IDs) != 2 {
		t.Errorf("Expected 2 IDs, got %d", len(result.IDs))
	}

	if client.resolveCollectionIDResult != "resolved-id" {
		t.Errorf("Expected resolveCollectionIDResult='resolved-id', got '%s'", client.resolveCollectionIDResult)
	}
}

func TestChromaService_GetDocuments_ListError(t *testing.T) {
	client := &mockChromaClient{
		listDocumentsErr: errors.New("list failed"),
	}
	svc := NewChromaService(client, &mockEmbedder{})

	_, err := svc.GetDocuments(context.Background(), "my_collection")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	if err != nil && !strings.Contains(err.Error(), "failed to list documents") {
		t.Errorf("Expected 'failed to list documents' in error, got: %v", err)
	}
}

func TestChromaService_Query(t *testing.T) {
	client := &mockChromaClient{
		resolveCollectionIDResult: "coll-id",
		queryBatchResult: &client.QueryResponse{
			IDs:       [][]string{{"id1"}},
			Documents: [][]string{{"doc1"}},
			Distances: [][]float32{{0.1}},
		},
	}
	embedder := &mockEmbedder{}
	svc := NewChromaService(client, embedder)

	ctx := context.Background()

	result, err := svc.QueryDocuments(ctx, "my_collection", []string{"query"}, 5)
	if err != nil {
		t.Errorf("Query failed: %v", err)
	}

	if result == nil {
		t.Fatalf("Expected result, got nil")
	}

	if len(result.IDs) != 1 {
		t.Errorf("Expected 1 result set, got %d", len(result.IDs))
	}
}

func TestChromaService_Query_NoEmbedder(t *testing.T) {
	client := &mockChromaClient{}
	svc := NewChromaService(client, nil) // no embedder

	_, err := svc.QueryDocuments(context.Background(), "coll", []string{"q"}, 1)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	if err != nil && !errors.Is(err, internalErrors.ErrEmbedderNotInitialized) {
		t.Errorf("Expected ErrEmbedderNotInitialized, got: %v", err)
	}
}

// ExampleChromaService demonstrates creating a service and using its main methods.
// This example shows the typical workflow: create service, add documents, query.
func ExampleChromaService() {
	// In a real application, use real implementations
	realClient := &client.ChromaClient{}
	embedder := &onnx.Embedder{} // Assume initialized

	svc := NewChromaService(realClient, embedder)

	// Use the service...
	_ = svc
}

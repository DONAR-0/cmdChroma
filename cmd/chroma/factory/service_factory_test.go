package factory

import (
	"context"
	"testing"

	"github.com/DONAR-0/cmdChroma/internal/client"
	"github.com/DONAR-0/cmdChroma/internal/onnx"
	"github.com/urfave/cli/v3"
)

// mockChromaClient implements client.ChromaClientInterface for testing
type mockChromaClient struct{}

func (m *mockChromaClient) TestConnection(ctx context.Context) error    { return nil }
func (m *mockChromaClient) GetTenant(ctx context.Context) (bool, error) { return true, nil }
func (m *mockChromaClient) ListDatabases(ctx context.Context) ([]client.Database, error) {
	return nil, nil
}
func (m *mockChromaClient) CreateDatabase(ctx context.Context, name string) error { return nil }
func (m *mockChromaClient) ListCollections(ctx context.Context) ([]client.Collection, error) {
	return nil, nil
}

func (m *mockChromaClient) CountDocuments(ctx context.Context, collectionID string) (int64, error) {
	return 0, nil
}

func (m *mockChromaClient) CreateCollection(ctx context.Context, name string) (string, error) {
	return "test-id", nil
}

func (m *mockChromaClient) AddBatch(ctx context.Context, collectionID string, docs []string, ids []string) error {
	return nil
}

func (m *mockChromaClient) AddBatchGeneric(ctx context.Context, collectionID string, documents []string, ids []string, metadatas []map[string]any) error {
	return nil
}

func (m *mockChromaClient) UpsertBatchGeneric(ctx context.Context, collectionID string, documents []string, ids []string, metadatas []map[string]any) error {
	return nil
}

func (m *mockChromaClient) QueryBatch(ctx context.Context, collectionID string, queryTexts []string, nResults int) (*client.QueryResponse, error) {
	return nil, nil
}

func (m *mockChromaClient) ListDocuments(ctx context.Context, collectionID string) (*client.GetRecordsResponse, error) {
	return nil, nil
}

func (m *mockChromaClient) ResolveCollectionID(ctx context.Context, input string) (string, error) {
	return "", nil
}
func (m *mockChromaClient) DeleteCollection(ctx context.Context, name string) error { return nil }
func (m *mockChromaClient) DeleteRecords(ctx context.Context, collectionID string, ids []string) error {
	return nil
}
func (m *mockChromaClient) SetEmbedder(e *onnx.Embedder) {}

// mockEmbedder implements onnx.EmbedderInterface for testing
type mockEmbedder struct{}

func (m *mockEmbedder) Embed(text string) ([]float32, error) { return []float32{0.1, 0.2, 0.3}, nil }
func (m *mockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	return [][]float32{{0.1, 0.2, 0.3}}, nil
}
func (m *mockEmbedder) Close() {}

func TestNewServiceFactory(t *testing.T) {
	f := NewServiceFactory()
	if f == nil {
		t.Fatal("NewServiceFactory() = nil")
	}
}

func TestCreateChromaClient(t *testing.T) {
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "host"},
			&cli.StringFlag{Name: "port"},
			&cli.StringFlag{Name: "tenant"},
			&cli.StringFlag{Name: "database"},
		},
	}
	if err := cmd.Set("host", "testhost"); err != nil {
		t.Fatal(err)
	}

	if err := cmd.Set("port", "9999"); err != nil {
		t.Fatal(err)
	}

	if err := cmd.Set("tenant", "testtenant"); err != nil {
		t.Fatal(err)
	}

	if err := cmd.Set("database", "testdb"); err != nil {
		t.Fatal(err)
	}

	f := NewServiceFactory()

	cli, err := f.CreateChromaClient(cmd)
	if err != nil {
		t.Fatalf("CreateChromaClient() error = %v", err)
	}

	if cli == nil {
		t.Fatal("CreateChromaClient() returned nil")
	}
}

func TestCreateChromaServiceWithEmbedder(t *testing.T) {
	mockClient := &mockChromaClient{}
	mockEmbedder := &mockEmbedder{}

	f := NewServiceFactory()

	svc := f.CreateChromaServiceWithEmbedder(mockClient, mockEmbedder)
	if svc == nil {
		t.Fatal("CreateChromaServiceWithEmbedder() returned nil")
	}
}

func TestServiceFactory_CreateChromaService_RequiresEmbedder(t *testing.T) {
	// CreateChromaService requires ONNX model files which don't exist in test env
	// This test verifies the function signature and basic setup
	f := NewServiceFactory()

	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "host", Value: "localhost"},
			&cli.StringFlag{Name: "port", Value: "8000"},
			&cli.StringFlag{Name: "tenant", Value: "default_tenant"},
			&cli.StringFlag{Name: "database", Value: "default_database"},
			&cli.StringFlag{Name: "model-path", Value: "/nonexistent/model.onnx"},
			&cli.StringFlag{Name: "tokenizer-path", Value: "/nonexistent/tokenizer.json"},
			&cli.StringFlag{Name: "onnx-lib", Value: "/nonexistent/libonnxruntime.so"},
		},
	}

	_, _, err := f.CreateChromaService(cmd)
	// Expected to fail because model files don't exist
	if err == nil {
		t.Error("expected error for missing model files, got nil")
	}
	// Verify it's the expected error about model file
	expectedErr := "model file not found"
	if err != nil && !containsString(err.Error(), expectedErr) {
		t.Errorf("error = %q, want to contain %q", err.Error(), expectedErr)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

func TestMockClient_Compiles(_ *testing.T) {
}

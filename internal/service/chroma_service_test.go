package service

import (
	"context"
	"errors"
	"testing"

	client "github.com/donar0/cmdChroma/internal/client"
)

// Mock implementations
type mockChromaClient struct {
	testConnectionErr error
	getTenantResult   bool
	getTenantErr      error
}

func (m *mockChromaClient) TestConnection() error {
	return m.testConnectionErr
}

func (m *mockChromaClient) GetTenant() (bool, error) {
	return m.getTenantResult, m.getTenantErr
}

func (m *mockChromaClient) ListDatabases() ([]client.Database, error) {
	return []client.Database{{Id: "1", Name: "db1"}}, nil
}

func (m *mockChromaClient) ListCollections() ([]client.Collection, error) {
	return []client.Collection{{ID: "1", Name: "col1"}}, nil
}

func (m *mockChromaClient) AddBatch(collectionID string, docs []string, ids []string) error {
	return nil
}

func (m *mockChromaClient) QueryBatch(collectionId string, queryTexts []string, nResults int) (*client.QueryResponse, error) {
	return &client.QueryResponse{
		IDs:       [][]string{{"id1"}},
		Documents: [][]string{{"doc1"}},
	}, nil
}

func (m *mockChromaClient) GetIDByName(name string) (string, error) {
	return "test-id", nil
}

type mockEmbedder struct{}

func (m *mockEmbedder) Embed(text string) ([]float32, error) {
	return []float32{0.1}, nil
}

func (m *mockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	return [][]float32{{0.1}}, nil
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

func TestChromaService_ListDatabases(t *testing.T) {
	client := &mockChromaClient{}
	svc := NewChromaService(client, nil)

	dbs, err := svc.ListDatabases()
	if err != nil {
		t.Errorf("ListDatabases failed: %v", err)
	}
	if len(dbs) != 1 {
		t.Errorf("Expected 1 db, got %d", len(dbs))
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

func TestChromaService_QueryDocuments(t *testing.T) {
	client := &mockChromaClient{}
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

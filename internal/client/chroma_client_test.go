package cClient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChromaClient_TestConnection(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/heartbeat" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "ok"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create client
	client := NewChromaDBClient(server.URL, "tenant", "db")

	// Test
	err := client.TestConnection()
	if err != nil {
		t.Errorf("TestConnection failed: %v", err)
	}
}

func TestChromaClient_GetTenant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/tenants/test_tenant" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "test_tenant", "db")

	exists, err := client.GetTenant()
	if err != nil {
		t.Errorf("GetTenant failed: %v", err)
	}
	if !exists {
		t.Errorf("Expected tenant to exist")
	}
}

func TestChromaClient_ListDatabases(t *testing.T) {
	dbs := []Database{
		{Id: "1", Name: "db1", Tenant: "tenant"},
	}
	data, _ := json.Marshal(dbs)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/tenants/tenant/databases" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	result, err := client.ListDatabases()
	if err != nil {
		t.Errorf("ListDatabases failed: %v", err)
	}
	if len(result) != 1 || result[0].Id != "1" {
		t.Errorf("Unexpected result: %v", result)
	}
}

// Mock embedder for testing
type mockEmbedder struct{}

func (m *mockEmbedder) Embed(text string) ([]float32, error) {
	return []float32{0.1, 0.2}, nil
}

func (m *mockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i := range result {
		result[i] = []float32{0.1, 0.2}
	}
	return result, nil
}

func (m *mockEmbedder) Close() {}

func TestChromaClient_AddBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/test/add" {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")
	client.Embedder = &mockEmbedder{}

	err := client.AddBatch("test", []string{"doc1"}, []string{"id1"})
	if err != nil {
		t.Errorf("AddBatch failed: %v", err)
	}
}

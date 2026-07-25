package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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

func TestChromaClient_TestConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/heartbeat" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "ok"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	err := client.TestConnection(context.Background())
	if err != nil {
		t.Errorf("TestConnection failed: %v", err)
	}
}

func TestChromaClient_TestConnection_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/heartbeat" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	err := client.TestConnection(context.Background())
	if err == nil {
		t.Errorf("Expected error for TestConnection when server returns 500")
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

	exists, err := client.GetTenant(context.Background())
	if err != nil {
		t.Errorf("GetTenant failed: %v", err)
	}

	if !exists {
		t.Errorf("Expected tenant to exist")
	}
}

func TestChromaClient_GetTenant_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/tenants/test_tenant" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "test_tenant", "db")

	exists, err := client.GetTenant(context.Background())
	if err == nil {
		t.Errorf("Expected error for GetTenant when server returns 500")
	}

	if exists {
		t.Errorf("Expected tenant to not exist when error occurs")
	}
}

func TestChromaClient_ListDatabases(t *testing.T) {
	dbs := []Database{
		{ID: "1", Name: "db1", Tenant: "tenant"},
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

	result, err := client.ListDatabases(context.Background())
	if err != nil {
		t.Errorf("ListDatabases failed: %v", err)
	}

	if len(result) != 1 || result[0].ID != "1" {
		t.Errorf("Unexpected result: %v", result)
	}
}

func TestChromaClient_ListDatabases_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/tenants/tenant/databases" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	_, err := client.ListDatabases(context.Background())
	if err == nil {
		t.Errorf("Expected error for ListDatabases when server returns 500")
	}
}

func TestChromaClient_ListCollections(t *testing.T) {
	collections := []Collection{
		{ID: "coll1", Name: "collection1", Tenant: "tenant", Database: "db"},
		{ID: "coll2", Name: "collection2", Tenant: "tenant", Database: "db"},
	}
	data, _ := json.Marshal(collections)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	result, err := client.ListCollections(context.Background())
	if err != nil {
		t.Errorf("ListCollections failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 collections, got %d", len(result))
	}

	if result[0].ID != "coll1" || result[0].Name != "collection1" {
		t.Errorf("Unexpected first collection: %v", result[0])
	}
}

func TestChromaClient_ListCollections_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	_, err := client.ListCollections(context.Background())
	if err == nil {
		t.Errorf("Expected error for ListCollections when server returns 500")
	}
}

func TestChromaClient_CreateCollection(t *testing.T) {
	resp := map[string]string{"id": "coll123"}
	data, _ := json.Marshal(resp)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")
	client.Embedder = &mockEmbedder{}

	id, err := client.CreateCollection(context.Background(), "test_collection")
	if err != nil {
		t.Errorf("CreateCollection failed: %v", err)
	}

	if id != "coll123" {
		t.Errorf("Expected ID coll123, got %s", id)
	}
}

func TestChromaClient_CreateCollection_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")
	client.Embedder = &mockEmbedder{}

	_, err := client.CreateCollection(context.Background(), "test_collection")
	if err == nil {
		t.Errorf("Expected error for CreateCollection when server returns 500")
	}
}

func TestChromaClient_ListDocuments(t *testing.T) {
	resp := GetRecordsResponse{
		IDs:       []string{"doc1", "doc2"},
		Documents: []string{"document one", "document two"},
		Metadatas: []map[string]any{{"key": "value"}, nil},
	}
	data, _ := json.Marshal(resp)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/test/get" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	result, err := client.ListDocuments(context.Background(), "test")
	if err != nil {
		t.Errorf("ListDocuments failed: %v", err)
	}

	if len(result.IDs) != 2 {
		t.Errorf("Expected 2 document IDs, got %d", len(result.IDs))
	}

	if result.IDs[0] != "doc1" || result.Documents[0] != "document one" {
		t.Errorf("Unexpected first document")
	}
}

func TestChromaClient_ListDocuments_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/test/get" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	_, err := client.ListDocuments(context.Background(), "test")
	if err == nil {
		t.Errorf("Expected error for ListDocuments when server returns 500")
	}
}

func TestChromaClient_ResolveCollectionID(t *testing.T) {
	collections := []Collection{
		{ID: "coll123", Name: "my_collection", Tenant: "tenant", Database: "db"},
	}
	data, _ := json.Marshal(collections)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	// Test existing collection
	id, err := client.ResolveCollectionID(context.Background(), "my_collection")
	if err != nil {
		t.Errorf("ResolveCollectionID failed for existing collection: %v", err)
	}

	if id != "coll123" {
		t.Errorf("Expected ID coll123, got %s", id)
	}

	// Test non-existing collection (should error)
	_, err = client.ResolveCollectionID(context.Background(), "non_existent")
	if err == nil {
		t.Error("ResolveCollectionID should error for non-existent collection")
	}
}

func TestChromaClient_ResolveCollectionID_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	// Should error when listing fails
	_, err := client.ResolveCollectionID(context.Background(), "non_existent")
	if err == nil {
		t.Error("ResolveCollectionID should error when listing fails")
	}
}

func TestChromaClient_GetIDByName(t *testing.T) {
	collections := []Collection{
		{ID: "coll123", Name: "my_collection", Tenant: "tenant", Database: "db"},
	}
	data, _ := json.Marshal(collections)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	id, err := client.GetIDByName(context.Background(), "my_collection")
	if err != nil {
		t.Errorf("GetIDByName failed: %v", err)
	}

	if id != "coll123" {
		t.Errorf("Expected ID coll123, got %s", id)
	}

	// Test non-existing
	id, err = client.GetIDByName(context.Background(), "non_existent")
	if err != nil {
		t.Errorf("GetIDByName failed for non-existent: %v", err)
	}

	if id != "non_existent" {
		t.Errorf("Expected input as ID, got %s", id)
	}
}

func TestChromaClient_GetIDByName_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	_, err := client.GetIDByName(context.Background(), "non_existent")
	if err != nil {
		t.Errorf("GetIDByName should not error for non-existent collection when listing fails: %v", err)
	}
}

func TestChromaClient_DeleteCollection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections":
			data, _ := json.Marshal([]Collection{{ID: "test-uuid", Name: "test_collection"}})
			_, _ = w.Write(data)
		case r.Method == "DELETE" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/test-uuid":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	err := client.DeleteCollection(context.Background(), "test_collection")
	if err != nil {
		t.Errorf("DeleteCollection failed: %v", err)
	}
}

func TestChromaClient_DeleteCollection_CollectionNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections":
			data, _ := json.Marshal([]Collection{{ID: "other-uuid", Name: "other_collection"}})
			_, _ = w.Write(data)
		case r.Method == "DELETE" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/other-uuid":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	err := client.DeleteCollection(context.Background(), "test_collection")
	if err == nil {
		t.Error("Expected error for DeleteCollection when collection not found")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestChromaClient_DeleteCollection_ListError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	err := client.DeleteCollection(context.Background(), "test_collection")
	if err == nil {
		t.Error("Expected error for DeleteCollection when listing fails")
	}
}

func TestChromaClient_DeleteCollection_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/test_collection" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	err := client.DeleteCollection(context.Background(), "test_collection")
	if err == nil {
		t.Errorf("Expected error for DeleteCollection when server returns 500")
	}
}

func TestChromaClient_CountDocuments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/test/count" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("42"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	count, err := client.CountDocuments(context.Background(), "test")
	if err != nil {
		t.Errorf("CountDocuments failed: %v", err)
	}

	if count != 42 {
		t.Errorf("Expected count 42, got %d", count)
	}
}

func TestChromaClient_CountDocuments_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/test/count" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	_, err := client.CountDocuments(context.Background(), "test")
	if err == nil {
		t.Error("Expected error for CountDocuments when server returns 500")
	}
}

func TestChromaClient_CountDocuments_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/test/count" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not a number"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	_, err := client.CountDocuments(context.Background(), "test")
	if err == nil {
		t.Error("Expected error for CountDocuments with invalid response")
	}
}

func TestChromaClient_DeleteRecords(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections":
			data, _ := json.Marshal([]Collection{{ID: "test-uuid", Name: "test"}})
			_, _ = w.Write(data)
		case r.Method == "POST" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/test-uuid/delete":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	err := client.DeleteRecords(context.Background(), "test", []string{"id1", "id2"})
	if err != nil {
		t.Errorf("DeleteRecords failed: %v", err)
	}
}

func TestChromaClient_DeleteRecords_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections":
			data, _ := json.Marshal([]Collection{{ID: "test-uuid", Name: "test"}})
			_, _ = w.Write(data)
		case r.Method == "POST" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/test-uuid/delete":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	err := client.DeleteRecords(context.Background(), "test", []string{"id1"})
	if err == nil {
		t.Errorf("Expected error for DeleteRecords when server returns 500")
	}
}

func TestChromaClient_AddDocument(t *testing.T) {
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

	vector, _ := client.Embedder.Embed("document text")

	err := client.AddDocument("test", "document text", "doc1", vector)
	if err != nil {
		t.Errorf("AddDocument failed: %v", err)
	}
}

func TestChromaClient_AddDocument_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/test/add" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")
	client.Embedder = &mockEmbedder{}

	vector, _ := client.Embedder.Embed("document text")

	err := client.AddDocument("test", "document text", "doc1", vector)
	if err == nil {
		t.Errorf("Expected error for AddDocument when server returns 500")
	}
}

func TestChromaClient_GenerateLocalEmbedding(t *testing.T) {
	client := NewChromaDBClient("", "tenant", "db")
	client.Embedder = &mockEmbedder{}

	embedding, err := client.GenerateLocalEmbedding("test text")
	if err != nil {
		t.Errorf("GenerateLocalEmbedding failed: %v", err)
	}

	if len(embedding) != 2 || embedding[0] != 0.1 || embedding[1] != 0.2 {
		t.Errorf("Unexpected embedding: %v", embedding)
	}
}

func TestChromaClient_GenerateLocalEmbedding_NoEmbedder(t *testing.T) {
	client := NewChromaDBClient("", "tenant", "db")
	// Embedder is nil

	_, err := client.GenerateLocalEmbedding("test text")
	if err == nil {
		t.Errorf("Expected error for GenerateLocalEmbedding when embedder is nil")
	}
}

func TestChromaClient_QueryBatch(t *testing.T) {
	resp := QueryResponse{
		IDs:       [][]string{{"doc1", "doc2"}},
		Documents: [][]string{{"document one", "document two"}},
		Metadatas: [][]map[string]any{{{"key": "value"}}, {nil}},
		Distances: [][]float32{{0.1, 0.2}},
	}
	data, _ := json.Marshal(resp)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/test/query" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")
	client.Embedder = &mockEmbedder{}

	result, err := client.QueryBatch(context.Background(), "test", []string{"test query"}, 2)
	if err != nil {
		t.Errorf("QueryBatch failed: %v", err)
	}

	if len(result.IDs) != 1 || len(result.IDs[0]) != 2 {
		t.Errorf("Expected 1 query with 2 results, got %d queries with %d results", len(result.IDs), len(result.IDs[0]))
	}

	if result.IDs[0][0] != "doc1" || result.Documents[0][0] != "document one" {
		t.Errorf("Unexpected first result")
	}
}

func TestChromaClient_QueryBatch_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/test/query" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")
	client.Embedder = &mockEmbedder{}

	_, err := client.QueryBatch(context.Background(), "test", []string{"test query"}, 2)
	if err == nil {
		t.Errorf("Expected error for QueryBatch when server returns 500")
	}
}

func TestChromaClient_QueryBatch_EmptyQueries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/test/query" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ids": [[]], "documents": [[]], "metadatas": [[]], "distances": [[]]}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")
	client.Embedder = &mockEmbedder{}

	result, err := client.QueryBatch(context.Background(), "test", []string{}, 2)
	if err != nil {
		t.Errorf("QueryBatch with empty queries failed: %v", err)
	}

	if len(result.IDs) != 1 || len(result.IDs[0]) != 0 {
		t.Errorf("Expected 1 query with 0 results, got %d queries with %d results", len(result.IDs), len(result.IDs[0]))
	}
}

func TestChromaClient_AddBatchGeneric(t *testing.T) {
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

	docs := []string{"doc1", "doc2"}
	ids := []string{"id1", "id2"}
	metas := []map[string]any{{"key": "value"}, nil}

	err := client.AddBatchGeneric(context.Background(), "test", docs, ids, metas)
	if err != nil {
		t.Errorf("AddBatchGeneric failed: %v", err)
	}
}

func TestChromaClient_AddBatchGeneric_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/test/add" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")
	client.Embedder = &mockEmbedder{}

	docs := []string{"doc1", "doc2"}
	ids := []string{"id1", "id2"}
	metas := []map[string]any{{"key": "value"}, nil}

	err := client.AddBatchGeneric(context.Background(), "test", docs, ids, metas)
	if err == nil {
		t.Errorf("Expected error for AddBatchGeneric when server returns 500")
	}
}

func TestChromaClient_UpsertBatchGeneric(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/test/upsert" {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")
	client.Embedder = &mockEmbedder{}

	docs := []string{"doc1", "doc2"}
	ids := []string{"id1", "id2"}
	metas := []map[string]any{{"key": "value"}, nil}

	err := client.UpsertBatchGeneric(context.Background(), "test", docs, ids, metas)
	if err != nil {
		t.Errorf("UpsertBatchGeneric failed: %v", err)
	}
}

func TestChromaClient_UpsertBatchGeneric_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/test/upsert" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"upsert failed"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")
	client.Embedder = &mockEmbedder{}

	docs := []string{"doc1"}
	ids := []string{"id1"}

	err := client.UpsertBatchGeneric(context.Background(), "test", docs, ids, nil)
	if err == nil {
		t.Errorf("Expected error for UpsertBatchGeneric when server returns 500")
	}
}

func TestChromaClient_AddBatch_EmptySlice(t *testing.T) {
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

	err := client.AddBatch(context.Background(), "test", []string{}, []string{})
	if err != nil {
		t.Errorf("AddBatch with empty slices failed: %v", err)
	}
}

func TestChromaClient_AddBatch_MismatchedLengths(t *testing.T) {
	// This test ensures the method doesn't panic; behavior may vary.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/v2/tenants/tenant/databases/db/collections/test/add" {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")
	client.Embedder = &mockEmbedder{}

	_ = client.AddBatch(context.Background(), "test", []string{"doc1", "doc2"}, []string{"id1"})
	// No assertion on error; just ensure no panic
}

func TestChromaClient_CreateDatabase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/v2/tenants/tenant/databases" {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	err := client.CreateDatabase(context.Background(), "newdb")
	if err != nil {
		t.Errorf("CreateDatabase failed: %v", err)
	}
}

func TestChromaClient_CreateDatabase_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/api/v2/tenants/tenant/databases" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"creation failed"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	err := client.CreateDatabase(context.Background(), "newdb")
	if err == nil {
		t.Error("Expected error for CreateDatabase when server returns 500")
	}
}

func TestChromaMetadataKeyHint(t *testing.T) {
	tests := []struct {
		body string
		want string
	}{
		{`metadatas[23].questionText: data did not match any variant`, "questionText"},
		{`metadatas[0].answer: some error here`, "answer"},
		{`no prefix here`, ""},
		{`metadatas[0]: no dot`, ""},
		{`metadatas[0].:`, ""},
		{``, ""},
	}
	for _, tt := range tests {
		got := chromaMetadataKeyHint([]byte(tt.body))
		require.Equal(t, tt.want, got, "chromaMetadataKeyHint(%q)", tt.body)
	}
}

func TestWrapChromaError(t *testing.T) {
	t.Run("422 metadata error", func(t *testing.T) {
		body := []byte(`metadatas[0].questionText: data did not match any variant of "MetadataValue"`)
		err := wrapChromaError(422, body)
		require.Error(t, err)
		require.Contains(t, err.Error(), "questionText")
		require.Contains(t, err.Error(), "exclude-field")
	})

	t.Run("422 with unknown field", func(t *testing.T) {
		body := []byte(`MetadataValue validation error (no field hint)`)
		err := wrapChromaError(422, body)
		require.Error(t, err)
		require.Contains(t, err.Error(), "must be strings, numbers, or booleans")
	})

	t.Run("non-422 error", func(t *testing.T) {
		body := []byte(`internal server error`)
		err := wrapChromaError(500, body)
		require.Error(t, err)
		require.Contains(t, err.Error(), "500")
		require.Contains(t, err.Error(), "internal server error")
	})
}

func TestSanitizeValue(t *testing.T) {
	require.Equal(t, nil, sanitizeValue(nil))
	require.Equal(t, "hello", sanitizeValue("hello"))
	require.Equal(t, 42, sanitizeValue(42))
	require.Equal(t, true, sanitizeValue(true))
	require.Equal(t, "[1 2 3]", sanitizeValue([]int{1, 2, 3}))
	require.Equal(t, "map[key:value]", sanitizeValue(map[string]string{"key": "value"}))
}

func TestSanitizeMetadataForChroma(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		result := sanitizeMetadataForChroma(nil)
		require.Nil(t, result)
	})

	t.Run("sanitizes complex values", func(t *testing.T) {
		input := []map[string]any{
			{"simple": "value", "nested": map[string]any{"k": "v"}, "slice": []int{1, 2}},
		}
		result := sanitizeMetadataForChroma(input)
		require.Len(t, result, 1)
		require.Equal(t, "value", result[0]["simple"])
	})

	t.Run("nil metadata entry", func(t *testing.T) {
		input := []map[string]any{
			{"key": nil},
		}
		result := sanitizeMetadataForChroma(input)
		require.Len(t, result, 1)
		require.Nil(t, result[0]["key"])
	})
}

func TestChromaClient_SetEmbedder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")

	// Initially embedder should be nil
	if client.Embedder != nil {
		t.Error("Expected Embedder to be nil initially")
	}

	// Set embedder
	mockEmb := &mockEmbedder{}
	client.SetEmbedder(mockEmb)

	// Verify it's set
	if client.Embedder == nil {
		t.Error("Expected Embedder to be set after SetEmbedder")
	}

	if client.Embedder != mockEmb {
		t.Error("Embedder not the expected instance")
	}
}

// slowEmbedder simulates an embedder that takes time, supporting context cancellation.
type slowEmbedder struct {
	delay time.Duration
}

func (e *slowEmbedder) Embed(text string) ([]float32, error) {
	return []float32{0.1}, nil
}

func (e *slowEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	select {
	case <-time.After(e.delay):
		result := make([][]float32, len(texts))
		for i := range result {
			result[i] = []float32{0.1}
		}

		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (e *slowEmbedder) Close() {}

func TestChromaClient_QueryBatch_ContextCancellation(t *testing.T) {
	resp := QueryResponse{IDs: [][]string{{"doc1"}}}
	data, _ := json.Marshal(resp)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer server.Close()

	client := NewChromaDBClient(server.URL, "tenant", "db")
	client.Embedder = &slowEmbedder{delay: 500 * time.Millisecond}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, err := client.QueryBatch(ctx, "coll", []string{"query"}, 1)
	if err == nil {
		t.Error("Expected error for cancelled context, got nil")
	}

	if err != nil && !strings.Contains(err.Error(), "context deadline exceeded") && err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got: %v", err)
	}
}

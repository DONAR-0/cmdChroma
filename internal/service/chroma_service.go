package service

import (
	"fmt"

	client "github.com/donar0/cmdChroma/internal/client"
	"github.com/donar0/cmdChroma/internal/onnx"
)

// ChromaService handles business logic for ChromaDB operations.
type ChromaService struct {
	client   client.ChromaClientInterface
	embedder onnx.EmbedderInterface
}

// NewChromaService creates a new service with the given client and embedder.
func NewChromaService(c client.ChromaClientInterface, e onnx.EmbedderInterface) *ChromaService {
	return &ChromaService{
		client:   c,
		embedder: e,
	}
}

// TestConnection tests the connection to ChromaDB.
func (s *ChromaService) TestConnection() error {
	return s.client.TestConnection()
}

// GetTenant checks if the tenant exists.
func (s *ChromaService) GetTenant() (bool, error) {
	return s.client.GetTenant()
}

// ListDatabases lists all databases for the tenant.
func (s *ChromaService) ListDatabases() ([]client.Database, error) {
	return s.client.ListDatabases()
}

// ListCollections lists all collections in the database.
func (s *ChromaService) ListCollections() ([]client.Collection, error) {
	return s.client.ListCollections()
}

// AddDocuments adds documents to a collection with embeddings.
func (s *ChromaService) AddDocuments(collectionName string, docs []string, ids []string) error {
	if s.embedder == nil {
		return fmt.Errorf("embedder not available")
	}
	return s.client.AddBatch(collectionName, docs, ids)
}

// QueryDocuments queries documents in a collection.
func (s *ChromaService) QueryDocuments(collectionName string, queries []string, nResults int) (*client.QueryResponse, error) {
	if s.embedder == nil {
		return nil, fmt.Errorf("embedder not available")
	}
	return s.client.QueryBatch(collectionName, queries, nResults)
}

// IngestRecords ingests records from a file.
func (s *ChromaService) IngestRecords(collectionName, filePath string) error {
	// Implementation to be added
	return fmt.Errorf("not implemented")
}

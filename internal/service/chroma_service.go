package service

// Package service implements the business logic layer for cmdChroma.
// It orchestrates client operations, manages embedding generation, and
// provides high-level operations (add, query, ingest) that combine multiple
// client calls. The service layer is where transaction boundaries, batching,
// and error handling policies are defined.
import (
	"fmt"
	"log/slog"
	"strings"

	client "github.com/DONAR-0/cmdChroma/internal/client"
	"github.com/DONAR-0/cmdChroma/internal/errors"
	"github.com/DONAR-0/cmdChroma/internal/ingest"
	"github.com/DONAR-0/cmdChroma/internal/onnx"
)

// ============ Service Definition ============

// ChromaService handles business logic for ChromaDB operations.
// It coordinates between the client (HTTP API) and embedder (vector generation)
// to provide high-level operations like AddDocuments, QueryDocuments, and IngestRecords.
type ChromaService struct {
	// client is the underlying ChromaDB HTTP client. Must be non-nil.
	client client.ChromaClientInterface
	// embedder is used to generate embeddings for documents and queries.
	// Must be non-nil for operations that require embeddings.
	embedder onnx.EmbedderInterface
}

// NewChromaService creates a new service with the given client and embedder.
// The embedder is automatically injected into the client via SetEmbedder,
// enabling the client to perform embedding operations directly.
//
// Both c and e must be non-nil for full functionality. Passing a nil embedder
// will cause methods that require embeddings to return errors.
func NewChromaService(c client.ChromaClientInterface, e onnx.EmbedderInterface) *ChromaService {
	slog.Info("Creating ChromaService",
		"client_type", fmt.Sprintf("%T", c),
		"embedder_type", fmt.Sprintf("%T", e))

	// Inject embedder into the client via the interface method
	c.SetEmbedder(e)

	return &ChromaService{
		client:   c,
		embedder: e,
	}
}

// ============ Connection & Discovery ============

// TestConnection tests the connection to ChromaDB.
func (s *ChromaService) TestConnection() error {
	slog.Info("Testing ChromaDB connection")

	err := s.client.TestConnection()
	if err != nil {
		slog.Error("Connection test failed", "error", err)
	} else {
		slog.Info("Connection test successful")
	}

	return err
}

// GetTenant checks if the tenant exists.
func (s *ChromaService) GetTenant() (bool, error) {
	slog.Info("Checking tenant existence")

	exists, err := s.client.GetTenant()
	if err != nil {
		slog.Error("Tenant check failed", "error", err)
	} else {
		slog.Info("Tenant check completed", "exists", exists)
	}

	return exists, err
}

// ListDatabases lists all databases for the tenant.
func (s *ChromaService) ListDatabases() ([]client.Database, error) {
	slog.Info("Listing databases")

	databases, err := s.client.ListDatabases()
	if err != nil {
		slog.Error("Failed to list databases", "error", err)
		return nil, err
	}

	slog.Info("Databases listed", "count", len(databases))

	return databases, nil
}

// ListCollections lists all collections in the database.
func (s *ChromaService) ListCollections() ([]client.Collection, error) {
	slog.Info("Listing collections")

	collections, err := s.client.ListCollections()
	if err != nil {
		slog.Error("Failed to list collections", "error", err)
		return nil, err
	}

	slog.Info("Collections listed", "count", len(collections))

	return collections, nil
}

// ============ Document Operations ============

// AddDocuments adds documents to a collection with embeddings.
func (s *ChromaService) AddDocuments(collectionName string, docs []string, ids []string) error {
	if s.embedder == nil {
		err := errors.ErrEmbedderNotInitialized
		slog.Error("Cannot add documents: embedder not initialized", "error", err)

		return err
	}

	slog.Info("Adding documents",
		"collection", collectionName,
		"document_count", len(docs))
	// Resolve collection name (or ID) to a valid collection ID
	collectionID, err := s.client.ResolveCollectionID(collectionName)
	if err != nil {
		slog.Error("Failed to resolve collection", "collection", collectionName, "error", err)
		return fmt.Errorf("failed to resolve collection '%s': %w", collectionName, err)
	}

	err = s.client.AddBatchGeneric(collectionID, docs, ids, nil)
	if err != nil {
		slog.Error("Failed to add batch", "collection", collectionName, "batch_size", len(docs), "error", err)
	} else {
		slog.Info("Documents added successfully", "collection", collectionName, "count", len(docs))
	}

	return err
}

// UpsertDocuments upserts documents to a collection with embeddings (insert or update).
func (s *ChromaService) UpsertDocuments(collectionName string, docs []string, ids []string) error {
	if s.embedder == nil {
		err := errors.ErrEmbedderNotInitialized
		slog.Error("Cannot upsert documents: embedder not initialized", "error", err)

		return err
	}

	slog.Info("Upserting documents",
		"collection", collectionName,
		"document_count", len(docs))
	// Resolve collection name (or ID) to a valid collection ID
	collectionID, err := s.client.ResolveCollectionID(collectionName)
	if err != nil {
		slog.Error("Failed to resolve collection", "collection", collectionName, "error", err)
		return fmt.Errorf("failed to resolve collection '%s': %w", collectionName, err)
	}

	err = s.client.UpsertBatchGeneric(collectionID, docs, ids, nil)
	if err != nil {
		slog.Error("Failed to upsert batch", "collection", collectionName, "batch_size", len(docs), "error", err)
	} else {
		slog.Info("Documents upserted successfully", "collection", collectionName, "count", len(docs))
	}

	return err
}

// QueryDocuments queries documents in a collection.
func (s *ChromaService) QueryDocuments(collectionName string, queries []string, nResults int) (*client.QueryResponse, error) {
	if s.embedder == nil {
		err := errors.ErrEmbedderNotInitialized
		slog.Error("Cannot query: embedder not initialized", "error", err)

		return nil, err
	}

	slog.Info("Querying documents",
		"collection", collectionName,
		"query_count", len(queries),
		"n_results", nResults)
	// Resolve collection name (or ID) to a valid collection ID
	collectionID, err := s.client.ResolveCollectionID(collectionName)
	if err != nil {
		slog.Error("Failed to resolve collection", "collection", collectionName, "error", err)
		return nil, fmt.Errorf("failed to resolve collection '%s': %w", collectionName, err)
	}

	result, err := s.client.QueryBatch(collectionID, queries, nResults)
	if err != nil {
		slog.Error("Query failed", "collection", collectionName, "error", err)
		return nil, err
	}

	slog.Info("Query completed", "collection", collectionName, "results_count", len(result.IDs))

	return result, nil
}

// ============ Resource Management ============

// Close releases resources used by the service.
func (s *ChromaService) Close() {
	if s.embedder != nil {
		s.embedder.Close()
	}
}

// ============ Ingestion ============

// IngestRecords ingests records from a file (JSONL or Parquet) into the collection.
// It detects the file format, parses the file, extracts content and metadata,
// generates embeddings, and uploads in batches. collectionName can be a name or UUID.
// If cfg is nil, sensible defaults are used.
func (s *ChromaService) IngestRecords(collectionName, filePath string, cfg *ingest.Config) error {
	if s.embedder == nil {
		return errors.ErrEmbedderNotInitialized
	}

	// Resolve collection
	collectionID, err := s.client.ResolveCollectionID(collectionName)
	if err != nil {
		return fmt.Errorf("failed to resolve collection '%s': %w", collectionName, err)
	}

	// Use provided config or fall back to defaults
	if cfg == nil {
		cfg = ingest.DefaultConfig()
	}

	processor := ingest.NewProcessor(cfg)

	// Detect file format and stream records
	ext := getFileExt(filePath)

	var (
		records <-chan *ingest.Record
		errChan <-chan error
	)

	switch ext {
	case ".jsonl":
		records, errChan = processor.ProcessJSONL(filePath)
	case ".parquet":
		records, errChan = processor.ProcessParquet(filePath)
	default:
		return fmt.Errorf("unsupported file format: %s (supported: .jsonl, .parquet)", ext)
	}

	// Batch accumulation with progress tracking
	var (
		docs          []string
		ids           []string
		metas         []map[string]any
		batchIdx      int
		totalUploaded int
		progressN     = 10 // log progress every N documents processed
		nextProgress  = progressN
	)

	for record := range records {
		docs = append(docs, record.Content)
		ids = append(ids, record.ID)
		metas = append(metas, record.Metadata)
		batchIdx++

		// Current total processed (including current batch)
		currentTotal := totalUploaded + batchIdx

		// Progress update every N documents
		if currentTotal >= nextProgress && batchIdx < cfg.BatchSize {
			slog.Info("Progress", "total_processed", currentTotal, "batch_accumulated", batchIdx)

			nextProgress += progressN
		}

		if batchIdx >= cfg.BatchSize {
			if err := s.uploadBatch(collectionID, docs, ids, metas); err != nil {
				return fmt.Errorf("batch upload failed at document %d: %w", totalUploaded, err)
			}

			totalUploaded += len(docs)
			slog.Info("Batch uploaded", "batch_size", len(docs), "total_uploaded", totalUploaded)
			docs, ids, metas = nil, nil, nil
			batchIdx = 0
			nextProgress = totalUploaded + progressN // set next milestone
		}
	}

	// Final batch
	if len(docs) > 0 {
		if err := s.uploadBatch(collectionID, docs, ids, metas); err != nil {
			return fmt.Errorf("final batch upload failed at document %d: %w", totalUploaded, err)
		}

		totalUploaded += len(docs)
		slog.Info("Final batch uploaded", "batch_size", len(docs), "total_uploaded", totalUploaded)
	}

	// Check for errors from processor
	if err, ok := <-errChan; ok && err != nil {
		return fmt.Errorf("ingestion error: %w", err)
	}

	slog.Info("Ingestion complete", "total_documents", totalUploaded)

	return nil
}

// ============ Private Helpers ============

// getFileExt returns the lowercase file extension for format detection.
func getFileExt(filePath string) string {
	// Find the last dot to handle files with dots in their name
	for i := len(filePath) - 1; i >= 0; i-- {
		if filePath[i] == '.' {
			return strings.ToLower(filePath[i:])
		}

		if filePath[i] == '/' || filePath[i] == '\\' {
			break
		}
	}

	return ""
}

// uploadBatch uploads a batch of documents to Chroma.
// The client's AddBatchGeneric generates embeddings internally.
func (s *ChromaService) uploadBatch(collectionID string, docs, ids []string, metas []map[string]any) error {
	if len(docs) == 0 {
		return nil
	}

	return s.client.AddBatchGeneric(collectionID, docs, ids, metas)
}

package service

import (
	"fmt"
	"log/slog"

	client "github.com/donar0/cmdChroma/internal/client"
	"github.com/donar0/cmdChroma/internal/errors"
	"github.com/donar0/cmdChroma/internal/ingest"
	"github.com/donar0/cmdChroma/internal/onnx"
)

// ChromaService handles business logic for ChromaDB operations.
type ChromaService struct {
	client   client.ChromaClientInterface
	embedder onnx.EmbedderInterface
}

// NewChromaService creates a new service with the given client and embedder.
// If the client is a concrete *client.ChromaClient, the embedder is injected into it.
func NewChromaService(c client.ChromaClientInterface, e onnx.EmbedderInterface) *ChromaService {
	slog.Info("Creating ChromaService",
		"client_type", fmt.Sprintf("%T", c),
		"embedder_type", fmt.Sprintf("%T", e))
	// Inject embedder into the client if possible
	if ch, ok := c.(*client.ChromaClient); ok {
		ch.Embedder = e
	}

	return &ChromaService{
		client:   c,
		embedder: e,
	}
}

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

// Close releases resources used by the service.
func (s *ChromaService) Close() {
	if s.embedder != nil {
		s.embedder.Close()
	}
}

// IngestRecords ingests records from a JSONL file into the collection.
// It parses the file, extracts content and metadata, generates embeddings,
// and uploads in batches. collectionName can be a name or UUID.
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

	// Stream records
	records, errChan := processor.ProcessJSONL(filePath)

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

// uploadBatch uploads a batch of documents to Chroma.
// The client's AddBatchGeneric generates embeddings internally.
func (s *ChromaService) uploadBatch(collectionID string, docs, ids []string, metas []map[string]any) error {
	if len(docs) == 0 {
		return nil
	}

	return s.client.AddBatchGeneric(collectionID, docs, ids, metas)
}

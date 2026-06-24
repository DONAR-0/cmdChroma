package service

// Package service implements the business logic layer for cmdChroma.
// It orchestrates client operations, manages embedding generation, and
// provides high-level operations (add, query, ingest) that combine multiple
// client calls. The service layer is where transaction boundaries, batching,
// and error handling policies are defined.
import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

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
func NewChromaService(c client.ChromaClientInterface, e onnx.EmbedderInterface) *ChromaService {
	c.SetEmbedder(e)

	return &ChromaService{
		client:   c,
		embedder: e,
	}
}

// ============ Connection & Discovery ============

// TestConnection tests the connection to ChromaDB. ctx propagates to the HTTP
// round trip.
func (s *ChromaService) TestConnection(ctx context.Context) error {
	err := s.client.TestConnection(ctx)
	if err != nil {
		slog.Error("Connection test failed", "error", err)
	}

	return err
}

// GetTenant checks if the tenant exists. ctx propagates to the HTTP round trip.
func (s *ChromaService) GetTenant(ctx context.Context) (bool, error) {
	exists, err := s.client.GetTenant(ctx)
	if err != nil {
		slog.Error("Tenant check failed", "error", err)
	}

	return exists, err
}

// ListDatabases lists all databases for the tenant. ctx propagates to the HTTP
// round trip.
func (s *ChromaService) ListDatabases(ctx context.Context) ([]client.Database, error) {
	databases, err := s.client.ListDatabases(ctx)
	if err != nil {
		slog.Error("Failed to list databases", "error", err)
		return nil, err
	}

	return databases, nil
}

// ListCollections lists all collections in the database. ctx propagates to the
// HTTP round trip.
func (s *ChromaService) ListCollections(ctx context.Context) ([]client.Collection, error) {
	collections, err := s.client.ListCollections(ctx)
	if err != nil {
		slog.Error("Failed to list collections", "error", err)
		return nil, err
	}

	return collections, nil
}

// ============ Document Operations ============

// AddDocuments adds documents to a collection with embeddings. ctx propagates
// to all client HTTP calls.
func (s *ChromaService) AddDocuments(ctx context.Context, collectionName string, docs []string, ids []string) error {
	if s.embedder == nil {
		return errors.ErrEmbedderNotInitialized
	}

	collectionID, err := s.client.ResolveCollectionID(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to resolve collection '%s': %w", collectionName, err)
	}

	err = s.client.AddBatchGeneric(ctx, collectionID, docs, ids, nil)
	if err != nil {
		slog.Error("Failed to add batch", "collection", collectionName, "batch_size", len(docs), "error", err)
	}

	return err
}

// UpsertDocuments upserts documents to a collection with embeddings (insert
// or update). ctx propagates to all client HTTP calls.
func (s *ChromaService) UpsertDocuments(ctx context.Context, collectionName string, docs []string, ids []string) error {
	if s.embedder == nil {
		return errors.ErrEmbedderNotInitialized
	}

	collectionID, err := s.client.ResolveCollectionID(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to resolve collection '%s': %w", collectionName, err)
	}

	err = s.client.UpsertBatchGeneric(ctx, collectionID, docs, ids, nil)
	if err != nil {
		slog.Error("Failed to upsert batch", "collection", collectionName, "batch_size", len(docs), "error", err)
	}

	return err
}

// QueryDocuments queries documents in a collection. ctx propagates to all
// client HTTP calls.
func (s *ChromaService) QueryDocuments(ctx context.Context, collectionName string, queries []string, nResults int) (*client.QueryResponse, error) {
	if s.embedder == nil {
		return nil, errors.ErrEmbedderNotInitialized
	}

	collectionID, err := s.client.ResolveCollectionID(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve collection '%s': %w", collectionName, err)
	}

	result, err := s.client.QueryBatch(ctx, collectionID, queries, nResults)
	if err != nil {
		slog.Error("Query failed", "collection", collectionName, "error", err)
		return nil, err
	}

	return result, nil
}

// GetDocuments returns documents from a collection by name or ID. It handles
// collection name→ID resolution internally. ctx propagates to all client
// HTTP calls.
func (s *ChromaService) GetDocuments(ctx context.Context, collectionName string) (*client.GetRecordsResponse, error) {
	collectionID, err := s.client.ResolveCollectionID(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve collection '%s': %w", collectionName, err)
	}

	result, err := s.client.ListDocuments(ctx, collectionID)
	if err != nil {
		slog.Error("Failed to list documents", "collection", collectionName, "error", err)
		return nil, fmt.Errorf("failed to list documents from '%s': %w", collectionName, err)
	}

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
// If cfg is nil, sensible defaults are used. ctx propagates to all client HTTP
// calls so cancellation aborts the in-flight ingest.
func (s *ChromaService) IngestRecords(ctx context.Context, collectionName, filePath string, cfg *ingest.Config) error {
	if s.embedder == nil {
		return errors.ErrEmbedderNotInitialized
	}

	collectionID, err := s.client.ResolveCollectionID(ctx, collectionName)
	if err != nil {
		return fmt.Errorf("failed to resolve collection '%s': %w", collectionName, err)
	}

	// Use provided config or fall back to defaults
	if cfg == nil {
		cfg = ingest.DefaultConfig()
	}

	processor := ingest.NewProcessor(cfg).WithContext(ctx)

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
	case ".csv":
		records, errChan = processor.ProcessCSV(filePath)
	default:
		return fmt.Errorf("unsupported file format: %s (supported: .jsonl, .parquet, .csv)", ext)
	}

	// Batch accumulation with progress tracking
	var (
		docs          []string
		ids           []string
		metas         []map[string]any
		batchIdx      int
		totalUploaded int
		totalBatches  int
		progressN     = 10 // log progress every N documents processed
		nextProgress  = progressN
		seenIDs       map[string]int // dedup tracking; lazily allocated
		startTime     = time.Now()
	)

	if cfg.DedupMode != "none" {
		seenIDs = make(map[string]int)
	}

	emitProgress := func(processed int, done bool) {
		if cfg.OnProgress != nil {
			cfg.OnProgress(ingest.ProgressInfo{
				Processed: processed,
				Total:     cfg.Total,
				BatchSize: cfg.BatchSize,
				Batches:   totalBatches,
				Elapsed:   time.Since(startTime).Nanoseconds(),
				Done:      done,
			})
		}
	}

	for record := range records {
		// Dedup check
		if seenIDs != nil {
			if _, exists := seenIDs[record.ID]; exists {
				switch cfg.DedupMode {
				case "warn":
					slog.Warn("Duplicate ID skipped", "id", record.ID)
				case "skip":
					// silent skip
				}

				continue
			}
		}

		if seenIDs != nil {
			seenIDs[record.ID] = 1
		}

		docs = append(docs, record.Content)
		ids = append(ids, record.ID)
		metas = append(metas, record.Metadata)
		batchIdx++

		// Current total processed (including current batch)
		currentTotal := totalUploaded + batchIdx

		// Progress update every N documents
		if currentTotal >= nextProgress && batchIdx < cfg.BatchSize {
			slog.Debug("Progress", "total_processed", currentTotal, "batch_accumulated", batchIdx)
			emitProgress(currentTotal, false)

			nextProgress += progressN
		}

		if batchIdx >= cfg.BatchSize {
			if err := s.uploadBatch(ctx, collectionID, docs, ids, metas, cfg.Upsert); err != nil {
				return fmt.Errorf("batch upload failed at document %d: %w", totalUploaded, err)
			}

			totalUploaded += len(docs)
			totalBatches++

			slog.Debug("Batch uploaded", "batch_size", len(docs), "total_uploaded", totalUploaded)
			docs, ids, metas = nil, nil, nil
			batchIdx = 0
			nextProgress = totalUploaded + progressN // set next milestone
		}
	}

	// Final batch
	if len(docs) > 0 {
		if err := s.uploadBatch(ctx, collectionID, docs, ids, metas, cfg.Upsert); err != nil {
			return fmt.Errorf("final batch upload failed at document %d: %w", totalUploaded, err)
		}

		totalUploaded += len(docs)
		totalBatches++

		slog.Debug("Final batch uploaded", "batch_size", len(docs), "total_uploaded", totalUploaded)
	}

	emitProgress(totalUploaded, true)

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
// If upsert is true, UpsertBatchGeneric is used instead to update existing documents.
func (s *ChromaService) uploadBatch(ctx context.Context, collectionID string, docs, ids []string, metas []map[string]any, upsert bool) error {
	if len(docs) == 0 {
		return nil
	}

	if len(docs) != len(ids) {
		return fmt.Errorf("batch slice length mismatch: docs=%d ids=%d", len(docs), len(ids))
	}

	if metas == nil {
		metas = []map[string]any{}
	}

	if upsert {
		return s.client.UpsertBatchGeneric(ctx, collectionID, docs, ids, metas)
	}

	return s.client.AddBatchGeneric(ctx, collectionID, docs, ids, metas)
}

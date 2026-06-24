package ingest

// Package ingest provides streaming ingestion of documents from files (JSONL, Parquet)
// into ChromaDB. It handles parsing, embedding generation, and batch uploads with
// progress tracking. The package is designed for memory-efficient processing of
// large datasets via Go channels.
import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reflect"
	"strings"

	"github.com/parquet-go/parquet-go"
	"github.com/tmc/langchaingo/textsplitter"

	"github.com/DONAR-0/cmdChroma/internal"
)

// ============ Data Types ============

// Record represents a document to be ingested, with id, content, and optional metadata.
type Record struct {
	// ID is the unique identifier for the document.
	ID string
	// Content is the text content to embed and store.
	Content string
	// Metadata holds arbitrary key-value pairs for filtering and annotations.
	Metadata map[string]any
}

// ProgressInfo holds the current state of an ingestion operation for UI display.
type ProgressInfo struct {
	Processed int
	Total     int // 0 if unknown
	BatchSize int
	Batches   int   // batches uploaded so far
	Elapsed   int64 // nanoseconds
	Done      bool
}

// ProgressFunc is a callback for ingestion progress updates.
type ProgressFunc func(ProgressInfo)

// Config controls the ingestion pipeline behavior.
type Config struct {
	// BatchSize is the number of documents sent per HTTP request.
	// Larger batches reduce overhead but increase memory usage.
	BatchSize int
	// ContentField specifies the JSON field containing document text.
	// Defaults to "text" if empty.
	ContentField string
	// IDField specifies the JSON field to use as document ID.
	// Defaults to "id" if empty.
	IDField string
	// MetadataFields lists specific fields to extract as metadata.
	// If empty and AllMetadata is false, no metadata is extracted.
	MetadataFields []string // specific fields to extract
	// AllMetadata, when true, extracts all JSON fields except ContentField, IDField, and ExcludeFields.
	AllMetadata bool // extract all fields except content/id
	// ExcludeFields lists fields to exclude from metadata when AllMetadata is true.
	ExcludeFields []string
	// Limit restricts the maximum number of records to ingest.
	// Zero means no limit.
	Limit int // max records to ingest, 0 = unlimited

	// ChunkSize is the maximum number of characters per chunk.
	// If 0, no splitting is performed.
	ChunkSize int
	// ChunkOverlap is the number of characters to overlap between chunks.
	ChunkOverlap int

	// AutoID ignores user-provided IDs and always generates content-hash IDs.
	AutoID bool
	// DedupMode controls handling of duplicate IDs: "none", "warn", or "skip".
	// "warn" logs a warning and skips the duplicate; "skip" silently skips.
	DedupMode string
	// Upsert uses the upsert endpoint instead of add, so existing IDs are updated.
	Upsert bool
	// Total is the known total number of records (0 if unknown).
	// Used for progress display.
	Total int
	// OnProgress is an optional callback for progress updates during ingestion.
	OnProgress ProgressFunc
}

// Processor handles the ingestion workflow: reading, parsing, embedding, and uploading.
type Processor struct {
	cfg *Config
	ctx context.Context // for cancellation; nil means not cancelable
}

// WithContext returns a copy of the processor with a context for cancellation.
// The context is used by ProcessJSONL and ProcessParquet to abort early.
func (p *Processor) WithContext(ctx context.Context) *Processor {
	p.ctx = ctx
	return p
}

// Primitive reflect.Kinds that can be kept as-is
var primitiveTypes = map[reflect.Kind]bool{
	reflect.String: true,
	reflect.Int:    true, reflect.Int8: true, reflect.Int16: true, reflect.Int32: true, reflect.Int64: true,
	reflect.Uint: true, reflect.Uint8: true, reflect.Uint16: true, reflect.Uint32: true, reflect.Uint64: true,
	reflect.Float32: true, reflect.Float64: true,
	reflect.Bool: true,
}

// ============ Constructors ============

func DefaultConfig() *Config {
	return &Config{
		BatchSize:    100,
		ContentField: "text",
		IDField:      "id",
	}
}

func NewProcessor(cfg *Config) *Processor {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	return &Processor{cfg: cfg}
}

// ============ Public Processing Methods ============

// ProcessJSONL reads a JSONL file and streams records through the provided channel.
// The channel is closed when processing is complete or an unrecoverable error occurs.
func (p *Processor) ProcessJSONL(filePath string) (<-chan *Record, <-chan error) {
	records := make(chan *Record)
	errChan := make(chan error, 1)

	go func() {
		defer close(records)
		defer close(errChan)

		file, err := os.Open(filePath)
		if err != nil {
			errChan <- fmt.Errorf("failed to open file: %w", err)
			return
		}
		defer internal.CheckDefer(file.Close)

		scanner := bufio.NewScanner(file)

		const maxCapacity = 1 * 1024 * 1024
		scanner.Buffer(make([]byte, maxCapacity), maxCapacity)

		var count int

		for scanner.Scan() {
			var rec map[string]any
			if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
				slog.Error("Failed to parse line", "error", err)
				continue
			}

			recordsList, err := p.extractRecord(rec)
			if err != nil {
				slog.Error("Skipping record", "error", err)
				continue
			}

			if len(recordsList) == 0 {
				continue
			}

			for _, record := range recordsList {
				if p.ctx != nil {
					select {
					case records <- record:
					case <-p.ctx.Done():
						return
					}
				} else {
					records <- record
				}
			}

			count++

			// Check limit
			if p.cfg.Limit > 0 && count >= p.cfg.Limit {
				slog.Info("Limit reached", "count", count)
				break
			}
		}

		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("scanner error: %w", err)
		}

		slog.Info("Processing complete", "total", count)
	}()

	return records, errChan
}

// ProcessJSONLFull reads entire file and returns all records (for small files).
func (p *Processor) ProcessJSONLFull(filePath string) ([]*Record, error) {
	recordsChan, errChan := p.ProcessJSONL(filePath)

	var records []*Record
	for rec := range recordsChan {
		records = append(records, rec)
	}

	if err, ok := <-errChan; ok && err != nil {
		return records, err
	}

	return records, nil
}

// ProcessParquet reads a Parquet file and streams records through the provided channel.
// It uses parquet-go's GenericReader to read rows as map[string]any, then converts
// each row to a Record using the same extractRecord logic as JSONL.
func (p *Processor) ProcessParquet(filePath string) (<-chan *Record, <-chan error) {
	records := make(chan *Record)
	errChan := make(chan error, 1)

	go func() {
		defer close(records)
		defer close(errChan)

		f, err := os.Open(filePath)
		if err != nil {
			errChan <- fmt.Errorf("failed to open parquet file: %w", err)
			return
		}
		defer internal.CheckDefer(f.Close)

		// Use GenericReader to read rows as map[string]any
		reader := parquet.NewGenericReader[any](f)

		defer func() {
			if err := reader.Close(); err != nil {
				slog.Error("Failed to close parquet reader", "error", err)
			}
		}()

		// Determine batch size for reading
		batchSize := p.cfg.BatchSize
		if batchSize <= 0 {
			batchSize = 100
		}

		batch := make([]any, batchSize)

		var total int

		for {
			n, err := reader.Read(batch)
			if n > 0 {
				for _, row := range batch[:n] {
					rowMap, ok := row.(map[string]any)
					if !ok {
						slog.Error("Skipping record with unexpected type", "type", fmt.Sprintf("%T", row))
						continue
					}

					recordsList, err := p.extractRecord(rowMap)
					if err != nil {
						slog.Error("Skipping record", "error", err)
						continue
					}

					if len(recordsList) == 0 {
						continue
					}

					for _, rec := range recordsList {
						if p.ctx != nil {
							select {
							case records <- rec:
							case <-p.ctx.Done():
								return
							}
						} else {
							records <- rec
						}
					}

					total++

					if p.cfg.Limit > 0 && total >= p.cfg.Limit {
						slog.Info("Limit reached", "count", total)
						return
					}
				}
			}

			if err != nil {
				if err == io.EOF {
					break
				}

				errChan <- fmt.Errorf("parquet read error: %w", err)

				return
			}
		}

		slog.Info("Processing complete", "total", total)
	}()

	return records, errChan
}

// ParquetRowCount returns the number of rows in a parquet file.
// Returns 0 if the row count cannot be determined.
func ParquetRowCount(filePath string) int {
	f, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer func() {
		_ = f.Close()
	}()

	info, err := f.Stat()
	if err != nil {
		return 0
	}

	pf, err := parquet.OpenFile(f, info.Size())
	if err != nil {
		return 0
	}

	return int(pf.NumRows())
}

// ============ Private Helpers ============

// getNestedValue retrieves nested values using dot notation.
func getNestedValue(m map[string]any, path string) any {
	parts := strings.Split(path, ".")

	var current any = m
	for _, part := range parts {
		if next, ok := current.(map[string]any); ok {
			current = next[part]
		} else {
			return nil
		}
	}

	return current
}

// stringifyIfComplex converts non-primitive values to strings while preserving primitive types
// and recursively flattening nested maps into the parent with underscore-separated keys.
func (p *Processor) stringifyIfComplex(value any) any {
	if value == nil {
		return nil
	}

	rt := reflect.TypeOf(value)
	// Pointers: dereference and check underlying type
	for rt.Kind() == reflect.Pointer {
		v := reflect.ValueOf(value)
		if v.IsNil() {
			return nil
		}

		value = v.Elem().Interface()
		rt = reflect.TypeOf(value)
	}

	// Recursively flatten nested maps into the parent (foo_bar becomes foo.bar)
	// to preserve structure while meeting ChromaDB's flat-key requirement.
	if m, ok := value.(map[string]any); ok {
		result := make(map[string]any)
		for k, v := range m {
			result[k] = p.stringifyIfComplex(v)
		}

		return result
	}

	if rt == nil {
		return nil
	}
	// Convert slices/arrays to string representation.
	if rt.Kind() == reflect.Slice || rt.Kind() == reflect.Array {
		return fmt.Sprintf("%v", value)
	}

	// Check if primitive
	if primitiveTypes[rt.Kind()] {
		return value
	}

	// Complex type - convert to string
	return fmt.Sprintf("%v", value)
}

// extractRecord converts a raw Parquet/JSON row into Records, splitting
// content into chunks if ChunkSize is configured.
func (p *Processor) extractRecord(raw map[string]any) ([]*Record, error) {
	contentVal := getNestedValue(raw, p.cfg.ContentField)
	if contentVal == nil {
		return nil, nil
	}

	content := fmt.Sprintf("%v", contentVal)

	baseID := p.resolveID(raw, content)

	meta := p.collectMetadata(raw)

	if p.cfg.ChunkSize <= 0 {
		return []*Record{{ID: baseID, Content: content, Metadata: meta}}, nil
	}

	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(p.cfg.ChunkSize),
		textsplitter.WithChunkOverlap(p.cfg.ChunkOverlap),
	)

	chunks, err := splitter.SplitText(content)
	if err != nil {
		return nil, fmt.Errorf("text splitting failed: %w", err)
	}

	records := make([]*Record, 0, len(chunks))
	for i, chunk := range chunks {
		records = append(records, &Record{
			ID:       fmt.Sprintf("%s_chunk%d", baseID, i),
			Content:  chunk,
			Metadata: meta,
		})
	}

	return records, nil
}

// resolveID returns the record ID from the configured field, or a content hash.
func (p *Processor) resolveID(raw map[string]any, content string) string {
	if p.cfg.AutoID {
		return p.contentHash(content)
	}

	if idVal := getNestedValue(raw, p.cfg.IDField); idVal != nil {
		return fmt.Sprintf("%v", idVal)
	}

	return p.contentHash(content)
}

// contentHash returns a deterministic hex hash of the given string.
func (p *Processor) contentHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:12])
}

// collectMetadata extracts metadata from a raw row based on the config.
func (p *Processor) collectMetadata(raw map[string]any) map[string]any {
	meta := make(map[string]any)

	if p.cfg.AllMetadata {
		excluded := map[string]bool{
			p.cfg.ContentField: true,
			p.cfg.IDField:      true,
		}
		for _, f := range p.cfg.ExcludeFields {
			excluded[f] = true
		}

		for k, v := range raw {
			if !excluded[k] {
				meta[k] = p.stringifyIfComplex(v)
			}
		}

		return meta
	}

	for _, k := range p.cfg.MetadataFields {
		if v, ok := raw[k]; ok {
			meta[k] = p.stringifyIfComplex(v)
		}
	}

	return meta
}

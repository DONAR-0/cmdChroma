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

			recs, err := p.extractRecord(rec)
			if err != nil {
				slog.Error("Skipping record", "error", err)
				continue
			}

			if len(recs) == 0 {
				continue
			}

			for _, record := range recs {
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

// processParquetReader reads rows from an already-configured parquet reader and
// streams records through the provided channel. It handles schema detection and
// auto-fallback for content/id fields.
func (p *Processor) processParquetReader(reader *parquet.GenericReader[any], records chan<- *Record, errChan chan<- error) {
	batchSize := p.cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	batch := make([]any, batchSize)

	var (
		total        int
		schemaLogged bool
		contentField string
	)

	for {
		n, err := reader.Read(batch)
		if n > 0 {
			for _, row := range batch[:n] {
				rowMap, ok := row.(map[string]any)
				if !ok {
					slog.Error("Skipping record with unexpected type", "type", fmt.Sprintf("%T", row))
					continue
				}

				// Log schema from first row
				if !schemaLogged {
					keys := make([]string, 0, len(rowMap))
					for k := range rowMap {
						keys = append(keys, k)
					}

					slog.Info("Parquet schema", "columns", keys)

					schemaLogged = true

					// Auto-detect content field if configured field yields no results
					contentField = p.detectContentField(rowMap)
					if contentField != p.cfg.ContentField {
						slog.Info("Auto-detected content field", "from", p.cfg.ContentField, "to", contentField)
					}
				}

				recs, err := p.extractRecordWithField(rowMap, contentField, p.cfg.IDField)
				if err != nil {
					slog.Error("Skipping record", "error", err)
					continue
				}

				if len(recs) == 0 {
					continue
				}

				for _, rec := range recs {
					records <- rec
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
}

// detectContentField tries to find a suitable content column from the first row.
// If the configured field is missing or has a nil value, it falls back to the
// longest string column, preferring ones with common content-related names.
func (p *Processor) detectContentField(firstRow map[string]any) string {
	if cfg := p.cfg.ContentField; cfg != "" {
		if v, ok := firstRow[cfg]; ok && v != nil {
			return cfg
		}
	}

	// Score candidate fields: prefer longer text and content-like names
	type candidate struct {
		name  string
		score int
		len   int
	}

	var candidates []candidate

	contentKeywords := []string{"text", "content", "body", "description", "answer", "response", "output", "question", "instruction", "utterance", "dialogue", "story"}

	for k, v := range firstRow {
		s, ok := v.(string)
		if !ok {
			continue
		}

		score := 0

		for _, kw := range contentKeywords {
			if strings.Contains(strings.ToLower(k), kw) {
				score++
			}
		}

		candidates = append(candidates, candidate{name: k, score: score, len: len(s)})
	}

	if len(candidates) == 0 {
		return p.cfg.ContentField
	}

	// Sort by score desc, then by length desc
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.score > best.score || (c.score == best.score && c.len > best.len) {
			best = c
		}
	}

	if best.score > 0 || best.len > 0 {
		return best.name
	}

	return p.cfg.ContentField
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

		// Check Parquet magic bytes before creating reader to avoid panic on invalid files
		magic := make([]byte, 4)
		if _, err := f.Read(magic); err != nil || string(magic) != "PAR1" {
			errChan <- fmt.Errorf("invalid parquet file: missing PAR1 magic header")
			return
		}

		if _, err := f.Seek(0, 0); err != nil {
			errChan <- fmt.Errorf("failed to seek parquet file: %w", err)
			return
		}

		// Use GenericReader to read rows as map[string]any
		reader := parquet.NewGenericReader[any](f)

		defer func() {
			if err := reader.Close(); err != nil {
				slog.Error("Failed to close parquet reader", "error", err)
			}
		}()

		p.processParquetReader(reader, records, errChan)
	}()

	return records, errChan
}

// extractRecordWithField converts raw JSON/Parquet map into a Record using explicit field names.
func (p *Processor) extractRecordWithField(raw map[string]any, contentField, idField string) ([]*Record, error) {
	contentVal := getNestedValue(raw, contentField)
	if contentVal == nil {
		return nil, nil // skip records without content
	}

	rawContent := fmt.Sprintf("%v", contentVal)

	// Generate or extract ID
	var id string

	if p.cfg.AutoID {
		hash := sha256.Sum256([]byte(rawContent))
		id = hex.EncodeToString(hash[:12])
	} else if idVal := getNestedValue(raw, idField); idVal != nil {
		id = fmt.Sprintf("%v", idVal)
	} else {
		hash := sha256.Sum256([]byte(rawContent))
		id = hex.EncodeToString(hash[:12])
	}

	// Extract metadata
	meta := make(map[string]any)

	isExcluded := func(k string) bool {
		if k == contentField || k == idField {
			return true
		}

		for _, ex := range p.cfg.ExcludeFields {
			if k == ex {
				return true
			}
		}

		return false
	}

	if p.cfg.AllMetadata || len(p.cfg.MetadataFields) == 0 {
		for k, v := range raw {
			if !isExcluded(k) {
				meta[k] = p.stringifyIfComplex(v)
			}
		}
	} else {
		for _, k := range p.cfg.MetadataFields {
			if v, exists := raw[k]; exists {
				meta[k] = p.stringifyIfComplex(v)
			}
		}
	}

	if p.cfg.ChunkSize <= 0 || len(rawContent) <= p.cfg.ChunkSize {
		return []*Record{{ID: id, Content: rawContent, Metadata: meta}}, nil
	}

	return p.chunkContent(id, rawContent, meta), nil
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

// chunkContent splits content into overlapping chunks.
func (p *Processor) chunkContent(baseID, content string, meta map[string]any) []*Record {
	var records []*Record

	chunkSize := p.cfg.ChunkSize
	overlap := p.cfg.ChunkOverlap
	start := 0
	chunkIdx := 0

	for start < len(content) {
		end := start + chunkSize
		if end > len(content) {
			end = len(content)
		}

		chunkID := baseID
		if chunkIdx > 0 {
			chunkID = fmt.Sprintf("%s_chunk_%d", baseID, chunkIdx)
		}

		records = append(records, &Record{
			ID:       chunkID,
			Content:  content[start:end],
			Metadata: meta,
		})

		if end >= len(content) {
			break
		}

		start = end - overlap
		if start < 0 {
			start = 0
		}

		chunkIdx++
	}

	return records
}

// ParquetRowCount returns the number of rows in a Parquet file.
// Returns 0 if the file cannot be opened or read.
func ParquetRowCount(filePath string) int {
	f, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer func() { _ = f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		return 0
	}

	pf, err := parquet.OpenFile(f, fi.Size())
	if err != nil {
		return 0
	}

	return int(pf.NumRows())
}

// extractRecord converts raw JSON/Parquet map into Records using the configured field names.
// Returns multiple records when chunking is enabled.
func (p *Processor) extractRecord(raw map[string]any) ([]*Record, error) {
	return p.extractRecordWithField(raw, p.cfg.ContentField, p.cfg.IDField)
}

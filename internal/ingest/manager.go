package ingest

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/donar0/cmdChroma/internal"
)

type Record struct {
	ID       string
	Content  string
	Metadata map[string]any
}

type Config struct {
	BatchSize      int
	ContentField   string
	IDField        string
	MetadataFields []string // specific fields to extract
	AllMetadata    bool     // extract all fields except content/id
	Limit          int      // max records to ingest, 0 = unlimited
}

func DefaultConfig() *Config {
	return &Config{
		BatchSize:    100,
		ContentField: "text",
		IDField:      "id",
	}
}

type Processor struct {
	cfg *Config
}

func NewProcessor(cfg *Config) *Processor {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	return &Processor{cfg: cfg}
}

// ProcessJSONL reads a JSONL file and streams records through the provided channel.
// The channel is closed when processing is complete or an unrecoverable error occurs.
func (p *Processor) ProcessJSONL(filePath string) (<-chan *Record, <-chan error) {
	records := make(chan *Record)
	errChan := make(chan error, 1)

	go func() {
		defer close(records)
		defer close(errChan)

		file, err := os.Open(filePath)
		if err != nil && err != io.EOF {
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

			record, err := p.extractRecord(rec)
			if err != nil {
				slog.Error("Skipping record", "error", err)
				continue
			}

			if record == nil {
				continue
			}

			records <- record

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

// extractRecord converts raw JSON map into a Record.
func (p *Processor) extractRecord(raw map[string]any) (*Record, error) {
	contentVal := getNestedValue(raw, p.cfg.ContentField)
	if contentVal == nil {
		return nil, nil // skip records without content
	}

	content := fmt.Sprintf("%v", contentVal)

	// Generate or extract ID
	var id string

	idVal := getNestedValue(raw, p.cfg.IDField)
	if idVal != nil {
		id = fmt.Sprintf("%v", idVal)
	} else {
		// Deterministic hash from content
		hash := sha256.Sum256([]byte(content))
		id = hex.EncodeToString(hash[:12])
	}

	// Extract metadata
	meta := make(map[string]any)

	if p.cfg.AllMetadata {
		for k, v := range raw {
			if k != p.cfg.ContentField && k != p.cfg.IDField {
				meta[k] = v
			}
		}
	} else {
		for _, k := range p.cfg.MetadataFields {
			if v, exists := raw[k]; exists {
				meta[k] = v
			}
		}
	}

	return &Record{
		ID:       id,
		Content:  content,
		Metadata: meta,
	}, nil
}

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

// extractArrowValue safely extracts primitive values from Arrow column arrays.
func extractArrowValue(arr arrow.Array, row int) any {
	if arr.IsNull(row) {
		return nil
	}

	// Arrow enforces strong typing at the array level.
	// We type-switch on the underlying array implementation to extract the raw Go value.
	switch a := arr.(type) {
	case *array.String:
		return a.Value(row)
	case *array.LargeString:
		return a.Value(row)
	case *array.Int32:
		return a.Value(row)
	case *array.Int64:
		return a.Value(row)
	case *array.Float32:
		return a.Value(row)
	case *array.Float64:
		return a.Value(row)
	case *array.Boolean:
		return a.Value(row)
	case *array.Binary:
		return string(a.Value(row))
	case *array.LargeBinary:
		return string(a.Value(row))
	default:
		// For complex types like Structs or Lists (Thrift Type 14),
		// we fallback to Arrow's internal string representation of that cell.
		return fmt.Sprintf("%v", arr)
	}
}

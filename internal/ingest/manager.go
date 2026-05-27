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
	"reflect"
	"strings"

	"github.com/donar0/cmdChroma/internal"
	"github.com/parquet-go/parquet-go"
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

// primitiveTypes holds a set of reflect.Kind values that are considered primitive
var primitiveTypes = map[reflect.Kind]bool{
	reflect.String: true,
	reflect.Int:    true, reflect.Int8: true, reflect.Int16: true, reflect.Int32: true, reflect.Int64: true,
	reflect.Uint: true, reflect.Uint8: true, reflect.Uint16: true, reflect.Uint32: true, reflect.Uint64: true,
	reflect.Float32: true, reflect.Float64: true,
	reflect.Bool: true,
}

// stringifyIfComplex converts non-primitive values to strings while preserving primitive types
func (p *Processor) stringifyIfComplex(value any) any {
	if value == nil {
		return nil
	}

	rt := reflect.TypeOf(value)
	// Pointers: dereference and check underlying type
	for rt.Kind() == reflect.Ptr {
		if value == nil {
			return nil
		}

		value = reflect.ValueOf(value).Elem().Interface()
		rt = reflect.TypeOf(value)
	}
	// Check if primitive
	if primitiveTypes[rt.Kind()] {
		return value
	}
	// Complex type - convert to string
	return fmt.Sprintf("%v", value)
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

					rec, err := p.extractRecord(rowMap)
					if err != nil {
						slog.Error("Skipping record", "error", err)
						continue
					}

					if rec == nil {
						continue
					}

					records <- rec

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

package ingest

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/parquet-go/parquet-go"

	"github.com/donar0/cmdChroma/internal"
)

// Record represents a single data record ready for ingestion.
type Record struct {
	ID       string
	Content  string
	Metadata map[string]any
}

// Config configures the ingestion behavior.
type Config struct {
	BatchSize      int
	ContentField   string
	IDField        string
	MetadataFields []string // specific fields to extract
	AllMetadata    bool     // extract all fields except content/id
	Limit          int      // max records to ingest, 0 = unlimited

	// Parquet-specific: column names (overrides ContentField/IDField for parquet)
	ParquetIDColumn   string
	ParquetTextColumn string
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		BatchSize:    100,
		ContentField: "text",
		IDField:      "id",
	}
}

// Processor processes a file and yields records.
type Processor struct {
	cfg *Config
}

// NewProcessor creates a new ingestion processor.
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

// ParquetConfig defines column mappings for Parquet ingestion.
type ParquetConfig struct {
	IDColumn   string
	TextColumn string
	BatchSize  int
}

// DefaultParquetConfig returns defaults for Parquet ingestion.
func DefaultParquetConfig() *ParquetConfig {
	return &ParquetConfig{
		BatchSize: 100,
	}
}

// ProcessParquet reads a Parquet file and streams records.
// It uses the processor's Config for field extraction (IDField, ContentField, etc.).
func (p *Processor) ProcessParquet(filePath string, parquetCfg *ParquetConfig) (<-chan *Record, <-chan error) {
	records := make(chan *Record)
	errChan := make(chan error, 1)

	if parquetCfg == nil {
		parquetCfg = DefaultParquetConfig()
	}

	go func() {
		defer close(records)
		defer close(errChan)

		file, err := os.Open(filePath)
		if err != nil {
			errChan <- fmt.Errorf("failed to open parquet file: %w", err)
			return
		}
		defer internal.CheckDefer(file.Close)

		pf, err := parquet.OpenFile(file, 0)
		if err != nil {
			errChan <- fmt.Errorf("failed to parse parquet file: %w", err)
			return
		}

		reader := parquet.NewGenericReader[any](pf)
		defer internal.CheckDefer(reader.Close)

		rows := make([]any, parquetCfg.BatchSize)
		var totalCount int

		for {
			n, err := reader.Read(rows)
			if n == 0 {
				break // EOF
			}
			if err != nil {
				errChan <- fmt.Errorf("error reading parquet rows: %w", err)
				return
			}

			for i := 0; i < n; i++ {
				row, ok := rows[i].(map[string]any)
				if !ok {
					errChan <- fmt.Errorf("row %d is not a map[string]any", totalCount+i)
					return
				}

				// Use processor's extractRecord with column mapping
				record, err := p.extractParquetRecord(row, parquetCfg)
				if err != nil {
					slog.Error("Skipping parquet row", "error", err)
					continue
				}
				if record == nil {
					continue
				}

				records <- record
				totalCount++

				if p.cfg.Limit > 0 && totalCount >= p.cfg.Limit {
					slog.Info("Limit reached", "count", totalCount)
					break
				}
			}

			if p.cfg.Limit > 0 && totalCount >= p.cfg.Limit {
				break
			}
		}

		slog.Info("Parquet processing complete", "total", totalCount)
	}()

	return records, errChan
}

// extractParquetRecord converts a parquet row map into a Record.
// It uses config's ParquetIDColumn/ParquetTextColumn if set, otherwise falls back
// to IDField/ContentField. Metadata extraction follows the processor's config.
func (p *Processor) extractParquetRecord(row map[string]any, parquetCfg *ParquetConfig) (*Record, error) {
	// Determine which column names to use for ID and text
	idColumn := parquetCfg.IDColumn
	textColumn := parquetCfg.TextColumn
	if idColumn == "" {
		idColumn = p.cfg.IDField
	}
	if textColumn == "" {
		textColumn = p.cfg.ContentField
	}

	// Extract ID
	idVal, exists := row[idColumn]
	if !exists {
		return nil, fmt.Errorf("column %s not found", idColumn)
	}
	id := fmt.Sprintf("%v", idVal)

	// Extract text
	textVal, exists := row[textColumn]
	if !exists {
		return nil, fmt.Errorf("column %s not found", textColumn)
	}
	text := fmt.Sprintf("%v", textVal)

	// Extract metadata using the processor's config
	meta := make(map[string]any)
	for k, v := range row {
		if k != idColumn && k != textColumn {
			// Check if we're filtering specific metadata fields
			if p.cfg.AllMetadata {
				meta[k] = v
			} else {
				for _, mf := range p.cfg.MetadataFields {
					if k == mf {
						meta[k] = v
						break
					}
				}
			}
		}
	}

	return &Record{
		ID:       id,
		Content:  text,
		Metadata: meta,
	}, nil
}

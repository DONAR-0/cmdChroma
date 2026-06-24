package ingest

import (
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/DONAR-0/cmdChroma/internal"
)

// ProcessCSV reads a CSV file and streams records through the provided channel.
// The first row is treated as the header (column names). Each subsequent row
// is converted to a Record using the same extractRecord logic as JSONL/Parquet.
func (p *Processor) ProcessCSV(filePath string) (<-chan *Record, <-chan error) {
	records := make(chan *Record)
	errChan := make(chan error, 1)

	go func() {
		defer close(records)
		defer close(errChan)

		f, err := os.Open(filePath)
		if err != nil {
			errChan <- fmt.Errorf("failed to open csv file: %w", err)
			return
		}
		defer internal.CheckDefer(f.Close)

		cr := csv.NewReader(f)

		header, err := cr.Read()
		if err != nil {
			errChan <- fmt.Errorf("failed to read csv header: %w", err)
			return
		}

		lineNo := 1

		var total int

		for {
			row, err := cr.Read()
			if err == io.EOF {
				break
			}

			if err != nil {
				slog.Error("Failed to read csv line", "line", lineNo+1, "error", err)
				lineNo++

				continue
			}

			raw := make(map[string]any, len(header))
			for i, col := range header {
				if i < len(row) {
					raw[col] = row[i]
				} else {
					raw[col] = ""
				}
			}

			recordsList, recErr := p.extractRecord(raw)
			if recErr != nil {
				slog.Error("Skipping csv record", "line", lineNo+1, "error", recErr)
				lineNo++

				continue
			}

			if len(recordsList) == 0 {
				lineNo++
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

			total++
			lineNo++

			if p.cfg.Limit > 0 && total >= p.cfg.Limit {
				slog.Info("Limit reached", "count", total)
				break
			}
		}

		slog.Info("CSV processing complete", "total", total)
	}()

	return records, errChan
}

// CSVRowCount returns the number of data rows (excluding header) in a CSV file.
// Returns 0 if the file cannot be opened or read.
func CSVRowCount(filePath string) int {
	f, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer func() { _ = f.Close() }()

	cr := csv.NewReader(f)

	// Skip header
	if _, err := cr.Read(); err != nil {
		return 0
	}

	var count int

	for {
		_, err := cr.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return 0
		}

		count++
	}

	return count
}

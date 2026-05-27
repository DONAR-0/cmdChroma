package ingest

import (
	"os"
	"testing"
)

func TestProcessParquet(t *testing.T) {
	// Use the small test Parquet file
	filePath := "../../train-00000-of-00001.parquet"

	// Verify file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Skip("Parquet test file not available")
	}

	cfg := DefaultConfig()
	cfg.ContentField = "question"
	processor := NewProcessor(cfg)
	recordsChan, errChan := processor.ProcessParquet(filePath)

	var records []*Record
	for rec := range recordsChan {
		records = append(records, rec)
	}

	if err, ok := <-errChan; ok && err != nil {
		t.Fatalf("ProcessParquet returned error: %v", err)
	}

	// We expect to get some records (at least 1)
	if len(records) == 0 {
		t.Fatal("Expected at least 1 record, got 0")
	}

	// Verify records have required fields
	for i, rec := range records {
		if rec.Content == "" {
			t.Errorf("Record %d has empty Content", i)
		}

		if rec.ID == "" {
			t.Errorf("Record %d has empty ID", i)
		}
	}

	t.Logf("Successfully processed %d records", len(records))
}

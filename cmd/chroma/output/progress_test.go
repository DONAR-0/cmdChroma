package output

import (
	"os"
	"testing"

	"github.com/DONAR-0/cmdChroma/internal/ingest"
)

func TestIngestProgress_Interactive(t *testing.T) {
	cfg := &Config{
		Stdout: os.Stdout,
		NoTTY:  false,
	}
	p := NewIngestProgress(cfg, "test_coll", "data.parquet")
	p.Update(ingest.ProgressInfo{Processed: 50, Total: 100, BatchSize: 10, Batches: 5, Elapsed: 1e9, Done: false})
	p.Update(ingest.ProgressInfo{Processed: 100, Total: 100, BatchSize: 10, Batches: 10, Elapsed: 2e9, Done: true})
}

func TestIngestProgress_InteractiveNoTotal(t *testing.T) {
	cfg := &Config{
		Stdout: os.Stdout,
		NoTTY:  false,
	}
	p := NewIngestProgress(cfg, "test_coll", "data.jsonl")
	p.Update(ingest.ProgressInfo{Processed: 50, Total: 0, Elapsed: 1e9, Done: false})
	p.Update(ingest.ProgressInfo{Processed: 50, Total: 0, Elapsed: 1e9, Done: true})
}

func TestIngestProgress_NonInteractive(t *testing.T) {
	cfg := &Config{
		Stdout: os.Stdout,
		NoTTY:  true,
	}
	p := NewIngestProgress(cfg, "test_coll", "data.parquet")
	p.Update(ingest.ProgressInfo{Processed: 50, Total: 100, Elapsed: 1e9, Done: false})
	p.Update(ingest.ProgressInfo{Processed: 100, Total: 100, Elapsed: 2e9, Done: true})
}

func TestIngestProgress_ZeroTotal(t *testing.T) {
	cfg := &Config{
		Stdout: os.Stdout,
		NoTTY:  false,
	}
	p := NewIngestProgress(cfg, "test_coll", "data.parquet")
	p.Update(ingest.ProgressInfo{Processed: 0, Total: 100, Elapsed: 0, Done: false})
	p.Update(ingest.ProgressInfo{Processed: 100, Total: 100, Elapsed: 1e9, Done: true})
}

package output

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/DONAR-0/cmdChroma/internal/ingest"
)

// IngestProgress renders a live-updating progress display for file ingestion.
// It automatically adapts to interactive (with \r overwrite) and non-interactive
// (periodic new lines) terminal modes.
//
// IngestProgress is NOT safe for concurrent use: Update must be called
// sequentially from a single goroutine.
type IngestProgress struct {
	cfg         *OutputConfig
	collection  string
	filePath    string
	barWidth    int
	lastLine    string
	lastOutput  time.Time
	updateEvery time.Duration
}

// NewIngestProgress creates a new ingestion progress display.
func NewIngestProgress(cfg *OutputConfig, collection, filePath string) *IngestProgress {
	return &IngestProgress{
		cfg:         cfg,
		collection:  collection,
		filePath:    filePath,
		barWidth:    24,
		updateEvery: 80 * time.Millisecond,
	}
}

// Update refreshes the progress display with the latest state.
func (p *IngestProgress) Update(info ingest.ProgressInfo) {
	if info.Done {
		p.renderDone(info)
		return
	}

	if !p.cfg.NoTTY {
		p.renderInteractive(info)
	} else if time.Since(p.lastOutput) >= p.updateEvery {
		p.renderLine(info)
	}
}

func (p *IngestProgress) renderInteractive(info ingest.ProgressInfo) {
	var line string

	elapsed, rate := formatProgress(info)

	if info.Total > 0 {
		pct := float64(info.Processed) / float64(info.Total) * 100

		filled := int(math.Floor(float64(p.barWidth) * pct / 100))
		if filled > p.barWidth {
			filled = p.barWidth
		}

		bar := fmt.Sprintf("[%s%s]",
			strings.Repeat("█", filled),
			strings.Repeat("░", p.barWidth-filled))

		eta := "?"

		if info.Processed > 0 {
			remaining := info.Total - info.Processed

			etaSec := int(float64(remaining) / float64(info.Processed) * float64(elapsed/time.Second))
			if etaSec > 0 {
				eta = fmt.Sprintf("%ds", etaSec)
			}
		}

		batchInfo := ""

		if info.Batches > 0 {
			totalBatches := (info.Total + info.BatchSize - 1) / info.BatchSize
			batchInfo = fmt.Sprintf("  batch %d/%d", info.Batches, totalBatches)
		}

		line = fmt.Sprintf("%s %d/%d (%.1f%%)  %s  %s  ETA %s%s",
			bar, info.Processed, info.Total, pct, rate, elapsed, eta, batchInfo)
	} else {
		line = fmt.Sprintf("⟳ %d docs  •  %s  •  %s",
			info.Processed, rate, elapsed)
	}

	if line != p.lastLine {
		_, _ = fmt.Fprintf(p.cfg.Stdout, "\r%s\033[K", line)
		p.lastLine = line
	}
}

func (p *IngestProgress) renderLine(info ingest.ProgressInfo) {
	var line string

	elapsed, rate := formatProgress(info)

	if info.Total > 0 {
		pct := float64(info.Processed) / float64(info.Total) * 100
		line = fmt.Sprintf("📥  %d/%d docs (%.1f%%)  •  %s  •  %s",
			info.Processed, info.Total, pct, rate, elapsed)
	} else {
		line = fmt.Sprintf("📥  %d docs  •  %s  •  %s",
			info.Processed, rate, elapsed)
	}

	_, _ = fmt.Fprintln(p.cfg.Stdout, line)
	p.lastOutput = time.Now()
}

func (p *IngestProgress) renderDone(info ingest.ProgressInfo) {
	elapsed, rate := formatProgress(info)

	if p.cfg.NoTTY {
		_, _ = fmt.Fprintf(p.cfg.Stdout, "✅ Imported %d docs in %s (%s)\n", info.Processed, elapsed, rate)
	} else {
		_, _ = fmt.Fprintf(p.cfg.Stdout, "\r✅ Imported %d docs in %s (%s)\033[K\n", info.Processed, elapsed, rate)
	}
}

// formatProgress computes elapsed time and throughput rate from ProgressInfo.
func formatProgress(info ingest.ProgressInfo) (time.Duration, string) {
	elapsed := time.Duration(info.Elapsed).Round(time.Second)

	rate := "?"
	if info.Elapsed > 0 {
		rate = fmt.Sprintf("%.0f/s", float64(info.Processed)/(float64(info.Elapsed)/1e9))
	}

	return elapsed, rate
}

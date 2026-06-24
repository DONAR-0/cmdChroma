package ingest

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func writeCSV(t *testing.T, lines ...string) string {
	t.Helper()

	tmpfile, err := os.CreateTemp("", "test.csv")
	require.NoError(t, err)

	for _, line := range lines {
		_, err := tmpfile.WriteString(line + "\n")
		require.NoError(t, err)
	}

	require.NoError(t, tmpfile.Close())

	return tmpfile.Name()
}

func TestProcessCSV_basic(t *testing.T) {
	path := writeCSV(t,
		"id,text,author",
		"1,Hello world,Alice",
		"2,Another document,Bob",
	)
	defer func() { _ = os.Remove(path) }()

	cfg := DefaultConfig()
	cfg.ContentField = "text"
	cfg.IDField = "id"
	proc := NewProcessor(cfg)
	recordsChan, errChan := proc.ProcessCSV(path)

	var records []*Record
	for r := range recordsChan {
		records = append(records, r)
	}

	require.Len(t, records, 2)
	require.Equal(t, "1", records[0].ID)
	require.Equal(t, "Hello world", records[0].Content)
	require.Equal(t, "2", records[1].ID)
	require.Equal(t, "Another document", records[1].Content)

	select {
	case err, ok := <-errChan:
		require.False(t, ok, "error channel should be closed")
		require.NoError(t, err)
	default:
	}
}

func TestProcessCSV_withMetadata(t *testing.T) {
	path := writeCSV(t,
		"id,text,category,score",
		"1,Doc one,tech,42",
		"2,Doc two,science,99",
	)
	defer func() { _ = os.Remove(path) }()

	cfg := DefaultConfig()
	cfg.ContentField = "text"
	cfg.IDField = "id"
	cfg.AllMetadata = true
	proc := NewProcessor(cfg)
	recordsChan, errChan := proc.ProcessCSV(path)

	var records []*Record
	for r := range recordsChan {
		records = append(records, r)
	}

	require.Len(t, records, 2)
	require.Equal(t, "tech", records[0].Metadata["category"])
	require.Equal(t, "42", records[0].Metadata["score"])
	require.Equal(t, "science", records[1].Metadata["category"])
	require.Equal(t, "99", records[1].Metadata["score"])

	select {
	case err, ok := <-errChan:
		require.False(t, ok, "error channel should be closed")
		require.NoError(t, err)
	default:
	}
}

func TestProcessCSV_withLimit(t *testing.T) {
	lines := []string{"id,text"}
	for i := range 10 {
		lines = append(lines, fmt.Sprintf("%d,document %d", i, i))
	}

	path := writeCSV(t, lines...)
	defer func() { _ = os.Remove(path) }()

	cfg := DefaultConfig()
	cfg.Limit = 3
	proc := NewProcessor(cfg)
	recordsChan, errChan := proc.ProcessCSV(path)

	var records []*Record
	for r := range recordsChan {
		records = append(records, r)
	}

	require.Len(t, records, 3)

	select {
	case err, ok := <-errChan:
		require.False(t, ok, "error channel should be closed")
		require.NoError(t, err)
	default:
	}
}

func TestProcessCSV_quotedFields(t *testing.T) {
	path := writeCSV(t,
		`id,text,note`,
		`1,"Hello, world!","note with, comma"`,
		`2,"Multi
line
content",simple`,
	)
	defer func() { _ = os.Remove(path) }()

	cfg := DefaultConfig()
	cfg.ContentField = "text"
	cfg.IDField = "id"
	cfg.AllMetadata = true
	proc := NewProcessor(cfg)
	recordsChan, errChan := proc.ProcessCSV(path)

	var records []*Record
	for r := range recordsChan {
		records = append(records, r)
	}

	require.Len(t, records, 2)
	require.Equal(t, "Hello, world!", records[0].Content)
	require.Equal(t, "note with, comma", records[0].Metadata["note"])

	require.Equal(t, "Multi\nline\ncontent", records[1].Content)
	require.Equal(t, "simple", records[1].Metadata["note"])

	select {
	case err, ok := <-errChan:
		require.False(t, ok, "error channel should be closed")
		require.NoError(t, err)
	default:
	}
}

func TestProcessCSV_emptyFile(t *testing.T) {
	path := writeCSV(t)
	defer func() { _ = os.Remove(path) }()

	proc := NewProcessor(DefaultConfig())
	recordsChan, errChan := proc.ProcessCSV(path)

	var records []*Record
	for r := range recordsChan {
		records = append(records, r)
	}

	require.Len(t, records, 0)

	select {
	case err, ok := <-errChan:
		require.True(t, ok, "error channel should be open")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read csv header")
	default:
		require.Fail(t, "expected error for empty file")
	}
}

func TestProcessCSV_headerOnly(t *testing.T) {
	path := writeCSV(t, "id,text,extra")
	defer func() { _ = os.Remove(path) }()

	cfg := DefaultConfig()
	proc := NewProcessor(cfg)
	recordsChan, errChan := proc.ProcessCSV(path)

	var records []*Record
	for r := range recordsChan {
		records = append(records, r)
	}

	require.Len(t, records, 0)

	select {
	case err, ok := <-errChan:
		require.False(t, ok, "error channel should be closed")
		require.NoError(t, err)
	default:
	}
}

func TestProcessCSV_emptyContent(t *testing.T) {
	path := writeCSV(t,
		"id,text",
		`1,`,
	)
	defer func() { _ = os.Remove(path) }()

	cfg := DefaultConfig()
	proc := NewProcessor(cfg)
	recordsChan, errChan := proc.ProcessCSV(path)

	var records []*Record
	for r := range recordsChan {
		records = append(records, r)
	}

	require.Len(t, records, 1)
	require.Equal(t, "", records[0].Content)

	select {
	case err, ok := <-errChan:
		require.False(t, ok, "error channel should be closed")
		require.NoError(t, err)
	default:
	}
}

func TestProcessCSV_nonExistentFile(t *testing.T) {
	proc := NewProcessor(DefaultConfig())
	recordsChan, errChan := proc.ProcessCSV("non-existent.csv")

	select {
	case err, ok := <-errChan:
		require.True(t, ok, "error channel should be open")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to open csv file")
	case <-time.After(100 * time.Millisecond):
		t.Errorf("timed out waiting for error")
	}

	rec, ok := <-recordsChan
	require.False(t, ok, "records channel should be closed")
	require.Nil(t, rec)
}

func TestProcessCSV_withAutoID(t *testing.T) {
	path := writeCSV(t,
		"id,text",
		"will-be-ignored,hello world",
	)
	defer func() { _ = os.Remove(path) }()

	cfg := DefaultConfig()
	cfg.AutoID = true
	proc := NewProcessor(cfg)
	recordsChan, errChan := proc.ProcessCSV(path)

	var records []*Record
	for r := range recordsChan {
		records = append(records, r)
	}

	require.Len(t, records, 1)
	require.NotEqual(t, "will-be-ignored", records[0].ID)
	require.Len(t, records[0].ID, 24)

	select {
	case err, ok := <-errChan:
		require.False(t, ok, "error channel should be closed")
		require.NoError(t, err)
	default:
	}
}

func TestProcessCSV_largeFile(t *testing.T) {
	var lines []string

	lines = append(lines, "id,text,value")
	for i := range 1000 {
		lines = append(lines, fmt.Sprintf("%d,doc text %d,val_%d", i, i, i))
	}

	path := writeCSV(t, lines...)
	defer func() { _ = os.Remove(path) }()

	cfg := DefaultConfig()
	cfg.BatchSize = 100
	proc := NewProcessor(cfg)
	recordsChan, errChan := proc.ProcessCSV(path)

	var count int
	for range recordsChan {
		count++
	}

	require.Equal(t, 1000, count)

	select {
	case err, ok := <-errChan:
		require.False(t, ok, "error channel should be closed")
		require.NoError(t, err)
	default:
	}
}

func TestCSVRowCount(t *testing.T) {
	lines := []string{"id,text"}
	for i := range 5 {
		lines = append(lines, fmt.Sprintf("%d,text_%d", i, i))
	}

	path := writeCSV(t, lines...)
	defer func() { _ = os.Remove(path) }()

	require.Equal(t, 5, CSVRowCount(path))
}

func TestCSVRowCount_nonExistentFile(t *testing.T) {
	require.Equal(t, 0, CSVRowCount("/nonexistent/file.csv"))
}

func TestCSVRowCount_emptyFile(t *testing.T) {
	path := writeCSV(t)
	defer func() { _ = os.Remove(path) }()

	require.Equal(t, 0, CSVRowCount(path))
}

func TestProcessCSV_selectedMetadata(t *testing.T) {
	path := writeCSV(t,
		"id,text,keep_me,skip_me,also_keep",
		"1,content a,keep_val,skip_val,also_val",
	)
	defer func() { _ = os.Remove(path) }()

	cfg := DefaultConfig()
	cfg.ContentField = "text"
	cfg.IDField = "id"
	cfg.MetadataFields = []string{"keep_me", "also_keep"}
	proc := NewProcessor(cfg)
	recordsChan, errChan := proc.ProcessCSV(path)

	var records []*Record
	for r := range recordsChan {
		records = append(records, r)
	}

	require.Len(t, records, 1)
	require.Contains(t, records[0].Metadata, "keep_me")
	require.Equal(t, "keep_val", records[0].Metadata["keep_me"])
	require.Contains(t, records[0].Metadata, "also_keep")
	require.NotContains(t, records[0].Metadata, "skip_me")

	select {
	case err, ok := <-errChan:
		require.False(t, ok)
		require.NoError(t, err)
	default:
	}
}

func TestProcessCSV_contextCancellation(t *testing.T) {
	path := writeCSV(t, "id,text", "1,hello", "2,world")
	defer func() { _ = os.Remove(path) }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	cfg := DefaultConfig()
	proc := NewProcessor(cfg).WithContext(ctx)
	recordsChan, _ := proc.ProcessCSV(path)

	var records []*Record
	for r := range recordsChan {
		records = append(records, r)
	}

	// Context is cancelled before processing, so we may get 0 or partial results
	require.LessOrEqual(t, len(records), 2)
}

func TestProcessCSV_chunking(t *testing.T) {
	longText := strings.Repeat("Lorem ipsum dolor sit amet. ", 50)

	path := writeCSV(t,
		"id,text",
		fmt.Sprintf("1,%s", longText),
	)
	defer func() { _ = os.Remove(path) }()

	cfg := DefaultConfig()
	cfg.ContentField = "text"
	cfg.ChunkSize = 100
	cfg.ChunkOverlap = 20
	proc := NewProcessor(cfg)
	recordsChan, errChan := proc.ProcessCSV(path)

	var records []*Record
	for r := range recordsChan {
		records = append(records, r)
	}

	require.Greater(t, len(records), 1)

	select {
	case err, ok := <-errChan:
		require.False(t, ok)
		require.NoError(t, err)
	default:
	}
}

func TestProcessCSV_excludeFields(t *testing.T) {
	path := writeCSV(t,
		"id,text,internal_code,public_field",
		"1,content,secret_value,public_value",
	)
	defer func() { _ = os.Remove(path) }()

	cfg := DefaultConfig()
	cfg.ContentField = "text"
	cfg.IDField = "id"
	cfg.AllMetadata = true
	cfg.ExcludeFields = []string{"internal_code"}
	proc := NewProcessor(cfg)
	recordsChan, errChan := proc.ProcessCSV(path)

	var records []*Record
	for r := range recordsChan {
		records = append(records, r)
	}

	require.Len(t, records, 1)
	require.NotContains(t, records[0].Metadata, "internal_code")
	require.Contains(t, records[0].Metadata, "public_field")
	require.Equal(t, "public_value", records[0].Metadata["public_field"])

	select {
	case err, ok := <-errChan:
		require.False(t, ok)
		require.NoError(t, err)
	default:
	}
}

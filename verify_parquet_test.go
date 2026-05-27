package cmdchroma

import (
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/parquet-go/parquet-go"
)

func TestParquetInspection(t *testing.T) {
	filePath := "train-00000-of-00001.parquet"

	t.Logf("Inspecting Parquet file: %s", filePath)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Skipf("Parquet test file not found: %s", filePath)
	}

	// Open file
	f, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	t.Logf("File size: %d bytes (%.2f MB)", stat.Size(), float64(stat.Size())/(1024*1024))

	// Open Parquet file
	pfile, err := parquet.OpenFile(f, stat.Size())
	if err != nil {
		t.Fatalf("Failed to open Parquet file: %v", err)
	}

	schema := pfile.Schema()
	t.Logf("Schema: %s", schema)
	t.Logf("Num Rows: %d", pfile.NumRows())
	t.Logf("Num Row Groups: %d", len(pfile.RowGroups()))

	// Get top-level column names from schema
	fields := schema.Fields()
	t.Logf("Top-level columns (%d):", len(fields))
	for i, field := range fields {
		t.Logf("  [%d] %s (type: %v)", i, field.Name(), field.Type())
	}

	// Show row group details (without assuming colIdx matches field index)
	for rgIdx, rg := range pfile.RowGroups() {
		t.Logf("Row Group %d: %d rows, %d column chunks", rgIdx, rg.NumRows(), len(rg.ColumnChunks()))
	}

	// Read a sample of rows using GenericReader
	if _, err := f.Seek(io.SeekStart, 0); err != nil {
		t.Fatalf("Failed to seek: %v", err)
	}

	reader := parquet.NewGenericReader[any](f)
	rows := make([]any, 3)
	n, err := reader.Read(rows)
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to read sample rows: %v", err)
	}

	t.Logf("Successfully read %d sample rows using GenericReader", n)

	// Print schema of the sample data
	t.Log("Sample row structure (using parquet-go's generic reader):")
	if n > 0 {
		row := rows[0]
		t.Logf("  Row type: %T", row)
		rt := reflect.TypeOf(row)
		if rt.Kind() == reflect.Struct {
			t.Log("  Available fields via reflection:")
			for i := 0; i < rt.NumField(); i++ {
				field := rt.Field(i)
				t.Logf("    %s (%s)", field.Name, field.Type)
			}
		}
	}

	t.Log("\n✅ Parquet file verification passed!")
	t.Log("✅ The parquet-go library can successfully:")
	t.Log("   1. Open and parse the Parquet file structure")
	t.Log("   2. Read row data via GenericReader")
	t.Log("   3. Access column metadata")
	t.Log("\n✅ We are ready to implement ProcessParquet in internal/ingest/manager.go")
}

package ingest

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func BenchmarkStringifyIfComplex(b *testing.B) {
	p := &Processor{}

	b.Run("nil", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			p.stringifyIfComplex(nil)
		}
	})

	b.Run("string", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			p.stringifyIfComplex("hello")
		}
	})

	b.Run("int", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			p.stringifyIfComplex(42)
		}
	})

	b.Run("nested_map", func(b *testing.B) {
		v := map[string]any{"a": map[string]any{"b": map[string]any{"c": "deep"}}}

		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			p.stringifyIfComplex(v)
		}
	})

	b.Run("slice", func(b *testing.B) {
		v := []int{1, 2, 3, 4, 5}

		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			p.stringifyIfComplex(v)
		}
	})

	b.Run("pointer_to_int", func(b *testing.B) {
		v := new(int)
		*v = 99

		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			p.stringifyIfComplex(v)
		}
	})
}

func BenchmarkExtractRecord(b *testing.B) {
	raw := buildBenchRow(20)

	b.Run("no_chunking", func(b *testing.B) {
		p := &Processor{cfg: &Config{
			ContentField: "text",
			IDField:      "id",
		}}

		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = p.extractRecord(raw)
		}
	})

	b.Run("with_metadata", func(b *testing.B) {
		p := &Processor{cfg: &Config{
			ContentField: "text",
			IDField:      "id",
			AllMetadata:  true,
		}}

		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = p.extractRecord(raw)
		}
	})

	b.Run("with_chunking", func(b *testing.B) {
		p := &Processor{cfg: &Config{
			ContentField: "text",
			IDField:      "id",
			ChunkSize:    100,
			ChunkOverlap: 20,
		}}
		bigRaw := buildBenchRow(5)
		bigRaw["text"] = strings.Repeat("Lorem ipsum dolor sit amet consectetur adipiscing elit. ", 500)

		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _ = p.extractRecord(bigRaw)
		}
	})
}

func BenchmarkGetNestedValue(b *testing.B) {
	data := map[string]any{
		"a": map[string]any{
			"b": map[string]any{
				"c": "deep_value",
			},
		},
		"x": 42,
	}

	b.Run("shallow", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			getNestedValue(data, "x")
		}
	})

	b.Run("deep", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			getNestedValue(data, "a.b.c")
		}
	})

	b.Run("missing", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			getNestedValue(data, "a.b.c.d.e.f")
		}
	})
}

func BenchmarkProcessCSV(b *testing.B) {
	for _, size := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("rows_%d", size), func(b *testing.B) {
			path := benchWriteCSV(b, size)
			cfg := DefaultConfig()
			cfg.ContentField = "text"
			cfg.IDField = "id"
			cfg.AllMetadata = true
			proc := NewProcessor(cfg)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				recordsChan, errChan := proc.ProcessCSV(path)
				for range recordsChan {
				}

				if err, ok := <-errChan; ok && err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkCSVRowCount(b *testing.B) {
	path := benchWriteCSV(b, 1000)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		CSVRowCount(path)
	}
}

func benchWriteCSV(b *testing.B, rows int) string {
	b.Helper()

	tmpfile, err := os.CreateTemp("", "bench_*.csv")
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() { _ = os.Remove(tmpfile.Name()) })

	_, err = fmt.Fprintf(tmpfile, "id,text,category,score\n")
	if err != nil {
		b.Fatal(err)
	}

	for i := range rows {
		_, err := fmt.Fprintf(tmpfile, "%d,text_%d,cat_%d,%d\n", i, i, i%10, i)
		if err != nil {
			b.Fatal(err)
		}
	}

	if err := tmpfile.Close(); err != nil {
		b.Fatal(err)
	}

	return tmpfile.Name()
}

func buildBenchRow(nFields int) map[string]any {
	m := make(map[string]any, nFields+2)
	m["id"] = "doc-123"
	m["text"] = "This is the document content for benchmarking purposes."

	m["skip_this"] = "will be excluded"
	for i := range nFields {
		m[fmt.Sprintf("f%d", i)] = fmt.Sprintf("value-%d", i)
	}

	return m
}

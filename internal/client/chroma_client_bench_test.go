package client

import (
	"fmt"
	"testing"
)

func BenchmarkSanitizeMetadataForChroma(b *testing.B) {
	sizes := []struct {
		name  string
		count int
	}{
		{"1_record", 1},
		{"10_records", 10},
		{"100_records", 100},
		{"1000_records", 1000},
	}

	for _, s := range sizes {
		b.Run(s.name, func(b *testing.B) {
			metadatas := makeBenchMetadatas(s.count, 10)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				sanitizeMetadataForChroma(metadatas)
			}
		})
	}
}

func BenchmarkSanitizeMetadataKeys(b *testing.B) {
	keyCounts := []struct {
		name string
		n    int
	}{
		{"5_keys", 5},
		{"20_keys", 20},
		{"50_keys", 50},
	}

	for _, k := range keyCounts {
		b.Run(k.name, func(b *testing.B) {
			metadatas := makeBenchMetadatas(100, k.n)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				sanitizeMetadataForChroma(metadatas)
			}
		})
	}
}

func BenchmarkSanitizeValue(b *testing.B) {
	inputs := []struct {
		name string
		val  any
	}{
		{"nil", nil},
		{"string", "hello"},
		{"int", 42},
		{"slice", []int{1, 2, 3, 4, 5}},
		{"map", map[string]any{"nested": "value", "num": 1.0}},
		{"deep_map", map[string]any{"a": map[string]any{"b": map[string]any{"c": "deep"}}}},
	}

	for _, in := range inputs {
		b.Run(in.name, func(b *testing.B) {
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				sanitizeValue(in.val)
			}
		})
	}
}

func makeBenchMetadatas(count, keysPerRecord int) []map[string]any {
	result := make([]map[string]any, count)
	for i := range count {
		m := make(map[string]any, keysPerRecord)
		for j := range keysPerRecord {
			switch j % 4 {
			case 0:
				m[fmt.Sprintf("str_%d", j)] = fmt.Sprintf("value-%d", i*keysPerRecord+j)
			case 1:
				m[fmt.Sprintf("num_%d", j)] = float64(i + j)
			case 2:
				m[fmt.Sprintf("bool_%d", j)] = (i+j)%2 == 0
			case 3:
				m[fmt.Sprintf("slice_%d", j)] = []int{i, j, i + j}
			}
		}

		result[i] = m
	}

	return result
}

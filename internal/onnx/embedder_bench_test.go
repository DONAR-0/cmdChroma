package onnx

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var benchTexts []string

func init() {
	const corpusSize = 500

	benchTexts = make([]string, corpusSize)
	for i := range corpusSize {
		benchTexts[i] = fmt.Sprintf("This is document number %d with some varied content to ensure unique embeddings %x.", i, rand.Int63())
	}
}

func makeTexts(n int) []string {
	if n > len(benchTexts) {
		n = len(benchTexts)
	}

	return benchTexts[:n]
}

func benchPaths() (modelPath, tokenizersPath, libpath string) {
	_, currentFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(currentFile)
	projectRoot := filepath.Join(dir, "..", "..")
	modelPath = filepath.Join(projectRoot, "models", "all-MiniLM-L6-v2", "model.onnx")
	tokenizersPath = filepath.Join(projectRoot, "models", "all-MiniLM-L6-v2", "tokenizer.json")
	libpath = filepath.Join(projectRoot, "models", "onnx_runtime", "lib", "libonnxruntime.so.1")

	return
}

func newBenchEmbedder(b *testing.B, opts ...EmbedderOption) *Embedder {
	modelPath, tokenizersPath, libpath := benchPaths()
	if _, err := os.Stat(libpath); err != nil {
		b.Skipf("ONNX library not found: %v", err)
	}

	emb, err := NewEmbedder(modelPath, tokenizersPath, libpath, opts...)
	if err != nil {
		b.Fatalf("Failed to create embedder: %v", err)
	}

	return emb
}

func BenchmarkEmbed(b *testing.B) {
	emb := newBenchEmbedder(b)
	defer emb.Close()

	sizes := []struct {
		name string
		text string
	}{
		{"short", "Hello, world!"},
		{"medium", strings.Repeat("The quick brown fox jumps over the lazy dog. ", 20)},
		{"long", strings.Repeat("Go is a statically typed, compiled programming language designed at Google. ", 200)},
	}

	for _, s := range sizes {
		b.Run(s.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := emb.Embed(s.text)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkEmbedDocuments(b *testing.B) {
	emb := newBenchEmbedder(b)
	defer emb.Close()

	ctx := context.Background()

	sizes := []struct {
		name  string
		count int
	}{
		{"sequential_1", 1},
		{"sequential_5", 5},
		{"parallel_50", 50},
		{"parallel_200", 200},
	}

	for _, s := range sizes {
		b.Run(s.name, func(b *testing.B) {
			texts := makeTexts(s.count)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := emb.EmbedDocuments(ctx, texts)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkEmbedDocumentsWorkerCount(b *testing.B) {
	ctx := context.Background()
	texts := makeTexts(100)

	workerCounts := []int{1, 2, 4, 8, runtime.NumCPU()}
	for _, n := range workerCounts {
		b.Run(fmt.Sprintf("workers_%d", n), func(b *testing.B) {
			emb := newBenchEmbedder(b, WithNumWorkers(n))
			defer emb.Close()

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := emb.EmbedDocuments(ctx, texts)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkEmbedConcurrent(b *testing.B) {
	emb := newBenchEmbedder(b)
	defer emb.Close()

	ctx := context.Background()
	texts := makeTexts(20)

	for _, conc := range []int{1, 4, 8} {
		b.Run(fmt.Sprintf("goroutines_%d", conc), func(b *testing.B) {
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, err := emb.EmbedDocuments(ctx, texts)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

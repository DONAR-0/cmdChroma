package onnx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	testEmbedder *Embedder
)

// TestMain initializes the ONNX runtime once for all tests
func TestMain(m *testing.M) {
	// Set up paths
	_, currentFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(currentFile)
	projectRoot := filepath.Join(dir, "..", "..")

	modelPath := filepath.Join(projectRoot, "models", "all-MiniLM-L6-v2", "model.onnx")
	tokenizersPath := filepath.Join(projectRoot, "models", "all-MiniLM-L6-v2", "tokenizer.json")
	libpath := filepath.Join(projectRoot, "models", "onnx_runtime", "lib", "libonnxruntime.so.1")

	// Check if the ONNX library exists
	if _, err := os.Stat(libpath); err != nil {
		panic(fmt.Sprintf("ONNX library not found at %s: %v", libpath, err))
	}

	// Initialize embedder once
	var err error

	testEmbedder, err = NewEmbedder(modelPath, tokenizersPath, libpath)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize embedder: %v", err))
	}

	// Run tests
	code := m.Run()

	// Cleanup
	testEmbedder.Close()

	os.Exit(code)
}

func TestNewEmbedder(t *testing.T) {
	require.NotNil(t, testEmbedder)
	require.NotNil(t, testEmbedder.session)
	require.NotNil(t, testEmbedder.tokenizer)
}

func TestEmbedder_Embed(t *testing.T) {
	text := "Hello, world!"
	embedding, err := testEmbedder.Embed(text)
	require.NoError(t, err)
	require.NotNil(t, embedding)
	require.Equal(t, 384, len(embedding), "expected embedding length 384")
}

func TestEmbedder_EmbedDocuments(t *testing.T) {
	texts := []string{"Hello, world!", "Go is great."}
	embeddings, err := testEmbedder.EmbedDocuments(context.Background(), texts)
	require.NoError(t, err)
	require.NotNil(t, embeddings)
	require.Equal(t, 2, len(embeddings), "expected 2 embeddings")

	for i, emb := range embeddings {
		require.NotNil(t, emb, "embedding %d should not be nil", i)
		require.Equal(t, 384, len(emb), "expected embedding length 384 for text %d", i)
	}
}

func TestEmbedder_Close(_ *testing.T) {
	// Close is tested in TestMain cleanup
	// We can test that calling Close twice doesn't panic
	testEmbedder.Close()
}

func TestWithNumWorkers(t *testing.T) {
	// Create a new embedder with 2 workers
	_, currentFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(currentFile)
	projectRoot := filepath.Join(dir, "..", "..")

	modelPath := filepath.Join(projectRoot, "models", "all-MiniLM-L6-v2", "model.onnx")
	tokenizersPath := filepath.Join(projectRoot, "models", "all-MiniLM-L6-v2", "tokenizer.json")
	libpath := filepath.Join(projectRoot, "models", "onnx_runtime", "lib", "libonnxruntime.so.1")

	emb, err := NewEmbedder(modelPath, tokenizersPath, libpath, WithNumWorkers(2))
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}
	defer emb.Close()

	if emb.numWorkers != 2 {
		t.Errorf("Expected numWorkers to be 2, got %d", emb.numWorkers)
	}
}

func TestEmbedder_EmbedDocuments_Parallel(t *testing.T) {
	// Use a larger batch to trigger parallel path
	// Set numWorkers to 1 and send 5 texts to ensure parallel is used (since len(texts) >= numWorkers)
	_, currentFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(currentFile)
	projectRoot := filepath.Join(dir, "..", "..")

	modelPath := filepath.Join(projectRoot, "models", "all-MiniLM-L6-v2", "model.onnx")
	tokenizersPath := filepath.Join(projectRoot, "models", "all-MiniLM-L6-v2", "tokenizer.json")
	libpath := filepath.Join(projectRoot, "models", "onnx_runtime", "lib", "libonnxruntime.so.1")

	emb, err := NewEmbedder(modelPath, tokenizersPath, libpath, WithNumWorkers(1))
	if err != nil {
		t.Fatalf("Failed to create embedder: %v", err)
	}
	defer emb.Close()

	texts := []string{"Hello", "World", "Go", "is", "great"} // 5 texts, > numWorkers=1

	embeddings, err := emb.EmbedDocuments(context.Background(), texts)
	if err != nil {
		t.Fatalf("EmbedDocuments failed: %v", err)
	}

	if len(embeddings) != len(texts) {
		t.Errorf("Expected %d embeddings, got %d", len(texts), len(embeddings))
	}

	for i, embd := range embeddings {
		if embd == nil {
			t.Errorf("Embedding %d is nil", i)
		}

		if len(embd) != 384 {
			t.Errorf("Embedding %d has wrong dimension: %d", i, len(embd))
		}
	}
}

// ExampleNewEmbedder demonstrates creating an embedder and generating a simple embedding.
func ExampleNewEmbedder() {
	// This example assumes model files are in the standard location.
	// In practice, use paths from your environment or configuration.
	_, currentFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(currentFile)
	projectRoot := filepath.Join(dir, "..", "..")

	modelPath := filepath.Join(projectRoot, "models", "all-MiniLM-L6-v2", "model.onnx")
	tokenizersPath := filepath.Join(projectRoot, "models", "all-MiniLM-L6-v2", "tokenizer.json")
	libpath := filepath.Join(projectRoot, "models", "onnx_runtime", "lib", "libonnxruntime.so.1")

	emb, err := NewEmbedder(modelPath, tokenizersPath, libpath)
	if err != nil {
		// Handle error (in production code)
		panic(err)
	}
	defer emb.Close()

	// Generate an embedding for a single text
	vec, err := emb.Embed("Hello, world!")
	if err != nil {
		panic(err)
	}

	_ = vec
	// fmt.Printf("Embedding length: %d\n", len(vec))
}

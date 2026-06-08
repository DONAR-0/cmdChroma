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

func TestEmbedder_Close(t *testing.T) {
	// Close is tested in TestMain cleanup
	// We can test that calling Close twice doesn't panic
	testEmbedder.Close()
}

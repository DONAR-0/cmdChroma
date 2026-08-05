// Package embedding provides a shallow, test-friendly abstraction over the ONNX embedder.
package embedding

import (
	"fmt"

	"github.com/DONAR-0/cmdChroma/internal/onnx"
)

// EmbeddingEngine is the public contract for embedding look-ups.
// It hides the underlying ONNX implementation behind a single method.
type EmbeddingEngine interface {
	// Encode returns a vector for the supplied text.
	Encode(text string) ([]float32, error)
}

// onnxEngine bridges the ONNX embedder to the EmbeddingEngine interface.
type onnxEngine struct {
	embedder *onnx.Embedder
}

// NewEmbeddingEngine builds an EmbeddingEngine backed by ONNX.
func NewEmbeddingEngine(
	modelPath, tokenizerPath, onnxLibPath string,
) (EmbeddingEngine, error) {
	embedder, err := onnx.NewEmbedder(modelPath, tokenizerPath, onnxLibPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create ONNX embedder: %w", err)
	}

	return &onnxEngine{embedder: embedder}, nil
}

// Encode implements EmbeddingEngine by delegating to the underlying ONNX embedder.
func (e *onnxEngine) Encode(text string) ([]float32, error) {
	return e.embedder.Embed(text)
}

// Compile-time check that onnxEngine satisfies EmbeddingEngine.
var _ EmbeddingEngine = (*onnxEngine)(nil)

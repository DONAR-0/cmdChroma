package llm

import (
	"context"
	"io"
)

// Package llm provides abstractions for Large Language Model providers.
// It supports pluggable backends (Ollama, NVIDIA NIM) for RAG-style generation.

// ProviderInterface defines the contract for LLM providers.
// Implementations generate answers given a prompt and optionally return
// the generated text synchronously.
type ProviderInterface interface {
	// Generate streams the response to the provided writer.
	// If writer is nil, output is sent to os.Stdout (backward compatible).
	// Returns error if generation fails or context is cancelled.
	Generate(ctx context.Context, prompt, model string, writer io.Writer) error

	// GenerateSync returns the full generated text as a string.
	// This is a convenience wrapper around streaming.
	GenerateSync(ctx context.Context, prompt, model string) (string, error)
}

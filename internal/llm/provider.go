package llm

// Package llm provides abstractions for Large Language Model providers.
// It supports pluggable backends (Ollama, NVIDIA NIM) for RAG-style generation.
import "context"

// ProviderInterface defines the contract for LLM providers.
// Implementations generate answers given a prompt and optionally return
// the generated text synchronously.
type ProviderInterface interface {
	// Generate streams the response to stdout. It returns when generation
	// completes or the context is cancelled.
	Generate(ctx context.Context, prompt, model string) error

	// GenerateSync returns the full generated text as a string.
	// This is a convenience wrapper around streaming.
	GenerateSync(ctx context.Context, prompt, model string) (string, error)
}

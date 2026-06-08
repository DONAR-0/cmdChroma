package llm

import "context"

// ProviderInterface defines the contract for LLM providers.
type ProviderInterface interface {
	Generate(ctx context.Context, prompt, model string) error
	GenerateSync(ctx context.Context, prompt, model string) (string, error)
}

package llm

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/tmc/langchaingo/llms/openai"
)

// NIMProvider handles LLM interactions with NVIDIA NIM API using LangChainGo.
type NIMProvider struct {
	adapter *LangChainAdapter
}

// NewNIMProvider creates a new NVIDIA NIM provider using the LangChainGo adapter.
// The baseURL should be the API endpoint (e.g., "https://api.nvidia.com/v1").
// The apiKey is the NVIDIA API key. If empty, it reads from NVIDIA_API_KEY env var.
func NewNIMProvider(baseURL, apiKey string) (*NIMProvider, error) {
	if apiKey == "" {
		apiKey = os.Getenv("NVIDIA_API_KEY")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("NVIDIA API key is required: set NVIDIA_API_KEY environment variable")
	}

	if baseURL == "" {
		baseURL = "https://api.nvidia.com/v1"
	}

	// NVIDIA NIM is OpenAI-compatible, so we use the OpenAI provider with a custom BaseURL.
	llm, err := openai.New(openai.WithBaseURL(baseURL), openai.WithToken(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize NIM llm via langchaingo: %w", err)
	}

	return &NIMProvider{
		adapter: NewLangChainAdapter(llm),
	}, nil
}

// Generate streams response from the NIM API.
func (p *NIMProvider) Generate(ctx context.Context, prompt, model string, w io.Writer) error {
	return p.adapter.Generate(ctx, prompt, model, w)
}

// GenerateSync returns the full response as a string.
func (p *NIMProvider) GenerateSync(ctx context.Context, prompt, model string) (string, error) {
	return p.adapter.GenerateSync(ctx, prompt, model)
}

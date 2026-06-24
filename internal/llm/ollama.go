package llm

import (
	"context"
	"fmt"
	"io"

	"github.com/tmc/langchaingo/llms/ollama"
)

// Provider handles LLM interactions using LangChainGo.
type Provider struct {
	adapter *LangChainAdapter
}

// NewProvider creates a new Ollama LLM provider using the LangChainGo adapter.
func NewProvider(baseURL string) *Provider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	// Using the updated package path from latest langchaingo
	llm, err := ollama.New(ollama.WithServerURL(baseURL))
	if err != nil {
		fmt.Printf("warning: failed to initialize ollama llm: %v\n", err)
	}

	return &Provider{
		adapter: NewLangChainAdapter(llm),
	}
}

// Generate streams response from the LLM.
func (p *Provider) Generate(ctx context.Context, prompt, model string, w io.Writer) error {
	return p.adapter.Generate(ctx, prompt, model, w)
}

// GenerateSync returns the full response as a string.
func (p *Provider) GenerateSync(ctx context.Context, prompt, model string) (string, error) {
	return p.adapter.GenerateSync(ctx, prompt, model)
}

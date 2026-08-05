package llm

import (
	"fmt"
	"os"

	"github.com/tmc/langchaingo/llms/openai"
)

func NewNIMProvider(baseURL, apiKey string) (*LangChainAdapter, error) {
	if apiKey == "" {
		apiKey = os.Getenv("NVIDIA_API_KEY")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("NVIDIA API key is required: set NVIDIA_API_KEY environment variable")
	}

	if baseURL == "" {
		baseURL = "https://api.nvidia.com/v1"
	}

	llm, err := openai.New(openai.WithBaseURL(baseURL), openai.WithToken(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize NIM llm via langchaingo: %w", err)
	}

	return NewLangChainAdapter(llm), nil
}

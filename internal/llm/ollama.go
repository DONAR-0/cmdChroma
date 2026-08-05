package llm

import (
	"fmt"

	"github.com/tmc/langchaingo/llms/ollama"
)

func NewProvider(baseURL string) *LangChainAdapter {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	llm, err := ollama.New(ollama.WithServerURL(baseURL))
	if err != nil {
		fmt.Printf("warning: failed to initialize ollama llm: %v\n", err)
	}

	return NewLangChainAdapter(llm)
}

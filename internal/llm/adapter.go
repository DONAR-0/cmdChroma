package llm

import (
	"context"
	"fmt"
	"io"

	"github.com/tmc/langchaingo/llms"
)

// LangChainAdapter wraps a LangChainGo LLM to satisfy the ProviderInterface.
type LangChainAdapter struct {
	llm llms.Model
}

// NewLangChainAdapter creates a new adapter for a given LangChainGo model.
func NewLangChainAdapter(model llms.Model) *LangChainAdapter {
	return &LangChainAdapter{llm: model}
}

func (a *LangChainAdapter) Generate(ctx context.Context, prompt, model string, writer io.Writer) error {
	msg := llms.TextParts(llms.ChatMessageTypeHuman, prompt)

	opts := []llms.CallOption{
		llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			_, err := writer.Write(chunk)
			return err
		}),
	}
	if model != "" {
		opts = append(opts, llms.WithModel(model))
	}

	_, err := a.llm.GenerateContent(ctx, []llms.MessageContent{msg}, opts...)
	if err != nil {
		return fmt.Errorf("langchaingo generation failed: %w", err)
	}

	return nil
}

// GenerateSync returns the full generated text as a string.
func (a *LangChainAdapter) GenerateSync(ctx context.Context, prompt, model string) (string, error) {
	msg := llms.TextParts(llms.ChatMessageTypeHuman, prompt)

	opts := []llms.CallOption{}
	if model != "" {
		opts = append(opts, llms.WithModel(model))
	}

	resp, err := a.llm.GenerateContent(ctx, []llms.MessageContent{msg}, opts...)
	if err != nil {
		return "", fmt.Errorf("langchaingo sync generation failed: %w", err)
	}

	var result string
	for _, choice := range resp.Choices {
		result += choice.Content
	}

	return result, nil
}

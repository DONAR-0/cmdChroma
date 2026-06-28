package llm

import (
	"context"
	"testing"

	"github.com/tmc/langchaingo/llms"
)

func TestProvider_DelegatesToAdapter(t *testing.T) {
	p := &Provider{adapter: NewLangChainAdapter(&mockModel{
		generateContentFunc: func(_ context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
			return &llms.ContentResponse{
				Choices: []*llms.ContentChoice{
					{Content: "delegated"},
				},
			}, nil
		},
	})}

	ctx := context.Background()

	out, err := p.GenerateSync(ctx, "test", "model")
	if err != nil {
		t.Fatalf("GenerateSync failed: %v", err)
	}

	if out != "delegated" {
		t.Errorf("expected 'delegated', got %q", out)
	}
}

func TestNIMProvider_DelegatesToAdapter(t *testing.T) {
	p := &NIMProvider{adapter: NewLangChainAdapter(&mockModel{
		generateContentFunc: func(_ context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
			return &llms.ContentResponse{
				Choices: []*llms.ContentChoice{
					{Content: "nim delegated"},
				},
			}, nil
		},
	})}

	ctx := context.Background()

	out, err := p.GenerateSync(ctx, "test", "nim://my-model")
	if err != nil {
		t.Fatalf("GenerateSync failed: %v", err)
	}

	if out != "nim delegated" {
		t.Errorf("expected 'nim delegated', got %q", out)
	}
}

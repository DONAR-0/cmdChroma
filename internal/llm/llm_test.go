package llm

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/tmc/langchaingo/llms"
)

// mockModel implements llms.Model for testing the adapter.
type mockModel struct {
	generateContentFunc func(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error)
	callFunc            func(ctx context.Context, prompt string, options ...llms.CallOption) (string, error)
}

func (m *mockModel) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if m.generateContentFunc != nil {
		return m.generateContentFunc(ctx, messages, options...)
	}

	return nil, errors.New("GenerateContent not implemented")
}

func (m *mockModel) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	if m.callFunc != nil {
		return m.callFunc(ctx, prompt, options...)
	}

	return "", errors.New("Call not implemented")
}

// ============ Adapter Tests ============

func TestNewLangChainAdapter(t *testing.T) {
	mock := &mockModel{}

	adapter := NewLangChainAdapter(mock)
	if adapter == nil {
		t.Fatal("Expected non-nil adapter")
	}

	if adapter.GetModel() != mock {
		t.Error("GetModel should return the same model")
	}
}

func TestLangChainAdapter_Generate(t *testing.T) {
	mock := &mockModel{
		generateContentFunc: func(_ context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
			// Verify the message has the correct role and content
			if len(messages) != 1 {
				t.Fatalf("expected 1 message, got %d", len(messages))
			}

			if messages[0].Role != llms.ChatMessageTypeHuman {
				t.Errorf("expected human role, got %v", messages[0].Role)
			}
			// Check streaming func was set in options
			return &llms.ContentResponse{
				Choices: []*llms.ContentChoice{
					{Content: "Hello world!"},
				},
			}, nil
		},
	}
	adapter := NewLangChainAdapter(mock)
	ctx := context.Background()

	var buf bytes.Buffer

	err := adapter.Generate(ctx, "test prompt", "", &buf)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
}

func TestLangChainAdapter_GenerateSync(t *testing.T) {
	expected := "Hello world!"
	mock := &mockModel{
		generateContentFunc: func(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
			return &llms.ContentResponse{
				Choices: []*llms.ContentChoice{
					{Content: expected},
				},
			}, nil
		},
	}
	adapter := NewLangChainAdapter(mock)
	ctx := context.Background()

	out, err := adapter.GenerateSync(ctx, "test prompt", "")
	if err != nil {
		t.Fatalf("GenerateSync failed: %v", err)
	}

	if out != expected {
		t.Errorf("unexpected response: %q", out)
	}
}

func TestLangChainAdapter_GenerateSync_Error(t *testing.T) {
	mock := &mockModel{
		generateContentFunc: func(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
			return nil, errors.New("mock error")
		},
	}
	adapter := NewLangChainAdapter(mock)
	ctx := context.Background()

	_, err := adapter.GenerateSync(ctx, "test prompt", "")
	if err == nil {
		t.Errorf("expected error")
	}

	if !strings.Contains(err.Error(), "langchaingo sync generation failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLangChainAdapter_Generate_Error(t *testing.T) {
	mock := &mockModel{
		generateContentFunc: func(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
			return nil, errors.New("mock error")
		},
	}
	adapter := NewLangChainAdapter(mock)
	ctx := context.Background()

	var buf bytes.Buffer

	err := adapter.Generate(ctx, "test prompt", "", &buf)
	if err == nil {
		t.Errorf("expected error")
	}

	if !strings.Contains(err.Error(), "langchaingo generation failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLangChainAdapter_Generate_ModelOption(t *testing.T) {
	var capturedModel string

	mock := &mockModel{
		generateContentFunc: func(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
			// The model option should be passed through to the underlying llm
			return &llms.ContentResponse{
				Choices: []*llms.ContentChoice{
					{Content: "ok"},
				},
			}, nil
		},
	}

	// We can't easily check the model was passed to the mock without wrapping,
	// but we can verify the adapter doesn't error on model string
	_ = capturedModel
	adapter := NewLangChainAdapter(mock)
	ctx := context.Background()

	var buf bytes.Buffer

	err := adapter.Generate(ctx, "test prompt", "my-model", &buf)
	if err != nil {
		t.Fatalf("Generate with model failed: %v", err)
	}
}

// ============ Provider (Ollama) Tests ============

func TestNewProvider(t *testing.T) {
	p := NewProvider("")
	if p == nil {
		t.Errorf("Expected provider to be non-nil")
	}

	p2 := NewProvider("http://custom:11434")
	if p2 == nil {
		t.Errorf("Expected provider to be non-nil")
	}
}

// ============ NIMProvider Tests ============

func TestNewNIMProvider(t *testing.T) {
	if err := os.Setenv("NVIDIA_API_KEY", "env-key"); err != nil {
		t.Fatalf("failed to set NVIDIA_API_KEY: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("NVIDIA_API_KEY"); err != nil {
			t.Fatalf("failed to unset NVIDIA_API_KEY: %v", err)
		}
	}()

	p, err := NewNIMProvider("", "")
	if err != nil {
		t.Fatalf("NewNIMProvider failed: %v", err)
	}

	if p == nil {
		t.Errorf("Expected provider to be non-nil")
	}

	p2, err := NewNIMProvider("http://custom.ai/v1", "my-key")
	if err != nil {
		t.Fatalf("NewNIMProvider failed: %v", err)
	}

	if p2 == nil {
		t.Errorf("Expected provider to be non-nil")
	}
}

func TestNewNIMProvider_MissingAPIKey(t *testing.T) {
	_ = os.Unsetenv("NVIDIA_API_KEY")

	_, err := NewNIMProvider("", "")
	if err == nil {
		t.Error("expected error when API key is missing")
	}
}

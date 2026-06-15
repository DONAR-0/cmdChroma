package factory

import (
	"os"
	"testing"
)

func TestLLMProviderFactory_CreateProvider_Ollama(t *testing.T) {
	f := NewLLMProviderFactory()

	prov, err := f.CreateProvider("llama2", "")
	if err != nil {
		t.Fatalf("CreateProvider failed: %v", err)
	}

	if prov == nil {
		t.Errorf("expected non-nil provider")
	}
}

func TestLLMProviderFactory_CreateProvider_NIM(t *testing.T) {
	if os.Getenv("NVIDIA_API_KEY") == "" {
		t.Skip("NVIDIA_API_KEY not set — skipping NIM integration test")
	}

	f := NewLLMProviderFactory()

	prov, err := f.CreateProvider("nim://mistralai/mistral-7b", "http://localhost:8080")
	if err != nil {
		t.Fatalf("CreateProvider failed: %v", err)
	}

	if prov == nil {
		t.Errorf("expected non-nil provider")
	}
}

func TestLLMProviderFactory_CreateProvider_EmptyModel(t *testing.T) {
	f := NewLLMProviderFactory()

	prov, err := f.CreateProvider("", "")
	if err != nil {
		t.Fatalf("CreateProvider failed: %v", err)
	}

	if prov == nil {
		t.Errorf("expected non-nil provider for empty model (defaults to Ollama)")
	}
}

func TestLLMProviderFactory_ValidateModel_NIM_EmptyID(t *testing.T) {
	f := NewLLMProviderFactory()

	err := f.ValidateModel("nim://")
	if err == nil {
		t.Errorf("expected error for empty NIM model ID")
	}

	if err != nil && !contains(err.Error(), "invalid NIM model format") {
		t.Errorf("expected 'invalid NIM model format' error, got: %v", err)
	}
}

func TestLLMProviderFactory_ValidateModel_Valid(t *testing.T) {
	f := NewLLMProviderFactory()
	if err := f.ValidateModel("llama2"); err != nil {
		t.Errorf("unexpected error for valid model: %v", err)
	}

	if err := f.ValidateModel("nim://mistralai/mistral-7b"); err != nil {
		t.Errorf("unexpected error for valid NIM model: %v", err)
	}
}

func TestLLMProviderFactory_ValidateModel_ValidOllama(t *testing.T) {
	f := NewLLMProviderFactory()

	err := f.ValidateModel("qwen:0.5b")
	if err != nil {
		t.Errorf("unexpected error for valid Ollama model: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}

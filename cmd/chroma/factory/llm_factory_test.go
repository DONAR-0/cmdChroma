package factory

import (
	"os"
	"strings"
	"testing"
)

func TestCreateProvider_Ollama(t *testing.T) {
	prov, err := CreateProvider("llama2", "")
	if err != nil {
		t.Fatalf("CreateProvider failed: %v", err)
	}

	if prov == nil {
		t.Errorf("expected non-nil provider")
	}
}

func TestCreateProvider_NIM(t *testing.T) {
	if os.Getenv("NVIDIA_API_KEY") == "" {
		t.Skip("NVIDIA_API_KEY not set — skipping NIM integration test")
	}

	prov, err := CreateProvider("nim://mistralai/mistral-7b", "http://localhost:8080")
	if err != nil {
		t.Fatalf("CreateProvider failed: %v", err)
	}

	if prov == nil {
		t.Errorf("expected non-nil provider")
	}
}

func TestCreateProvider_EmptyModel(t *testing.T) {
	prov, err := CreateProvider("", "")
	if err != nil {
		t.Fatalf("CreateProvider failed: %v", err)
	}

	if prov == nil {
		t.Errorf("expected non-nil provider for empty model (defaults to Ollama)")
	}
}

func TestValidateModel_NIM_EmptyID(t *testing.T) {
	err := ValidateModel("nim://")
	if err == nil {
		t.Errorf("expected error for empty NIM model ID")
	}

	if err != nil && !strings.Contains(err.Error(), "invalid NIM model format") {
		t.Errorf("expected 'invalid NIM model format' error, got: %v", err)
	}
}

func TestValidateModel_Valid(t *testing.T) {
	if err := ValidateModel("llama2"); err != nil {
		t.Errorf("unexpected error for valid model: %v", err)
	}

	if err := ValidateModel("nim://mistralai/mistral-7b"); err != nil {
		t.Errorf("unexpected error for valid NIM model: %v", err)
	}
}

func TestValidateModel_ValidOllama(t *testing.T) {
	err := ValidateModel("qwen:0.5b")
	if err != nil {
		t.Errorf("unexpected error for valid Ollama model: %v", err)
	}
}

func TestValidateModel_NIM_Valid(t *testing.T) {
	if err := ValidateModel("nim://mistralai/mistral-7b"); err != nil {
		t.Errorf("unexpected error for valid NIM model: %v", err)
	}
}

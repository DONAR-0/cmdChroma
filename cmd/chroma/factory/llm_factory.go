package factory

import (
	"errors"
	"strings"

	"github.com/DONAR-0/cmdChroma/internal/llm"
)

type LLMProviderFactory struct{}

func NewLLMProviderFactory() *LLMProviderFactory {
	return &LLMProviderFactory{}
}

func (f *LLMProviderFactory) CreateProvider(model, nimURL string) (llm.ProviderInterface, error) {
	if model == "" {
		model = "qwen:0.5b"
	}

	if strings.HasPrefix(model, "nim://") {
		if nimURL == "" {
			nimURL = "https://integrate.api.nvidia.com/v1"
		}

		return llm.NewNIMProvider(nimURL, "")
	}

	return llm.NewProvider(""), nil
}

func (f *LLMProviderFactory) ValidateModel(model string) error {
	if strings.HasPrefix(model, "nim://") {
		modelID := strings.TrimPrefix(model, "nim://")
		if modelID == "" {
			return errors.New("invalid NIM model format: use nim://<model-id>")
		}
	}

	return nil
}

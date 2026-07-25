package llm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
)

// LocalONNXProvider implements ProviderInterface using native CGO bindings
// to the onnxruntime-genai library.
type LocalONNXProvider struct {
	modelsDir string
	cache     map[string]*ONNXGenAI
	mu        sync.RWMutex
}

// NewLocalONNXProvider creates a new provider that manages native ONNX LLMs.
func NewLocalONNXProvider(modelsDir string) *LocalONNXProvider {
	return &LocalONNXProvider{
		modelsDir: modelsDir,
		cache:     make(map[string]*ONNXGenAI),
	}
}

func (p *LocalONNXProvider) getModel(modelID string) (*ONNXGenAI, error) {
	p.mu.RLock()

	if m, ok := p.cache[modelID]; ok {
		p.mu.RUnlock()
		return m, nil
	}

	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double check
	if m, ok := p.cache[modelID]; ok {
		return m, nil
	}

	// Resolve path (assuming modelID is a directory name in modelsDir)
	// In a real system, we'd use the ModelManager to verify installation.
	modelPath := fmt.Sprintf("%s/%s", p.modelsDir, modelID)

	genAI, err := NewONNXGenAI(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load native ONNX model %s: %w", modelID, err)
	}

	p.cache[modelID] = genAI

	return genAI, nil
}

// Generate handles text generation. Note: Native bindings currently
// implement Sync generation; streaming is simulated by writing the full result.
func (p *LocalONNXProvider) Generate(ctx context.Context, prompt, model string, writer io.Writer) error {
	// Strip local:// prefix if present
	modelID := model
	if len(model) > 8 && model[:8] == "local://" {
		modelID = model[8:]
	}

	genAI, err := p.getModel(modelID)
	if err != nil {
		return err
	}

	text, err := genAI.Generate(prompt)
	if err != nil {
		return err
	}

	_, err = writer.Write([]byte(text))

	return err
}

// GenerateSync returns the full generated text as a string.
func (p *LocalONNXProvider) GenerateSync(ctx context.Context, prompt, model string) (string, error) {
	var buf bytes.Buffer // Need to import bytes

	err := p.Generate(ctx, prompt, model, &buf)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (p *LocalONNXProvider) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, m := range p.cache {
		m.Close()
	}

	p.cache = make(map[string]*ONNXGenAI)
}

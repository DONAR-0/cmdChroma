package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/DONAR-0/cmdChroma/internal"
)

// ChatRequest is the payload sent to Ollama.
type ChatRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// ChatResponse represents one chunk of streamed output.
type ChatResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// Provider handles LLM interactions.
type Provider struct {
	baseURL string
	client  *http.Client
}

// NewProvider creates a new LLM provider.
func NewProvider(baseURL string) *Provider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &Provider{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// Generate streams response from the LLM.
func (p *Provider) Generate(ctx context.Context, prompt, model string) error {
	if model == "" {
		model = "qwen:0.5b"
	}

	reqBody := ChatRequest{
		Model:  model,
		Prompt: prompt,
		Stream: true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/generate", p.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama connection failed: %w", err)
	}
	defer internal.CheckDefer(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	// Stream output
	scanner := bufio.NewScanner(resp.Body)

	fmt.Println("\n🤖 AI Response:")
	fmt.Println(strings.Repeat("-", 20))

	for scanner.Scan() {
		var r ChatResponse
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			continue
		}

		fmt.Print(r.Response)

		if r.Done {
			break
		}
	}

	fmt.Println("\n" + strings.Repeat("-", 20))

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream reading error: %w", err)
	}

	return nil
}

// GenerateSync returns the full response as a string (non-streaming).
func (p *Provider) GenerateSync(ctx context.Context, prompt, model string) (string, error) {
	// For now we could implement non-streaming by collecting streaming chunks
	// Or Ollama also has /api/generate with stream=false
	if model == "" {
		model = "qwen:0.5b"
	}

	reqBody := ChatRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/generate", p.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama connection failed: %w", err)
	}
	defer internal.CheckDefer(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var fullResp struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&fullResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return fullResp.Response, nil
}

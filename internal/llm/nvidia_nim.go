package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/DONAR-0/cmdChroma/internal"
)

// NVIDIA NIM Chat Request/Response structures (OpenAI-compatible)

type NIMChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type NIMChatRequest struct {
	Model       string           `json:"model"`
	Messages    []NIMChatMessage `json:"messages"`
	Stream      bool             `json:"stream"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
}

type NIMChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int            `json:"index"`
		Message      NIMChatMessage `json:"message"`
		FinishReason string         `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param,omitempty"`
		Code    string `json:"code,omitempty"`
	} `json:"error,omitempty"`
}

type NIMStreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int            `json:"index"`
		Delta        NIMChatMessage `json:"delta"`
		FinishReason string         `json:"finish_reason"`
	} `json:"choices"`
}

// NIMProvider handles LLM interactions with NVIDIA NIM API.
type NIMProvider struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewNIMProvider creates a new NVIDIA NIM provider.
// The baseURL should be the API endpoint (e.g., "https://api.nvidia.com/v1")
// The apiKey is the NVIDIA API key. If empty, it reads from NVIDIA_API_KEY env var.
// Returns an error if the API key is missing.
func NewNIMProvider(baseURL, apiKey string) (*NIMProvider, error) {
	if apiKey == "" {
		apiKey = os.Getenv("NVIDIA_API_KEY")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("NVIDIA API key is required: set NVIDIA_API_KEY environment variable")
	}

	if baseURL == "" {
		baseURL = "https://api.nvidia.com/v1"
	}

	return &NIMProvider{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 120 * time.Second}, // Default timeout for LLM calls
	}, nil
}

// Generate streams response from the NIM API.
func (p *NIMProvider) Generate(ctx context.Context, prompt, model string) error {
	start := time.Now()

	modelID, err := p.validateModel(model)
	if err != nil {
		return err
	}

	slog.Info("nim_generate_start", "model", modelID, "prompt_len", len(prompt))

	reqBody := NIMChatRequest{
		Model: modelID,
		Messages: []NIMChatMessage{
			{Role: "user", Content: prompt},
		},
		Stream:      true,
		MaxTokens:   1024,
		Temperature: 0.7,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", p.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("NIM API request failed: %w", err)
	}
	defer internal.CheckDefer(resp.Body.Close)

	// Handle specific HTTP status codes
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("NIM authentication failed: invalid or expired API key")
	case http.StatusTooManyRequests:
		return fmt.Errorf("NIM rate limit exceeded: try again later")
	case http.StatusBadGateway, http.StatusServiceUnavailable:
		return fmt.Errorf("NIM service unavailable (status %d): try again later", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("NIM API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Stream output
	scanner := bufio.NewScanner(resp.Body)

	fmt.Println("\n🤖 AI Response (NVIDIA NIM):")
	fmt.Println(strings.Repeat("-", 20))

	for scanner.Scan() {
		// Check context cancellation during streaming
		select {
		case <-ctx.Done():
			return fmt.Errorf("generation cancelled: %w", ctx.Err())
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk NIMStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 {
			fmt.Print(chunk.Choices[0].Delta.Content)
		}
	}

	fmt.Println("\n" + strings.Repeat("-", 20))

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream reading error: %w", err)
	}

	slog.Info("nim_generate_complete", "model", modelID, "duration_ms", time.Since(start).Milliseconds())

	return nil
}

// GenerateSync returns the full response as a string (non-streaming).
func (p *NIMProvider) GenerateSync(ctx context.Context, prompt, model string) (string, error) {
	start := time.Now()

	modelID, err := p.validateModel(model)
	if err != nil {
		return "", err
	}

	slog.Info("nim_generate_sync_start", "model", modelID)

	reqBody := NIMChatRequest{
		Model: modelID,
		Messages: []NIMChatMessage{
			{Role: "user", Content: prompt},
		},
		Stream:      false,
		MaxTokens:   1024,
		Temperature: 0.7,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", p.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("NIM API request failed: %w", err)
	}
	defer internal.CheckDefer(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("NIM API returned status %d: %s", resp.StatusCode, string(body))
	}

	var fullResp NIMChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&fullResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(fullResp.Choices) > 0 {
		slog.Info("nim_generate_sync_complete", "model", modelID, "duration_ms", time.Since(start).Milliseconds())
		return fullResp.Choices[0].Message.Content, nil
	}

	if fullResp.Error != nil {
		return "", fmt.Errorf("NIM API error: %s", fullResp.Error.Message)
	}

	return "", fmt.Errorf("no response from NIM API")
}

// validateModel validates the model string and extracts the model ID.
func (p *NIMProvider) validateModel(model string) (string, error) {
	if model == "" {
		return "", fmt.Errorf("NIM model must be specified")
	}

	// Strip nim:// prefix if present
	model = strings.TrimPrefix(model, "nim://")
	if model == "" {
		return "", fmt.Errorf("NIM model ID is empty after stripping prefix")
	}

	return model, nil
}

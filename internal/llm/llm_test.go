package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Helper for Ollama
func makeOllamaStreamResponse(response string, done bool) string {
	return fmt.Sprintf(`{"response": %q, "done": %t}`, response, done)
}

// Local helper for NIM streaming (test only)
type nimStreamChunk struct {
	ID      string            `json:"id"`
	Object  string            `json:"object"`
	Created int64             `json:"created"`
	Model   string            `json:"model"`
	Choices []nimStreamChoice `json:"choices"`
}
type nimStreamChoice struct {
	Index        int            `json:"index"`
	Delta        NIMChatMessage `json:"delta"`
	FinishReason string         `json:"finish_reason"`
}

func makeNIMStreamChunkLine(content string, done bool) string {
	chunk := nimStreamChunk{
		ID:      "chatcmpl-123",
		Object:  "chat.completion.chunk",
		Created: 1699000000,
		Model:   "meta/llama2-70b-chat",
		Choices: []nimStreamChoice{
			{
				Index: 0,
				Delta: NIMChatMessage{
					Role:    "assistant",
					Content: content,
				},
			},
		},
	}
	if done {
		chunk.Choices[0].FinishReason = "stop"
	}

	data, _ := json.Marshal(chunk)

	return "data: " + string(data) + "\n"
}

// ============ Provider (Ollama) Tests ============

func TestNewProvider(t *testing.T) {
	p := NewProvider("")
	if p.baseURL != "http://localhost:11434" {
		t.Errorf("Expected default baseURL, got %s", p.baseURL)
	}

	p2 := NewProvider("http://custom:11434")
	if p2.baseURL != "http://custom:11434" {
		t.Errorf("Expected custom baseURL, got %s", p2.baseURL)
	}
}

func TestProvider_GenerateSync_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var reqBody map[string]any
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode error: %v", err)
		}

		if reqBody["stream"] != false {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if _, err := fmt.Fprint(w, `{"response":"Hello world!"}`); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	prov := NewProvider(server.URL)
	ctx := context.Background()

	out, err := prov.GenerateSync(ctx, "prompt", "llama2")
	if err != nil {
		t.Fatalf("GenerateSync failed: %v", err)
	}

	if out != "Hello world!" {
		t.Errorf("unexpected response: %q", out)
	}
}

func TestProvider_GenerateSync_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	prov := NewProvider(server.URL)
	ctx := context.Background()

	_, err := prov.GenerateSync(ctx, "test", "llama2")
	if err == nil {
		t.Errorf("expected error for GenerateSync")
	}
}

func TestProvider_Generate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if _, err := fmt.Fprint(w, makeOllamaStreamResponse(" part1", false)); err != nil {
			t.Fatalf("failed to write streaming response part1: %v", err)
		}

		if _, err := fmt.Fprint(w, makeOllamaStreamResponse(" part2", true)); err != nil {
			t.Fatalf("failed to write streaming response part2: %v", err)
		}
	}))
	defer server.Close()

	prov := NewProvider(server.URL)
	ctx := context.Background()

	err := prov.Generate(ctx, "prompt", "llama2")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
}

func TestProvider_Generate_HttpError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	prov := NewProvider(server.URL)
	ctx := context.Background()

	err := prov.Generate(ctx, "test", "llama2")
	if err == nil {
		t.Errorf("expected error for Generate with BadRequest")
	}
}

func TestProvider_Generate_DefaultModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody struct {
			Model  string `json:"model"`
			Prompt string `json:"prompt"`
			Stream bool   `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("decode error: %v", err)
		}

		if reqBody.Model != "qwen:0.5b" {
			t.Errorf("expected default model qwen:0.5b, got %s", reqBody.Model)
		}

		if _, err := fmt.Fprint(w, `{"response":"ok","done":true}`); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	prov := NewProvider(server.URL)
	ctx := context.Background()

	err := prov.Generate(ctx, "test", "")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
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

	if p.baseURL != "https://api.nvidia.com/v1" || p.apiKey != "env-key" {
		t.Errorf("unexpected defaults: %s %s", p.baseURL, p.apiKey)
	}

	p2, err := NewNIMProvider("http://custom.ai/v1", "my-key")
	if err != nil {
		t.Fatalf("NewNIMProvider failed: %v", err)
	}

	if p2.baseURL != "http://custom.ai/v1" || p2.apiKey != "my-key" {
		t.Errorf("unexpected values: %s %s", p2.baseURL, p2.apiKey)
	}
}

func TestNewNIMProvider_MissingAPIKey(t *testing.T) {
	// Ensure API key env var is not set
	_ = os.Unsetenv("NVIDIA_API_KEY")

	_, err := NewNIMProvider("", "")
	if err == nil {
		t.Error("expected error when API key is missing")
	}
}

func TestNIMProvider_GenerateSync_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var req NIMChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode error: %v", err)
		}

		if req.Stream != false {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if auth := r.Header.Get("Authorization"); auth != "Bearer dummy-key" {
			t.Errorf("unexpected auth header: %s", auth)
		}
		// Construct a minimal valid JSON response
		resp := `{"id":"chatcmpl-123","object":"chat.completion","created":1699000000,"model":"meta/llama2-70b-chat","choices":[{"index":0,"message":{"role":"assistant","content":"NIM says hi"}}]}`
		if _, err := fmt.Fprint(w, resp); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	prov, err := NewNIMProvider(server.URL, "dummy-key")
	if err != nil {
		t.Fatalf("NewNIMProvider failed: %v", err)
	}

	ctx := context.Background()

	out, err := prov.GenerateSync(ctx, "hello", "gpt-3.5-turbo")
	if err != nil {
		t.Fatalf("GenerateSync failed: %v", err)
	}

	if out != "NIM says hi" {
		t.Errorf("unexpected response: %q", out)
	}
}

func TestNIMProvider_GenerateSync_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)

		if _, err := fmt.Fprint(w, `{"error":{"message":"invalid api key"}}`); err != nil {
			t.Fatalf("failed to write error response: %v", err)
		}
	}))
	defer server.Close()

	prov, err := NewNIMProvider(server.URL, "bad-key")
	if err != nil {
		t.Fatalf("NewNIMProvider failed: %v", err)
	}

	ctx := context.Background()

	_, err = prov.GenerateSync(ctx, "hello", "gpt-3.5-turbo")
	if err == nil {
		t.Errorf("expected error for GenerateSync")
	}
}

func TestNIMProvider_Generate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("flusher not supported")
		}

		w.Header().Set("Content-Type", "text/event-stream")

		if _, err := fmt.Fprint(w, makeNIMStreamChunkLine("Hello", false)); err != nil {
			t.Fatalf("failed to write stream chunk: %v", err)
		}

		flusher.Flush()

		if _, err := fmt.Fprint(w, makeNIMStreamChunkLine(" World", true)); err != nil {
			t.Fatalf("failed to write stream chunk: %v", err)
		}
	}))
	defer server.Close()

	prov, err := NewNIMProvider(server.URL, "dummy-key")
	if err != nil {
		t.Fatalf("NewNIMProvider failed: %v", err)
	}

	ctx := context.Background()

	err = prov.Generate(ctx, "prompt", "gpt-3.5-turbo")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
}

func TestNIMProvider_Generate_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	prov, err := NewNIMProvider(server.URL, "dummy")
	if err != nil {
		t.Fatalf("NewNIMProvider failed: %v", err)
	}

	ctx := context.Background()

	err = prov.Generate(ctx, "prompt", "gpt-3.5-turbo")
	if err == nil {
		t.Errorf("expected error for Generate")
	}
}

func TestNIMProvider_GenerateSync_RequiresModel(t *testing.T) {
	// Set up a mock server that accepts the request (for auth validation)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"choices":[{"message":{"content":"ok"}}]}`)
	}))
	defer server.Close()

	prov, err := NewNIMProvider(server.URL, "dummy-key-for-test")
	if err != nil {
		t.Fatalf("NewNIMProvider failed: %v", err)
	}

	ctx := context.Background()

	_, err = prov.GenerateSync(ctx, "prompt", "")
	if err == nil {
		t.Errorf("expected error when model empty")
	}

	if !strings.Contains(err.Error(), "NIM model must be specified") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNIMProvider_GenerateSync_TrimPrefix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req NIMChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode error: %v", err)
		}

		if req.Model != "my-model" {
			t.Errorf("expected model name to have prefix stripped, got %s", req.Model)
		}

		resp := `{"choices":[{"index":0,"message":{"role":"assistant","content":"ok"}}]}`
		if _, err := fmt.Fprint(w, resp); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	prov, err := NewNIMProvider(server.URL, "key")
	if err != nil {
		t.Fatalf("NewNIMProvider failed: %v", err)
	}

	ctx := context.Background()

	out, err := prov.GenerateSync(ctx, "test", "nim://my-model")
	if err != nil {
		t.Fatalf("GenerateSync failed: %v", err)
	}

	if out != "ok" {
		t.Errorf("unexpected response: %s", out)
	}
}

func TestNIMProvider_Generate_AuthorizationHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer my-secret-key" {
			t.Errorf("unexpected Authorization header: %s", auth)
		}

		w.WriteHeader(http.StatusOK)

		resp := `{"choices":[{"index":0,"message":{"role":"assistant","content":"ok"}}]}`
		if _, err := fmt.Fprint(w, resp); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	prov, err := NewNIMProvider(server.URL, "my-secret-key")
	if err != nil {
		t.Fatalf("NewNIMProvider failed: %v", err)
	}

	ctx := context.Background()

	_, err = prov.GenerateSync(ctx, "test", "any-model")
	if err != nil {
		t.Fatalf("GenerateSync failed: %v", err)
	}
}

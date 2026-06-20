package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/DONAR-0/cmdChroma/cmd/chat-server/service"
	"github.com/DONAR-0/cmdChroma/cmd/chat-server/storage"
	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	svc          *service.ChatService
	sessions     *storage.SessionStore
	defaultModel string
	nimURL       string
}

func NewChatHandler(svc *service.ChatService, sessions *storage.SessionStore, defaultModel string) *ChatHandler {
	return &ChatHandler{
		svc:          svc,
		sessions:     sessions,
		defaultModel: defaultModel,
		nimURL:       "https://integrate.api.nvidia.com/v1",
	}
}

// ChatRequest is the POST /api/chat body.
type ChatRequest struct {
	Collection        string  `json:"collection" binding:"required"`
	Message           string  `json:"message" binding:"required"`
	Model             string  `json:"model"`
	SessionID         string  `json:"session_id"`
	NResults          int     `json:"n_results"`
	DistanceThreshold float64 `json:"distance_threshold"`
}

// Chat handles streaming RAG chat via SSE.
func (h *ChatHandler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.NResults == 0 {
		req.NResults = 3
	}

	apiKey := c.GetHeader("X-API-Key")

	model := req.Model
	if model == "" {
		model = h.defaultModel
	}

	// Get or create session — persists conversation history across turns
	session := h.sessions.GetOrCreate(apiKey, req.SessionID, req.Collection)

	// Switching collection mid-session clears history
	if session.Collection != req.Collection {
		_ = h.sessions.Clear(apiKey, session.ID)
		session.Collection = req.Collection
	}

	ctx := c.Request.Context()

	// 1: Semantic search in ChromaDB
	results, err := h.svc.Query(ctx, req.Collection, req.Message, req.NResults, req.DistanceThreshold)
	if err != nil {
		h.writeErrorSSE(c, "collection or query error: "+err.Error())
		return
	}

	// 2: Build context string and RAG prompt
	contextStr, hasRelevant := buildContext(results)

	var prompt string
	if len(session.Messages) > 0 {
		prompt = h.svc.BuildPromptWithHistory(session.Messages, req.Message, contextStr, hasRelevant)
	} else {
		prompt = h.svc.BuildRAGPrompt(req.Message, contextStr, hasRelevant)
	}

	// 3: Create the right LLM provider (strips nim:// internally)
	strippedModel := strings.TrimPrefix(model, "nim://")

	provider, err := h.svc.CreateProvider(strippedModel, apiKey)
	if err != nil {
		h.writeErrorSSE(c, err.Error())
		return
	}

	// 4: Begin SSE stream
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)

	if f, ok := c.Writer.(http.Flusher); ok {
		f.Flush()
	}

	// 5: Stream tokens to client
	sw := &sseWriter{w: c.Writer}
	err = provider.Generate(ctx, prompt, strippedModel, sw)

	if err != nil && ctx.Err() == nil {
		h.writeErrorSSE(c, "generation error: "+err.Error())
		return
	}

	// 6: Mark SSE stream done
	if _, err := fmt.Fprintf(sw, "event: done\\ndata: {}\n\n"); err != nil {
		h.writeErrorSSE(c, err.Error())
	}

	c.Writer.Flush()

	// 7: Append to session history
	if err := h.sessions.AppendMessage(session.ID, apiKey, "user", req.Message, 0); err != nil {
		h.writeErrorSSE(c, err.Error())
	}
}

// buildContext formats retrieved documents into a string for RAG prompt injection.
func buildContext(results []service.QueryResult) (string, bool) {
	if len(results) == 0 {
		return "", false
	}

	var sb strings.Builder
	for i, r := range results {
		fmt.Fprintf(&sb, "[Context %d]: %s\n", i+1, r.Content)
	}

	return sb.String(), true
}

// sseWriter wraps http.ResponseWriter to produce SSE token events with immediate flush.
type sseWriter struct{ w io.Writer }

func (sw *sseWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	event := fmt.Sprintf("event: token\ndata: %s\n\n", string(p))
	if _, err := sw.w.Write([]byte(event)); err != nil {
		return 0, err
	}

	if f, ok := sw.w.(http.Flusher); ok {
		f.Flush()
	}

	return len(p), nil
}

func (h *ChatHandler) writeErrorSSE(c *gin.Context, msg string) {
	if _, err := fmt.Fprintf(c.Writer, "event: error\\ndata: %s\n\n", msg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write error SSE: %v\n", err)
	}

	c.Writer.Flush()
}

// compile-time interface check
var _ io.Writer = (*sseWriter)(nil)

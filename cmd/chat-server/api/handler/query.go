package handler

import (
	"net/http"

	"github.com/DONAR-0/cmdChroma/cmd/chat-server/config"
	"github.com/DONAR-0/cmdChroma/cmd/chat-server/service"
	"github.com/DONAR-0/cmdChroma/cmd/chat-server/storage"
	"github.com/gin-gonic/gin"
)

// QueryHandler handles non-streaming semantic query requests.
type QueryHandler struct {
	service *service.ChatService
}

func NewQueryHandler(svc *service.ChatService) *QueryHandler {
	return &QueryHandler{service: svc}
}

// QueryRequest is the POST /api/query body.
type QueryRequest struct {
	Collection        string  `json:"collection" binding:"required"`
	Query             string  `json:"query" binding:"required"`
	NResults          int     `json:"n_results"`
	DistanceThreshold float64 `json:"distance_threshold"`
}

// Query runs a raw semantic search and returns documents without LLM involvement.
func (h *QueryHandler) Query(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.NResults == 0 {
		req.NResults = 5
	}

	results, err := h.service.Query(c.Request.Context(), req.Collection, req.Query, req.NResults, req.DistanceThreshold)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"documents": results,
		"count":     len(results),
	})
}

// SessionHandler handles session list/clear operations.
type SessionHandler struct {
	store *storage.SessionStore
}

func NewSessionHandler(ss *storage.SessionStore) *SessionHandler {
	return &SessionHandler{store: ss}
}

// ListCollections serves the server-configured collection list.
func (h *SessionHandler) ListCollections(collections []config.CollectionEntry) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing API key"})
			return
		}

		out := make([]gin.H, len(collections))
		for i, col := range collections {
			out[i] = gin.H{"name": col.Name, "description": col.Description}
		}

		c.JSON(http.StatusOK, gin.H{"collections": out})
	}
}

func (h *SessionHandler) ListSessions(c *gin.Context) {
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing API key"})
		return
	}

	sessions := h.store.List(apiKey)

	out := make([]gin.H, len(sessions))
	for i, sess := range sessions {
		out[i] = gin.H{
			"id":            sess.ID,
			"collection":    sess.Collection,
			"message_count": len(sess.Messages),
			"created_at":    sess.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{"sessions": out})
}

func (h *SessionHandler) ClearSession(c *gin.Context) {
	apiKey := c.GetHeader("X-API-Key")

	sessionID := c.Param("id")
	if apiKey == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing API key"})
		return
	}

	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session_id required"})
		return
	}

	cleared := h.store.Clear(apiKey, sessionID)
	if !cleared {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "cleared", "session_id": sessionID})
}

// HealthHandler returns server health status.
type HealthHandler struct {
	chroma   any
	embedder any
}

func NewHealthHandler(chroma, embedder any) *HealthHandler {
	return &HealthHandler{chroma: chroma, embedder: embedder}
}

func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"chroma":   "connected",
		"embedder": "ready",
	})
}

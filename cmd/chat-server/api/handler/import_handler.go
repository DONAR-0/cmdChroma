package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/DONAR-0/cmdChroma/cmd/chat-server/service"
	"github.com/gin-gonic/gin"
)

type ImportHandler struct {
	svc *service.ChatService
}

func NewImportHandler(svc *service.ChatService) *ImportHandler {
	return &ImportHandler{svc: svc}
}

func (h *ImportHandler) importSSE(c *gin.Context, collectionName, filePath, contentField, idField string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)

	if f, ok := c.Writer.(http.Flusher); ok {
		f.Flush()
	}

	sendEvent := func(event string, data string) {
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data)
		c.Writer.Flush()
	}

	sendEvent("start", `{"status":"importing"}`)

	progressFn := func(processed int) {
		sendEvent("progress", fmt.Sprintf(`{"processed":%d}`, processed))
	}

	if contentField == "" {
		contentField = "text"
	}

	if idField == "" {
		idField = "id"
	}

	err := h.svc.ImportFile(c.Request.Context(), collectionName, filePath, contentField, idField, progressFn)
	if err != nil {
		sendEvent("error", fmt.Sprintf(`{"error":"%s"}`, err.Error()))
		return
	}

	sendEvent("done", `{"status":"completed"}`)

	os.Remove(filePath)
}

func (h *ImportHandler) ImportJSONL(c *gin.Context) {
	collectionName := c.Param("name")
	if collectionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "collection name required"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required: " + err.Error()})
		return
	}
	defer file.Close()

	ext := filepath.Ext(header.Filename)
	if ext != ".jsonl" && ext != ".parquet" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported format, use .jsonl or .parquet"})
		return
	}

	tmpFile, err := os.CreateTemp("", "import-*"+ext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp file"})
		return
	}
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, file); err != nil {
		os.Remove(tmpFile.Name())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})

		return
	}

	tmpFile.Close()

	h.importSSE(c, collectionName, tmpFile.Name(), "text", "id")
}

type ImportURLRequest struct {
	URL     string `json:"url" binding:"required"`
	Content string `json:"content_field"`
	ID      string `json:"id_field"`
}

func (h *ImportHandler) ImportFromURL(c *gin.Context) {
	collectionName := c.Param("name")
	if collectionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "collection name required"})
		return
	}

	var req ImportURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Download file
	resp, err := http.Get(req.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to download: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("download failed: HTTP %d", resp.StatusCode)})
		return
	}

	// Detect format from Content-Type header, fallback to URL extension
	contentType := resp.Header.Get("Content-Type")

	ext := filepath.Ext(req.URL)
	if strings.Contains(contentType, "json") || ext == ".jsonl" {
		ext = ".jsonl"
	} else if strings.Contains(contentType, "parquet") || ext == ".parquet" {
		ext = ".parquet"
	} else {
		ext = ".jsonl"
	}

	tmpFile, err := os.CreateTemp("", "import-*"+ext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp file"})
		return
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		os.Remove(tmpFile.Name())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save downloaded file"})

		return
	}

	tmpFile.Close()

	h.importSSE(c, collectionName, tmpFile.Name(), req.Content, req.ID)
}

//go:build onnxgenai

package handler

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/DONAR-0/cmdChroma/cmd/server/service"
	"github.com/DONAR-0/cmdChroma/internal/ingest"
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
		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data); err != nil {
			return
		}

		c.Writer.Flush()
	}

	h.importSSEWithEvents(c, collectionName, filePath, contentField, idField, sendEvent)
}

func (h *ImportHandler) importSSEWithEvents(c *gin.Context, collectionName, filePath, contentField, idField string, sendEvent func(string, string)) {
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

	// Use a detached context so the import continues even if the HTTP request
	// context is cancelled by WriteTimeout. The SSE stream stays alive via
	// sendEvent writes to c.Writer which are independent of the context.
	// No timeout - the import runs until complete or cancelled by the user.
	importCtx := context.Background()

	slog.Info("Starting import", "collection", collectionName, "file", filePath)

	err := h.svc.ImportFile(importCtx, collectionName, filePath, contentField, idField, progressFn)
	if err != nil {
		slog.Error("Import failed", "error", err)
		sendEvent("error", fmt.Sprintf(`{"error":"%s"}`, err.Error()))

		return
	}

	slog.Info("Import completed successfully, sending done event")
	sendEvent("done", `{"status":"completed"}`)

	if err := os.Remove(filePath); err != nil {
		fmt.Fprintf(os.Stderr, "error removing temp file %s: %v\n", filePath, err)
	}
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
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing uploaded file: %v\n", err)
		}
	}()

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
	defer func() {
		if err := tmpFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "error closing temp file: %v\n", err)
		}
	}()

	if _, err := io.Copy(tmpFile, file); err != nil {
		if err := os.Remove(tmpFile.Name()); err != nil {
			fmt.Fprintf(os.Stderr, "error removing temp file: %v\n", err)
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})

		return
	}

	if err := tmpFile.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "error closing temp file: %v\n", err)
	}

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

	// Start SSE connection
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)

	if f, ok := c.Writer.(http.Flusher); ok {
		f.Flush()
	}

	sendEvent := func(event string, data string) {
		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data); err != nil {
			return
		}

		c.Writer.Flush()
	}

	// Send start event with downloading status
	sendEvent("start", `{"status":"downloading"}`)

	// Use a detached context for download so it's not affected by WriteTimeout
	downloadCtx := context.Background()

	// Download file with progress tracking
	tmpFilePath, err := ingest.DownloadFile(downloadCtx, req.URL, func(downloaded, total int64) {
		sendEvent("download-progress", fmt.Sprintf(`{"downloaded":%d,"total":%d}`, downloaded, total))
	})
	if err != nil {
		sendEvent("error", fmt.Sprintf(`{"error":"%s"}`, err.Error()))
		return
	}

	// Send download complete event
	sendEvent("download-complete", `{"status":"downloaded"}`)

	// Detect format from URL extension
	ext := filepath.Ext(req.URL)
	if ext != ".jsonl" && ext != ".parquet" {
		ext = ".jsonl"
	}

	// Rename temp file to have correct extension if needed
	filePath := tmpFilePath
	if !strings.HasSuffix(filePath, ext) {
		newPath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ext
		if err := os.Rename(filePath, newPath); err != nil {
			fmt.Fprintf(os.Stderr, "error renaming temp file: %v\n", err)
		} else {
			filePath = newPath
		}
	}

	// Start import process
	h.importSSEWithEvents(c, collectionName, filePath, req.Content, req.ID, sendEvent)
}

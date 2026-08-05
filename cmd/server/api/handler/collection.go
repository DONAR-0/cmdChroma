//go:build onnxgenai

package handler

import (
	"net/http"

	"github.com/DONAR-0/cmdChroma/cmd/server/service"
	"github.com/gin-gonic/gin"
)

type CollectionHandler struct {
	svc *service.ChatService
}

func NewCollectionHandler(svc *service.ChatService) *CollectionHandler {
	return &CollectionHandler{svc: svc}
}

func (h *CollectionHandler) ListCollections(c *gin.Context) {
	collections, err := h.svc.ListCollectionsWithCount(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"collections": collections})
}

type CreateCollectionRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *CollectionHandler) CreateCollection(c *gin.Context) {
	var req CreateCollectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	id, err := h.svc.CreateCollection(c.Request.Context(), req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name})
}

func (h *CollectionHandler) DeleteCollection(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "collection name required"})
		return
	}

	if err := h.svc.DeleteCollection(c.Request.Context(), name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted", "name": name})
}

type ListDocumentsRequest struct {
	Limit  int `json:"limit,omitempty" form:"limit"`
	Offset int `json:"offset,omitempty" form:"offset"`
}

func (h *CollectionHandler) ListDocuments(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "collection name required"})
		return
	}

	records, err := h.svc.ListDocuments(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	count := len(records.IDs)

	items := make([]gin.H, count)
	for i := range count {
		items[i] = gin.H{
			"id":       records.IDs[i],
			"content":  records.Documents[i],
			"metadata": records.Metadatas[i],
		}
	}

	c.JSON(http.StatusOK, gin.H{"documents": items, "count": count})
}

type AddDocumentsRequest struct {
	Documents []string         `json:"documents" binding:"required"`
	IDs       []string         `json:"ids"`
	Metadatas []map[string]any `json:"metadatas,omitempty"`
}

func (h *CollectionHandler) AddDocuments(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "collection name required"})
		return
	}

	var req AddDocumentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if err := h.svc.AddDocuments(c.Request.Context(), name, req.Documents, req.IDs, req.Metadatas); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "added", "count": len(req.Documents)})
}

type DeleteDocumentsRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

func (h *CollectionHandler) DeleteDocuments(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "collection name required"})
		return
	}

	var req DeleteDocumentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if err := h.svc.DeleteDocuments(c.Request.Context(), name, req.IDs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted", "count": len(req.IDs)})
}

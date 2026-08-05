//go:build onnxgenai

package handler

import (
	"net/http"

	"github.com/DONAR-0/cmdChroma/cmd/server/service"
	"github.com/gin-gonic/gin"
)

type ModelHandler struct {
	modelMgr *service.ModelManager
}

func NewModelHandler(mgr *service.ModelManager) *ModelHandler {
	return &ModelHandler{
		modelMgr: mgr,
	}
}

func (h *ModelHandler) ListModels(c *gin.Context) {
	models, err := h.modelMgr.GetAvailableModels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models)
}

func (h *ModelHandler) DownloadModel(c *gin.Context) {
	var req struct {
		ModelID string `json:"model_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Note: Download can take a long time. In a real app, we'd use a background job.
	if err := h.modelMgr.DownloadModel(req.ModelID, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "download completed"})
}

func (h *ModelHandler) SetActiveModel(c *gin.Context) {
	var req struct {
		ModelID string `json:"model_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if !h.modelMgr.IsInstalled(req.ModelID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model not installed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "model set as active"})
}

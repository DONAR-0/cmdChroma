//go:build onnxgenai

package api

import (
	"github.com/DONAR-0/cmdChroma/cmd/chat-server/api/handler"
	"github.com/DONAR-0/cmdChroma/cmd/chat-server/api/middleware"
	"github.com/DONAR-0/cmdChroma/cmd/chat-server/config"
	"github.com/gin-gonic/gin"
)

type RouterDeps struct {
	APIKey            string
	ChatHandler       *handler.ChatHandler
	QueryHandler      *handler.QueryHandler
	SessionHandler    *handler.SessionHandler
	HealthHandler     *handler.HealthHandler
	CollectionHandler *handler.CollectionHandler
	ImportHandler     *handler.ImportHandler
	ModelHandler      *handler.ModelHandler
	Collections       []config.CollectionEntry
}

func NewRouter(deps RouterDeps) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.CORS())

	// Health check — no auth required
	r.GET("/health", deps.HealthHandler.Health)

	// All /api/* routes require API key
	api := r.Group("/api", middleware.APIKeyAuth(deps.APIKey))
	{
		api.POST("/chat", deps.ChatHandler.Chat)
		api.POST("/query", deps.QueryHandler.Query)
		api.GET("/sessions", deps.SessionHandler.ListSessions)
		api.DELETE("/sessions/:id", deps.SessionHandler.ClearSession)
		api.GET("/collections", deps.CollectionHandler.ListCollections)
		api.POST("/collections", deps.CollectionHandler.CreateCollection)
		api.DELETE("/collections/:name", deps.CollectionHandler.DeleteCollection)
		api.GET("/collections/:name/documents", deps.CollectionHandler.ListDocuments)
		api.POST("/collections/:name/documents", deps.CollectionHandler.AddDocuments)
		api.DELETE("/collections/:name/documents", deps.CollectionHandler.DeleteDocuments)
		api.POST("/collections/:name/import", deps.ImportHandler.ImportJSONL)
		api.POST("/collections/:name/import/url", deps.ImportHandler.ImportFromURL)
		api.GET("/models", deps.ModelHandler.ListModels)
		api.POST("/models/download", deps.ModelHandler.DownloadModel)
		api.POST("/models/active", deps.ModelHandler.SetActiveModel)
	}

	return r
}

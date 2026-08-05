package middleware

import (
	"crypto/subtle"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth validates the X-API-Key header using constant-time comparison.
func APIKeyAuth(validKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		provided := c.GetHeader("X-API-Key")
		if provided == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing X-API-Key header",
			})

			return
		}

		if subtle.ConstantTimeCompare([]byte(provided), []byte(validKey)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid API key",
			})

			return
		}

		c.Next()
	}
}

// RequestLogger logs method, path, status, and latency for every request.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start).Milliseconds()
		status := c.Writer.Status()

		level := slog.Info
		if status >= 400 {
			level = slog.Warn
		}

		level("http request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"query", c.Request.URL.RawQuery,
			"status", status,
			"latency_ms", latency,
			"client_ip", c.ClientIP(),
		)
	}
}

// CORS adds standard CORS headers for browser clients.
// For team-internal use; expand to validate origin against an allowlist in production.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

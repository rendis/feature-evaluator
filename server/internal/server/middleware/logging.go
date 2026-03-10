package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logging logs each request with method, path, status, and duration.
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		//nolint:gosec // Request logging uses structured fields for observability and does not execute user input.
		slog.Info("request",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"duration", time.Since(start).String(),
			"requestId", GetRequestID(c),
			"clientIP", c.ClientIP(),
		)
	}
}

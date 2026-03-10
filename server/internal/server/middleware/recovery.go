package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// Recovery recovers from panics and returns a structured error response.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				//nolint:gosec // Panic recovery logs request metadata for debugging only.
				slog.Error("panic recovered",
					"error", r,
					"requestId", GetRequestID(c),
					"path", c.Request.URL.Path,
				)

				apiErr := apierror.NewInternal("internal server error", "error.internal")
				apiErr.RequestID = GetRequestID(c)
				c.AbortWithStatusJSON(http.StatusInternalServerError, apierror.ErrorResponse{Error: apiErr})
			}
		}()
		c.Next()
	}
}

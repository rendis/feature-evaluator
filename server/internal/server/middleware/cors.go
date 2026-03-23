package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/securitypolicy"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// CORS returns a middleware that handles Cross-Origin Resource Sharing.
func CORS(policyReader securitypolicy.Reader) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		c.Header("Vary", "Origin")
		if origin == "" {
			c.Next()
			return
		}

		originSet := buildOriginSet(policyReader)
		if _, allowed := originSet[origin]; allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, X-Workspace, X-Tenant-ID, X-Campus-ID, X-Program-ID, X-Api-Key")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Max-Age", "86400")
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(http.StatusNoContent)
				return
			}
			c.Next()
			return
		}

		apiErr := apierror.NewForbidden("origin is not allowed", "error.originNotAllowed")
		apiErr.RequestID = GetRequestID(c)
		c.AbortWithStatusJSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
	}
}

func buildOriginSet(policyReader securitypolicy.Reader) map[string]struct{} {
	if policyReader == nil {
		return nil
	}

	origins := policyReader.Snapshot().CORSOrigins.Effective
	originSet := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		originSet[origin] = struct{}{}
	}

	return originSet
}

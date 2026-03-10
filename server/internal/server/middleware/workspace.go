package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/workspace"
)

// CtxWorkspaceKey is the gin context key for workspace.
const CtxWorkspaceKey = "workspaceKey"

// WorkspaceResolver reads the X-Workspace header and injects the workspace key
// into both the gin context and the request's context.Context.
func WorkspaceResolver() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-Workspace")
		if key == "" {
			key = workspace.DefaultKey
		}
		c.Set(CtxWorkspaceKey, key)
		c.Request = c.Request.WithContext(workspace.WithKey(c.Request.Context(), key))
		c.Next()
	}
}

// GetWorkspaceKey returns the workspace key from gin context.
func GetWorkspaceKey(c *gin.Context) string {
	if v, ok := c.Get(CtxWorkspaceKey); ok {
		return v.(string)
	}
	return ""
}

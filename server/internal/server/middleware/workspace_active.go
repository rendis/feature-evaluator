package middleware

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/workspace"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// RequireActiveWorkspace ensures the current request targets an existing active workspace.
func RequireActiveWorkspace(workspaceSvc *workspace.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := GetWorkspaceKey(c)
		if key == "" {
			apiErr := apierror.NewBadRequest("workspace is required", "error.workspaceRequired")
			apiErr.RequestID = GetRequestID(c)
			c.AbortWithStatusJSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
			return
		}

		ws, err := workspaceSvc.GetByKey(c.Request.Context(), key)
		if err != nil {
			var apiErr *apierror.APIError
			if errors.As(err, &apiErr) {
				apiErr.RequestID = GetRequestID(c)
				c.AbortWithStatusJSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
				return
			}
			respErr := apierror.NewInternal("workspace lookup failed", "error.internal")
			respErr.RequestID = GetRequestID(c)
			c.AbortWithStatusJSON(respErr.HTTPStatus, apierror.ErrorResponse{Error: respErr})
			return
		}

		if ws.IsArchived() {
			apiErr := apierror.NewForbidden("workspace is archived", "error.workspaceArchived")
			apiErr.RequestID = GetRequestID(c)
			c.AbortWithStatusJSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
			return
		}

		c.Next()
	}
}

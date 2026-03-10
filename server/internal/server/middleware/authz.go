package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/member"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// RequirePermission checks that the authenticated user has the required permission.
// Supports both JWT (role-based) and API key (permissions array) auth methods.
func RequirePermission(perm member.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		authMethod := GetAuthMethod(c)

		// API key path: check permissions array.
		if authMethod == AuthMethodAPIKey {
			perms := GetAPIKeyPermissions(c)
			for _, p := range perms {
				if p == string(perm) {
					c.Next()
					return
				}
			}
			apiErr := apierror.NewForbidden("api key lacks required permission", "error.insufficientKeyPermission")
			apiErr.RequestID = GetRequestID(c)
			c.AbortWithStatusJSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
			return
		}

		// JWT path: check role-based permissions.
		role := GetUserRole(c)
		if role == "" {
			apiErr := apierror.NewForbidden("no role assigned", "error.noRole")
			apiErr.RequestID = GetRequestID(c)
			c.AbortWithStatusJSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
			return
		}

		if !member.HasPermission(role, perm) {
			apiErr := apierror.NewForbidden("insufficient permissions", "error.forbidden")
			apiErr.RequestID = GetRequestID(c)
			c.AbortWithStatusJSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
			return
		}

		c.Next()
	}
}

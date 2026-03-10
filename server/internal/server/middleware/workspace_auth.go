package middleware

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/config"
	"github.com/rendis/feature-evaluator/internal/domain/member"
	"github.com/rendis/feature-evaluator/internal/domain/workspace"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// WorkspaceReadAuth authenticates the user for workspace list/read operations without requiring a workspace member.
func WorkspaceReadAuth(cfg config.AuthConfig, jwtValidator *JWTValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.Disabled {
			c.Set(CtxUserEmail, cfg.DevUserEmail)
			c.Set(CtxUserRole, member.Role(cfg.DevUserRole))
			c.Set(CtxAuthenticated, true)
			c.Set(CtxAuthMethod, AuthMethodJWT)
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			apiErr := apierror.NewUnauthorized("missing or invalid authorization header", "error.unauthorized")
			apiErr.RequestID = GetRequestID(c)
			c.AbortWithStatusJSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := jwtValidator.Validate(token)
		if err != nil {
			slog.Warn("workspace JWT validation failed", "error", err)
			apiErr := apierror.NewUnauthorized("invalid bearer token", "error.invalidToken")
			apiErr.RequestID = GetRequestID(c)
			c.AbortWithStatusJSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
			return
		}

		c.Set(CtxUserEmail, claims.Email)
		c.Set(CtxAuthenticated, true)
		c.Set(CtxAuthMethod, AuthMethodJWT)
		c.Next()
	}
}

// WorkspaceManageAuth authenticates workspace mutations. If there are no active workspaces,
// any authenticated user can bootstrap the first one as owner.
//
//nolint:funlen,gocognit // Workspace bootstrap, archived checks, and role resolution share one auth gate by design.
func WorkspaceManageAuth(
	cfg config.AuthConfig,
	memberSvc *member.Service,
	workspaceSvc *workspace.Service,
	jwtValidator *JWTValidator,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.Disabled {
			c.Set(CtxUserEmail, cfg.DevUserEmail)
			c.Set(CtxUserRole, member.Role(cfg.DevUserRole))
			c.Set(CtxAuthenticated, true)
			c.Set(CtxAuthMethod, AuthMethodJWT)
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			apiErr := apierror.NewUnauthorized("missing or invalid authorization header", "error.unauthorized")
			apiErr.RequestID = GetRequestID(c)
			c.AbortWithStatusJSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := jwtValidator.Validate(token)
		if err != nil {
			slog.Warn("workspace JWT validation failed", "error", err)
			apiErr := apierror.NewUnauthorized("invalid bearer token", "error.invalidToken")
			apiErr.RequestID = GetRequestID(c)
			c.AbortWithStatusJSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
			return
		}

		activeCount, err := workspaceSvc.CountActive(c.Request.Context())
		if err != nil {
			respErr := apierror.NewInternal("workspace bootstrap lookup failed", "error.internal")
			respErr.RequestID = GetRequestID(c)
			c.AbortWithStatusJSON(respErr.HTTPStatus, apierror.ErrorResponse{Error: respErr})
			return
		}

		c.Set(CtxUserEmail, claims.Email)
		c.Set(CtxAuthenticated, true)
		c.Set(CtxAuthMethod, AuthMethodJWT)

		if activeCount == 0 {
			c.Set(CtxUserRole, member.RoleOwner)
			c.Next()
			return
		}

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

		m := LookupOrClaimMember(c, memberSvc, claims.Email, claims.Name)
		if m == nil {
			return
		}

		c.Set(CtxUserEmail, m.Email)
		c.Set(CtxUserRole, m.Role)
		c.Next()
	}
}

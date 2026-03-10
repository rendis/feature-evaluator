package middleware

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/config"
	"github.com/rendis/feature-evaluator/internal/domain/apikey"
	"github.com/rendis/feature-evaluator/internal/domain/member"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// Context keys for auth information.
const (
	CtxUserEmail         = "userEmail"
	CtxUserRole          = "userRole"
	CtxAuthenticated     = "authenticated"
	CtxAuthMethod        = "authMethod"
	CtxBearerToken       = "bearerToken"
	CtxRawAPIKey         = "rawApiKey"
	CtxAPIKeyID          = "apiKeyID"
	CtxAPIKeyPermissions = "apiKeyPermissions"
)

// Auth method values.
const (
	AuthMethodJWT    = "jwt"
	AuthMethodAPIKey = "apikey"
)

// Auth validates the Authorization header or dev mode mock.
// In dev mode (AUTH_DISABLED=true), it uses DEV_USER_EMAIL and DEV_USER_ROLE.
// In prod mode, it validates the JWT token against the OIDC provider,
// looks up the member by email, and auto-claims ownership if the workspace is empty.
func Auth(cfg config.AuthConfig, memberSvc *member.Service, jwtValidator *JWTValidator) gin.HandlerFunc {
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
			slog.Warn("admin JWT validation failed", "error", err)
			apiErr := apierror.NewUnauthorized("invalid bearer token", "error.invalidToken")
			apiErr.RequestID = GetRequestID(c)
			c.AbortWithStatusJSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
			return
		}

		email := claims.Email
		displayName := claims.Name

		m := LookupOrClaimMember(c, memberSvc, email, displayName)
		if m == nil {
			return // response already sent by LookupOrClaimMember
		}

		c.Set(CtxUserEmail, m.Email)
		c.Set(CtxUserRole, m.Role)
		c.Set(CtxAuthenticated, true)
		c.Set(CtxAuthMethod, AuthMethodJWT)
		c.Set(CtxBearerToken, token)
		c.Next()
	}
}

// AdminAuth validates authentication for admin endpoints.
// Accepts either Bearer JWT token or X-Api-Key header (admin keys only).
// JWT path: validates token, looks up member, sets role.
// API key path: validates key, checks type==admin, sets permissions context.
func AdminAuth(cfg config.AuthConfig, memberSvc *member.Service, jwtValidator *JWTValidator, apiKeySvc *apikey.Service) gin.HandlerFunc {
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
		apiKeyHeader := getAPIKeyHeader(c)

		// JWT path.
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := jwtValidator.Validate(token)
			if err != nil {
				slog.Warn("admin JWT validation failed", "error", err)
				apiErr := apierror.NewUnauthorized("invalid bearer token", "error.invalidToken")
				apiErr.RequestID = GetRequestID(c)
				c.AbortWithStatusJSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
				return
			}

			email := claims.Email
			displayName := claims.Name

			m := LookupOrClaimMember(c, memberSvc, email, displayName)
			if m == nil {
				return
			}

			c.Set(CtxUserEmail, m.Email)
			c.Set(CtxUserRole, m.Role)
			c.Set(CtxAuthenticated, true)
			c.Set(CtxAuthMethod, AuthMethodJWT)
			c.Set(CtxBearerToken, token)
			c.Next()
			return
		}

		// API key path.
		if apiKeyHeader != "" {
			key, err := apiKeySvc.Validate(c.Request.Context(), apiKeyHeader)
			if err != nil {
				slog.Warn("admin API key validation failed", "error", err)
				apiErr := apierror.NewUnauthorized("invalid api key", "error.invalidApiKey")
				apiErr.RequestID = GetRequestID(c)
				c.AbortWithStatusJSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
				return
			}

			if key.Type != apikey.KeyTypeAdmin {
				//nolint:gosec // Structured logging here records validated API-key metadata, not executable input.
				slog.Warn("eval key used on admin endpoint", "keyID", key.ID, "keyName", key.Name)
				apiErr := apierror.NewForbidden("eval keys cannot access admin endpoints", "error.evalKeyForbidden")
				apiErr.RequestID = GetRequestID(c)
				c.AbortWithStatusJSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
				return
			}

			c.Set(CtxAuthenticated, true)
			c.Set(CtxAuthMethod, AuthMethodAPIKey)
			c.Set(CtxRawAPIKey, apiKeyHeader)
			c.Set(CtxAPIKeyID, key.ID)
			c.Set(CtxAPIKeyPermissions, key.Permissions)
			c.Set(CtxUserEmail, "apikey:"+key.Name)

			// Fire-and-forget lastUsedAt update.
			keyID := key.ID
			go apiKeySvc.UpdateLastUsed(c.Request.Context(), keyID)

			c.Next()
			return
		}

		apiErr := apierror.NewUnauthorized(
			"authorization required: provide Bearer token or X-Api-Key header",
			"error.adminUnauthorized",
		)
		apiErr.RequestID = GetRequestID(c)
		c.AbortWithStatusJSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
	}
}

// LookupOrClaimMember finds the member by email. If not found and the collection
// is empty, it auto-creates the first user as owner (first-user auto-claim).
// Returns the member or nil (in which case the request is already aborted).
func LookupOrClaimMember(c *gin.Context, memberSvc *member.Service, email, displayName string) *member.Member {
	m, err := memberSvc.GetByEmail(c.Request.Context(), email)
	if err == nil {
		return m
	}

	// Check if the error is a "not found" error.
	var apiErr *apierror.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != apierror.CodeNotFound {
		// Unexpected error during lookup.
		slog.Error("member lookup failed", "email", email, "error", err)
		respErr := apierror.NewInternal("member lookup failed", "error.internal")
		respErr.RequestID = GetRequestID(c)
		c.AbortWithStatusJSON(respErr.HTTPStatus, apierror.ErrorResponse{Error: respErr})
		return nil
	}

	// Member not found — attempt first-user auto-claim.
	claimed, claimErr := memberSvc.ClaimOwnership(c.Request.Context(), email, displayName)
	if claimErr != nil {
		var claimAPIErr *apierror.APIError
		if errors.As(claimErr, &claimAPIErr) {
			claimAPIErr.RequestID = GetRequestID(c)
			c.AbortWithStatusJSON(claimAPIErr.HTTPStatus, apierror.ErrorResponse{Error: claimAPIErr})
		} else {
			slog.Error("ownership claim failed", "email", email, "error", claimErr)
			respErr := apierror.NewInternal("ownership claim failed", "error.internal")
			respErr.RequestID = GetRequestID(c)
			c.AbortWithStatusJSON(respErr.HTTPStatus, apierror.ErrorResponse{Error: respErr})
		}
		return nil
	}

	return claimed
}

// EvalAuth preserves raw eval request auth headers without validating them globally.
// Feature-bound auth profiles decide whether the incoming request is authenticated.
func EvalAuth(cfg config.AuthConfig, _ *JWTValidator, _ *apikey.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.Disabled {
			c.Next()
			return
		}
		c.Next()
	}
}

// GetUserEmail returns the authenticated user's email from context.
func GetUserEmail(c *gin.Context) string {
	if v, ok := c.Get(CtxUserEmail); ok {
		return v.(string)
	}
	return ""
}

// GetUserRole returns the authenticated user's role from context.
func GetUserRole(c *gin.Context) member.Role {
	if v, ok := c.Get(CtxUserRole); ok {
		return v.(member.Role)
	}
	return ""
}

// GetAuthMethod returns the authentication method used for the request.
func GetAuthMethod(c *gin.Context) string {
	if v, ok := c.Get(CtxAuthMethod); ok {
		return v.(string)
	}
	return ""
}

// GetAPIKeyPermissions returns the API key permissions from context.
func GetAPIKeyPermissions(c *gin.Context) []string {
	if v, ok := c.Get(CtxAPIKeyPermissions); ok {
		return v.([]string)
	}
	return nil
}

// GetBearerToken returns the validated bearer token from context.
func GetBearerToken(c *gin.Context) string {
	if v, ok := c.Get(CtxBearerToken); ok {
		return v.(string)
	}
	return ""
}

// GetRawAPIKey returns the validated raw API key from context.
func GetRawAPIKey(c *gin.Context) string {
	if v, ok := c.Get(CtxRawAPIKey); ok {
		return v.(string)
	}
	return ""
}

// IsAuthenticated returns whether the request is authenticated.
func IsAuthenticated(c *gin.Context) bool {
	if v, ok := c.Get(CtxAuthenticated); ok {
		return v.(bool)
	}
	return false
}

// getAPIKeyHeader returns the API key from request headers.
// Checks X-Api-Key first, then X-API-Key for OFREP spec compatibility.
func getAPIKeyHeader(c *gin.Context) string {
	if v := c.GetHeader("X-Api-Key"); v != "" {
		return v
	}
	return c.GetHeader("X-API-Key")
}

package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/authprofile"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/external"
	"github.com/rendis/feature-evaluator/internal/incomingauth"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
)

// AuthProfileHandler handles auth profile CRUD and testing.
type AuthProfileHandler struct {
	svc       *authprofile.Service
	validator *incomingauth.Validator
	extCaller *external.Caller
}

// NewAuthProfileHandler creates a new AuthProfileHandler.
func NewAuthProfileHandler(
	svc *authprofile.Service,
	validator *incomingauth.Validator,
	extCaller *external.Caller,
) *AuthProfileHandler {
	return &AuthProfileHandler{svc: svc, validator: validator, extCaller: extCaller}
}

// List returns all auth profiles in the current workspace.
func (h *AuthProfileHandler) List(c *gin.Context) {
	profiles, err := h.svc.List(c.Request.Context())
	if err != nil {
		slog.Error("listing auth profiles", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	data := make([]dto.AuthProfileResponse, 0, len(profiles))
	for i := range profiles {
		data = append(data, dto.ToAuthProfileResponse(&profiles[i]))
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

// Get returns a single auth profile.
func (h *AuthProfileHandler) Get(c *gin.Context) {
	profile, err := h.svc.GetByKey(c.Request.Context(), c.Param("key"))
	if err != nil {
		dto.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ToAuthProfileResponse(profile))
}

// Create creates a new auth profile.
func (h *AuthProfileHandler) Create(c *gin.Context) {
	var req dto.CreateAuthProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	profile := &authprofile.Profile{
		Key:             req.Key,
		Name:            req.Name,
		Active:          req.Active,
		Type:            req.Type,
		Config:          req.Config,
		CacheTTLSeconds: req.CacheTTLSeconds,
		CreatedBy:       middleware.GetUserEmail(c),
		UpdatedBy:       middleware.GetUserEmail(c),
	}
	if err := h.svc.Create(c.Request.Context(), profile, req.SecretPayload); err != nil {
		slog.Error("creating auth profile", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToAuthProfileResponse(profile))
}

// Update updates an existing auth profile.
func (h *AuthProfileHandler) Update(c *gin.Context) {
	currentKey := c.Param("key")
	var req dto.UpdateAuthProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	profile := &authprofile.Profile{
		Key:             req.Key,
		Name:            req.Name,
		Active:          req.Active,
		Type:            req.Type,
		Config:          req.Config,
		CacheTTLSeconds: req.CacheTTLSeconds,
		UpdatedBy:       middleware.GetUserEmail(c),
	}
	replaceSecret := req.ReplaceSecret || req.SecretPayload != nil
	if err := h.svc.Update(c.Request.Context(), currentKey, profile, req.SecretPayload, replaceSecret); err != nil {
		slog.Error("updating auth profile", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	updated, err := h.svc.GetByKey(c.Request.Context(), profile.Key)
	if err != nil {
		dto.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ToAuthProfileResponse(updated))
}

// Delete removes an auth profile.
func (h *AuthProfileHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("key")); err != nil {
		slog.Error("deleting auth profile", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "auth profile deleted"})
}

// Test validates a draft auth profile against a simulated eval request.
func (h *AuthProfileHandler) Test(c *gin.Context) {
	var req dto.TestAuthProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}
	profile := &authprofile.Profile{
		Key:             c.Param("key"),
		Name:            req.Name,
		Active:          req.Active,
		Type:            req.Type,
		Config:          req.Config,
		CacheTTLSeconds: req.CacheTTLSeconds,
	}
	result, err := h.validator.ValidateDraft(c.Request.Context(), profile, req.SecretPayload, map[string]any{
		"headers": normalizeTestStringMap(req.TestRequest.Headers),
		"query":   normalizeTestStringMap(req.TestRequest.Query),
		"body":    req.TestRequest.Body,
	})
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.AuthProfileTestResponse{
		OK:         result.Authenticated,
		Attempted:  result.Attempted,
		Cached:     result.Cached,
		HTTPStatus: result.HTTPStatus,
		Details:    result.Details,
	})
}

func normalizeTestStringMap(values map[string]string) map[string]any {
	if len(values) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}

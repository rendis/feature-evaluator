package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/apikey"
	"github.com/rendis/feature-evaluator/internal/domain/member"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
)

// APIKeyHandler handles API key management endpoints.
type APIKeyHandler struct {
	svc *apikey.Service
}

// NewAPIKeyHandler creates a new APIKeyHandler.
func NewAPIKeyHandler(svc *apikey.Service) *APIKeyHandler {
	return &APIKeyHandler{svc: svc}
}

// Create generates a new API key and returns the plaintext (shown once).
func (h *APIKeyHandler) Create(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Type        string   `json:"type" binding:"required"`
		Permissions []string `json:"permissions"`
		Description string   `json:"description"`
		ExpiresAt   *string  `json:"expiresAt"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	}

	// Collect creator's permissions from their role.
	role := middleware.GetUserRole(c)
	createdByPermissions := permissionsToStrings(member.GetPermissions(role))

	plaintext, key, err := h.svc.GenerateKey(
		c.Request.Context(),
		req.Name,
		apikey.KeyType(req.Type),
		req.Permissions,
		req.Description,
		middleware.GetUserEmail(c),
		createdByPermissions,
		expiresAt,
	)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"key":         plaintext,
		"id":          key.ID,
		"name":        key.Name,
		"prefix":      key.Prefix,
		"type":        key.Type,
		"description": key.Description,
		"permissions": key.Permissions,
		"createdBy":   key.CreatedBy,
		"createdAt":   key.CreatedAt,
		"expiresAt":   key.ExpiresAt,
	})
}

// Rotate replaces the secret of an existing API key.
// The old key is immediately invalid. Returns the new plaintext (shown once).
func (h *APIKeyHandler) Rotate(c *gin.Context) {
	id := c.Param("id")

	plaintext, key, err := h.svc.Rotate(c.Request.Context(), id)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key":         plaintext,
		"id":          key.ID,
		"name":        key.Name,
		"prefix":      key.Prefix,
		"type":        key.Type,
		"description": key.Description,
		"permissions": key.Permissions,
		"createdBy":   key.CreatedBy,
		"createdAt":   key.CreatedAt,
		"expiresAt":   key.ExpiresAt,
	})
}

// List returns all API keys (without hashes).
func (h *APIKeyHandler) List(c *gin.Context) {
	keys, err := h.svc.List(c.Request.Context())
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": keys})
}

// Revoke marks an API key as revoked.
func (h *APIKeyHandler) Revoke(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Revoke(c.Request.Context(), id); err != nil {
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "api key revoked"})
}

// permissionsToStrings converts a slice of Permission to a slice of string.
func permissionsToStrings(perms []member.Permission) []string {
	out := make([]string, len(perms))
	for i, p := range perms {
		out[i] = string(p)
	}
	return out
}

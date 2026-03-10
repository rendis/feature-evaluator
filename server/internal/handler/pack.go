package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/changelog"
	"github.com/rendis/feature-evaluator/internal/domain/pack"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// PackHandler handles pack CRUD and activation endpoints.
type PackHandler struct {
	svc          *pack.Service
	changelogSvc *changelog.Service
}

// NewPackHandler creates a new PackHandler.
func NewPackHandler(svc *pack.Service, changelogSvc *changelog.Service) *PackHandler {
	return &PackHandler{svc: svc, changelogSvc: changelogSvc}
}

// List returns all packs.
func (h *PackHandler) List(c *gin.Context) {
	packs, err := h.svc.List(c.Request.Context())
	if err != nil {
		slog.Error("listing packs", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	data := make([]dto.PackResponse, 0, len(packs))
	for i := range packs {
		data = append(data, dto.ToPackResponse(&packs[i]))
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// Get returns a single pack.
func (h *PackHandler) Get(c *gin.Context) {
	key := c.Param("key")
	p, err := h.svc.GetByKey(c.Request.Context(), key)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToPackResponse(p))
}

// Create creates a new pack.
func (h *PackHandler) Create(c *gin.Context) {
	var req dto.CreatePackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	featureKeys := req.FeatureKeys
	if featureKeys == nil {
		featureKeys = []string{}
	}

	p := &pack.Pack{
		Key:         req.Key,
		Name:        req.Name,
		Description: req.Description,
		FeatureKeys: featureKeys,
		Enabled:     req.Enabled,
		Metadata:    req.Metadata,
		CreatedBy:   middleware.GetUserEmail(c),
		UpdatedBy:   middleware.GetUserEmail(c),
	}

	if err := h.svc.Create(c.Request.Context(), p); err != nil {
		slog.Error("creating pack", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntityPack,
		EntityKey:  p.Key,
		Action:     changelog.ActionCreate,
	})

	c.JSON(http.StatusCreated, dto.ToPackResponse(p))
}

// Update updates an existing pack.
func (h *PackHandler) Update(c *gin.Context) {
	key := c.Param("key")
	var req dto.UpdatePackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	existing, err := h.svc.GetByKey(c.Request.Context(), key)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	featureKeys := req.FeatureKeys
	if featureKeys == nil {
		featureKeys = []string{}
	}

	existing.Name = req.Name
	existing.Description = req.Description
	existing.FeatureKeys = featureKeys
	existing.Enabled = req.Enabled
	existing.Metadata = req.Metadata
	existing.UpdatedBy = middleware.GetUserEmail(c)

	if err := h.svc.Update(c.Request.Context(), existing); err != nil {
		slog.Error("updating pack", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntityPack,
		EntityKey:  key,
		Action:     changelog.ActionUpdate,
	})

	c.JSON(http.StatusOK, dto.ToPackResponse(existing))
}

// Delete removes a pack.
func (h *PackHandler) Delete(c *gin.Context) {
	key := c.Param("key")
	if err := h.svc.Delete(c.Request.Context(), key); err != nil {
		slog.Error("deleting pack", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntityPack,
		EntityKey:  key,
		Action:     changelog.ActionDelete,
	})

	c.JSON(http.StatusOK, gin.H{"message": "pack deleted"})
}

// Toggle enables or disables a pack.
func (h *PackHandler) Toggle(c *gin.Context) {
	key := c.Param("key")
	var req dto.ToggleFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	if err := h.svc.Toggle(c.Request.Context(), key, req.Enabled, middleware.GetUserEmail(c)); err != nil {
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntityPack,
		EntityKey:  key,
		Action:     changelog.ActionToggle,
		FieldChanges: []changelog.FieldChange{
			{Field: "enabled", NewValue: req.Enabled},
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "pack toggled", "enabled": req.Enabled})
}

// Activate creates a pack activation for a target.
func (h *PackHandler) Activate(c *gin.Context) {
	key := c.Param("key")
	var req dto.ActivatePackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	expiresAt, err := dto.ParseTimePtr(req.ExpiresAt)
	if err != nil {
		dto.RespondError(c, apierror.NewBadRequest(err.Error(), "error.invalidExpiresAt"))
		return
	}

	a := &pack.Activation{
		PackKey:     key,
		TargetType:  pack.TargetType(req.TargetType),
		TargetID:    req.TargetID,
		ActivatedBy: middleware.GetUserEmail(c),
		ExpiresAt:   expiresAt,
		Metadata:    req.Metadata,
	}

	if err := h.svc.Activate(c.Request.Context(), a); err != nil {
		slog.Error("activating pack", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToActivationResponse(a))
}

// Deactivate removes a pack activation for a target.
func (h *PackHandler) Deactivate(c *gin.Context) {
	key := c.Param("key")
	var req dto.DeactivatePackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	if err := h.svc.Deactivate(c.Request.Context(), key, pack.TargetType(req.TargetType), req.TargetID); err != nil {
		slog.Error("deactivating pack", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "pack deactivated"})
}

// ListActivations returns all activations for a pack.
func (h *PackHandler) ListActivations(c *gin.Context) {
	key := c.Param("key")
	activations, err := h.svc.ListActivations(c.Request.Context(), key)
	if err != nil {
		slog.Error("listing pack activations", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	data := make([]dto.ActivationResponse, 0, len(activations))
	for i := range activations {
		data = append(data, dto.ToActivationResponse(&activations[i]))
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// ByTarget returns all activations for a given target.
func (h *PackHandler) ByTarget(c *gin.Context) {
	targetType := c.Query("type")
	targetID := c.Query("id")

	if !pack.ValidTargetType(targetType) {
		dto.RespondError(c, apierror.NewBadRequest("invalid target type", "error.invalidTargetType"))
		return
	}
	if targetID == "" {
		dto.RespondError(c, apierror.NewBadRequest("target ID is required", "error.targetIdRequired"))
		return
	}

	activations, err := h.svc.FindByTarget(c.Request.Context(), pack.TargetType(targetType), targetID)
	if err != nil {
		slog.Error("finding pack activations by target", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	data := make([]dto.ActivationResponse, 0, len(activations))
	for i := range activations {
		data = append(data, dto.ToActivationResponse(&activations[i]))
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

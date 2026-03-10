package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/changelog"
	"github.com/rendis/feature-evaluator/internal/domain/pack"
	"github.com/rendis/feature-evaluator/internal/domain/tier"
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

// buildTierRef resolves a tier ref for a pack's tier key.
func (h *PackHandler) buildTierRef(tierKey *string) *dto.TierRef {
	if tierKey == nil || *tierKey == "" {
		return nil
	}
	td := tier.FindByKey(*tierKey)
	if td == nil {
		return nil
	}
	ref := dto.ToTierRef(td)
	return &ref
}

// resolveFeatureCount returns the total resolved feature count including inherited packs.
func (h *PackHandler) resolveFeatureCount(c *gin.Context, packKey string) int {
	resolvedKeys, err := h.svc.ResolveFeatureKeys(c.Request.Context(), packKey)
	if err != nil {
		//nolint:gosec // Structured logging records the pack key for operator debugging only.
		slog.Warn("resolving feature keys for pack", "packKey", packKey, "error", err, "requestId", middleware.GetRequestID(c))
		return 0
	}
	return len(resolvedKeys)
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
		tierRef := h.buildTierRef(packs[i].TierKey)
		resolvedCount := h.resolveFeatureCount(c, packs[i].Key)
		data = append(data, dto.ToPackResponse(&packs[i], tierRef, resolvedCount))
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

	tierRef := h.buildTierRef(p.TierKey)
	resolvedCount := h.resolveFeatureCount(c, p.Key)
	c.JSON(http.StatusOK, dto.ToPackResponse(p, tierRef, resolvedCount))
}

// Create creates a new pack.
func (h *PackHandler) Create(c *gin.Context) {
	var req dto.CreatePackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	if req.TierKey != nil && *req.TierKey != "" && !tier.ValidKey(*req.TierKey) {
		dto.RespondError(c, apierror.NewBadRequest("invalid tier key", "error.invalidTierKey"))
		return
	}

	featureKeys := req.FeatureKeys
	if featureKeys == nil {
		featureKeys = []string{}
	}

	trialUntil, err := dto.ParseTimePtr(req.TrialUntil)
	if err != nil {
		dto.RespondError(c, apierror.NewBadRequest(err.Error(), "error.invalidTrialUntil"))
		return
	}

	p := &pack.Pack{
		Key:          req.Key,
		Name:         req.Name,
		Description:  req.Description,
		FeatureKeys:  featureKeys,
		Enabled:      req.Enabled,
		Metadata:     req.Metadata,
		TierKey:      req.TierKey,
		InheritsFrom: req.InheritsFrom,
		TrialUntil:   trialUntil,
		CreatedBy:    middleware.GetUserEmail(c),
		UpdatedBy:    middleware.GetUserEmail(c),
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

	tierRef := h.buildTierRef(p.TierKey)
	resolvedCount := h.resolveFeatureCount(c, p.Key)
	c.JSON(http.StatusCreated, dto.ToPackResponse(p, tierRef, resolvedCount))
}

// Update updates an existing pack.
func (h *PackHandler) Update(c *gin.Context) {
	key := c.Param("key")
	var req dto.UpdatePackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	if req.TierKey != nil && *req.TierKey != "" && !tier.ValidKey(*req.TierKey) {
		dto.RespondError(c, apierror.NewBadRequest("invalid tier key", "error.invalidTierKey"))
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

	trialUntil, err := dto.ParseTimePtr(req.TrialUntil)
	if err != nil {
		dto.RespondError(c, apierror.NewBadRequest(err.Error(), "error.invalidTrialUntil"))
		return
	}

	existing.Name = req.Name
	existing.Description = req.Description
	existing.FeatureKeys = featureKeys
	existing.Enabled = req.Enabled
	existing.Metadata = req.Metadata
	existing.TierKey = req.TierKey
	existing.InheritsFrom = req.InheritsFrom
	existing.TrialUntil = trialUntil
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

	tierRef := h.buildTierRef(existing.TierKey)
	resolvedCount := h.resolveFeatureCount(c, existing.Key)
	c.JSON(http.StatusOK, dto.ToPackResponse(existing, tierRef, resolvedCount))
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

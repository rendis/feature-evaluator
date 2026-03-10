package handler

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/tier"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
)

// TierHandler handles tier-related HTTP endpoints.
type TierHandler struct {
	svc *tier.Service
}

// NewTierHandler creates a new TierHandler.
func NewTierHandler(svc *tier.Service) *TierHandler {
	return &TierHandler{svc: svc}
}

// List returns all tiers, optionally filtered by search.
func (h *TierHandler) List(c *gin.Context) {
	search := c.Query("search")
	tiers, err := h.svc.List(c.Request.Context(), search)
	if err != nil {
		slog.Error("listing tiers", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	data := make([]dto.TierResponse, 0, len(tiers))
	for i := range tiers {
		data = append(data, dto.ToTierResponse(&tiers[i]))
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// Get returns a single tier by key.
func (h *TierHandler) Get(c *gin.Context) {
	key := c.Param("key")
	t, err := h.svc.FindByKey(c.Request.Context(), key)
	if err != nil {
		slog.Error("getting tier", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToTierResponse(t))
}

// Create creates a new tier.
func (h *TierHandler) Create(c *gin.Context) {
	var req dto.CreateTierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	t, err := h.svc.Create(c.Request.Context(), req.Name, req.Level, req.Color, req.Icon, middleware.GetUserEmail(c))
	if err != nil {
		slog.Error("creating tier", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToTierResponse(t))
}

// Update updates a tier's name, level, color, and icon.
func (h *TierHandler) Update(c *gin.Context) {
	key := c.Param("key")
	var req dto.UpdateTierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	t, err := h.svc.Update(c.Request.Context(), key, req.Name, req.Level, req.Color, req.Icon)
	if err != nil {
		slog.Error("updating tier", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToTierResponse(t))
}

// Delete removes a tier.
func (h *TierHandler) Delete(c *gin.Context) {
	key := c.Param("key")
	if err := h.svc.Delete(c.Request.Context(), key); err != nil {
		slog.Error("deleting tier", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "tier deleted"})
}

// UploadIcon uploads a custom tier icon.
func (h *TierHandler) UploadIcon(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		slog.Error("reading icon upload", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}
	defer file.Close()

	name := c.PostForm("name")
	data, err := io.ReadAll(file)
	if err != nil {
		slog.Error("reading icon data", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	icon, err := h.svc.UploadIcon(c.Request.Context(), name, header.Header.Get("Content-Type"), data, middleware.GetUserEmail(c))
	if err != nil {
		slog.Error("uploading tier icon", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToTierIconResponse(icon))
}

// ListIcons returns all custom tier icons and built-in icon keys.
func (h *TierHandler) ListIcons(c *gin.Context) {
	icons, err := h.svc.ListIcons(c.Request.Context())
	if err != nil {
		slog.Error("listing tier icons", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	data := make([]dto.TierIconResponse, 0, len(icons))
	for i := range icons {
		data = append(data, dto.ToTierIconResponse(&icons[i]))
	}

	c.JSON(http.StatusOK, gin.H{"data": data, "builtinIcons": tier.BuiltinIcons})
}

// DeleteIcon removes a custom tier icon.
func (h *TierHandler) DeleteIcon(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.DeleteIcon(c.Request.Context(), id); err != nil {
		slog.Error("deleting tier icon", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "icon deleted"})
}

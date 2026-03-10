package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/tag"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
)

// TagHandler handles tag CRUD endpoints.
type TagHandler struct {
	svc *tag.Service
}

// NewTagHandler creates a new TagHandler.
func NewTagHandler(svc *tag.Service) *TagHandler {
	return &TagHandler{svc: svc}
}

// List returns all tags, optionally filtered by search.
func (h *TagHandler) List(c *gin.Context) {
	search := c.Query("search")
	tags, err := h.svc.List(c.Request.Context(), search)
	if err != nil {
		slog.Error("listing tags", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	data := make([]dto.TagDetailResponse, 0, len(tags))
	for i := range tags {
		data = append(data, dto.ToTagResponse(&tags[i]))
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// Create creates a new tag.
func (h *TagHandler) Create(c *gin.Context) {
	var req struct {
		Name  string `json:"name" binding:"required"`
		Color string `json:"color" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	t, err := h.svc.Create(c.Request.Context(), req.Name, req.Color, middleware.GetUserEmail(c))
	if err != nil {
		slog.Error("creating tag", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToTagResponse(t))
}

// Update updates a tag's name and color.
func (h *TagHandler) Update(c *gin.Context) {
	key := c.Param("key")
	var req struct {
		Name  string `json:"name" binding:"required"`
		Color string `json:"color" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	t, err := h.svc.Update(c.Request.Context(), key, req.Name, req.Color)
	if err != nil {
		slog.Error("updating tag", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToTagResponse(t))
}

// Delete removes a tag.
func (h *TagHandler) Delete(c *gin.Context) {
	key := c.Param("key")
	if err := h.svc.Delete(c.Request.Context(), key); err != nil {
		slog.Error("deleting tag", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "tag deleted"})
}

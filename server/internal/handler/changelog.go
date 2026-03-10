package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/changelog"
	"github.com/rendis/feature-evaluator/internal/dto"
)

// ChangelogHandler handles changelog endpoints.
type ChangelogHandler struct {
	svc *changelog.Service
}

// NewChangelogHandler creates a new ChangelogHandler.
func NewChangelogHandler(svc *changelog.Service) *ChangelogHandler {
	return &ChangelogHandler{svc: svc}
}

// List returns a paginated, filtered list of changelog entries.
func (h *ChangelogHandler) List(c *gin.Context) {
	params := changelog.ListParams{
		EntityType: c.Query("entityType"),
		EntityKey:  c.Query("entityKey"),
		Actor:      c.Query("actor"),
		Action:     c.Query("action"),
		From:       c.Query("from"),
		To:         c.Query("to"),
	}
	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		params.Page = page
	}
	if ps, err := strconv.Atoi(c.DefaultQuery("pageSize", "20")); err == nil {
		params.PageSize = ps
	}

	result, err := h.svc.List(c.Request.Context(), params)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	h.respondList(c, result)
}

// ListByEntity returns changelog entries for a specific entity.
func (h *ChangelogHandler) ListByEntity(c *gin.Context) {
	entityType := c.Param("entityType")
	entityKey := c.Param("entityKey")

	params := changelog.ListParams{
		Actor:  c.Query("actor"),
		Action: c.Query("action"),
		From:   c.Query("from"),
		To:     c.Query("to"),
	}
	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		params.Page = page
	}
	if ps, err := strconv.Atoi(c.DefaultQuery("pageSize", "20")); err == nil {
		params.PageSize = ps
	}

	result, err := h.svc.ListByEntity(c.Request.Context(), entityType, entityKey, params)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	h.respondList(c, result)
}

func (h *ChangelogHandler) respondList(c *gin.Context, result *changelog.ListResult) {
	data := make([]dto.ChangeEntryResponse, 0, len(result.Data))
	for i := range result.Data {
		data = append(data, dto.ToChangeEntryResponse(&result.Data[i]))
	}

	c.JSON(http.StatusOK, dto.ListResponse[dto.ChangeEntryResponse]{
		Data: data,
		Pagination: dto.PaginationResponse{
			Page:       result.Page,
			PageSize:   result.PageSize,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

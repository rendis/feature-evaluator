package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/changelog"
	"github.com/rendis/feature-evaluator/internal/domain/segment"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
)

// SegmentHandler handles segment CRUD endpoints.
type SegmentHandler struct {
	svc          *segment.Service
	changelogSvc *changelog.Service
}

// NewSegmentHandler creates a new SegmentHandler.
func NewSegmentHandler(svc *segment.Service, changelogSvc *changelog.Service) *SegmentHandler {
	return &SegmentHandler{svc: svc, changelogSvc: changelogSvc}
}

// List returns a paginated list of segments.
func (h *SegmentHandler) List(c *gin.Context) {
	params := segment.ListParams{
		Search: c.Query("search"),
	}
	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		params.Page = page
	}
	if ps, err := strconv.Atoi(c.DefaultQuery("pageSize", "20")); err == nil {
		params.PageSize = ps
	}

	result, err := h.svc.List(c.Request.Context(), params)
	if err != nil {
		slog.Error("listing segments", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	data := make([]dto.SegmentResponse, 0, len(result.Data))
	for i := range result.Data {
		data = append(data, dto.ToSegmentResponse(&result.Data[i]))
	}

	c.JSON(http.StatusOK, dto.ListResponse[dto.SegmentResponse]{
		Data: data,
		Pagination: dto.PaginationResponse{
			Page:       result.Page,
			PageSize:   result.PageSize,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

// Get returns a single segment.
func (h *SegmentHandler) Get(c *gin.Context) {
	key := c.Param("key")
	seg, err := h.svc.GetByKey(c.Request.Context(), key)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToSegmentResponse(seg))
}

// Create creates a new segment.
func (h *SegmentHandler) Create(c *gin.Context) {
	var req dto.CreateSegmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	seg := &segment.Segment{
		Key:         req.Key,
		Name:        req.Name,
		Description: req.Description,
		Metadata:    req.Metadata,
		CreatedBy:   middleware.GetUserEmail(c),
		UpdatedBy:   middleware.GetUserEmail(c),
	}

	if err := h.svc.Create(c.Request.Context(), seg); err != nil {
		slog.Error("creating segment", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntitySegment,
		EntityKey:  seg.Key,
		Action:     changelog.ActionCreate,
	})

	c.JSON(http.StatusCreated, dto.ToSegmentResponse(seg))
}

// Update updates an existing segment.
func (h *SegmentHandler) Update(c *gin.Context) {
	key := c.Param("key")
	var req dto.UpdateSegmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	seg := &segment.Segment{
		Key:         key,
		Name:        req.Name,
		Description: req.Description,
		Metadata:    req.Metadata,
		UpdatedBy:   middleware.GetUserEmail(c),
	}

	if err := h.svc.Update(c.Request.Context(), seg); err != nil {
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntitySegment,
		EntityKey:  key,
		Action:     changelog.ActionUpdate,
	})

	updated, _ := h.svc.GetByKey(c.Request.Context(), key)
	c.JSON(http.StatusOK, dto.ToSegmentResponse(updated))
}

// Delete removes a segment and its records.
func (h *SegmentHandler) Delete(c *gin.Context) {
	key := c.Param("key")
	if err := h.svc.Delete(c.Request.Context(), key); err != nil {
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntitySegment,
		EntityKey:  key,
		Action:     changelog.ActionDelete,
	})

	c.JSON(http.StatusOK, gin.H{"message": "segment deleted"})
}

// GetSchema returns the stored schema metadata for a segment.
func (h *SegmentHandler) GetSchema(c *gin.Context) {
	key := c.Param("key")
	seg, err := h.svc.GetByKey(c.Request.Context(), key)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToSegmentSchemaResponse(seg))
}

// ListRecords returns a paginated list of segment records.
func (h *SegmentHandler) ListRecords(c *gin.Context) {
	key := c.Param("key")
	params := segment.RecordListParams{
		SegmentKey: key,
		Query:      c.Query("q"),
	}
	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		params.Page = page
	}
	if ps, err := strconv.Atoi(c.DefaultQuery("pageSize", "20")); err == nil {
		params.PageSize = ps
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	result, err := h.svc.ListRecords(c.Request.Context(), params)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	data := make([]dto.SegmentRecordResponse, 0, len(result.Data))
	for i := range result.Data {
		data = append(data, dto.ToSegmentRecordResponse(&result.Data[i]))
	}

	c.JSON(http.StatusOK, dto.ListResponse[dto.SegmentRecordResponse]{
		Data: data,
		Pagination: dto.PaginationResponse{
			Page:       result.Page,
			PageSize:   result.PageSize,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

// ImportData imports a full dataset version into a segment.
func (h *SegmentHandler) ImportData(c *gin.Context) {
	key := c.Param("key")
	var req dto.ImportSegmentDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	mode := segment.ImportMode(req.Mode)
	if mode != segment.ImportModeReplace {
		dto.RespondError(c, fmt.Errorf("invalid import mode: %s", req.Mode))
		return
	}

	inserted, err := h.svc.ReplaceRecords(c.Request.Context(), key, segment.ReplaceInput{
		SourceType:    segment.SourceType(req.SourceType),
		RecordKeyPath: req.RecordKeyPath,
		Schema:        req.Schema,
		Records:       req.Records,
		UpdatedBy:     middleware.GetUserEmail(c),
	})
	if err != nil {
		slog.Error("importing segment data", "error", err, "segment", key, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	seg, _ := h.svc.GetByKey(c.Request.Context(), key)
	c.JSON(http.StatusOK, dto.ImportResultResponse{
		Inserted:       inserted,
		DatasetVersion: seg.ActiveDatasetVersion,
		PreviewFields:  seg.PreviewFields,
	})
}

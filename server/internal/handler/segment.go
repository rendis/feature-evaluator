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

// List godoc
// @Summary      List segments
// @Description  Returns a paginated list of segments
// @Tags         segments
// @Produce      json
// @Param        search    query     string  false  "Search filter"
// @Param        page      query     int     false  "Page number"   default(1)
// @Param        pageSize  query     int     false  "Page size"     default(20)
// @Success      200       {object}  dto.ListResponse[dto.SegmentResponse]
// @Failure      500       {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/segments [get]
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

// Get godoc
// @Summary      Get segment
// @Description  Returns a single segment by key
// @Tags         segments
// @Produce      json
// @Param        key  path      string  true  "Segment key"
// @Success      200  {object}  dto.SegmentResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/segments/{key} [get]
func (h *SegmentHandler) Get(c *gin.Context) {
	key := c.Param("key")
	seg, err := h.svc.GetByKey(c.Request.Context(), key)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToSegmentResponse(seg))
}

// Create godoc
// @Summary      Create segment
// @Description  Creates a new segment
// @Tags         segments
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateSegmentRequest  true  "Segment creation payload"
// @Success      201   {object}  dto.SegmentResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      409   {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/segments [post]
func (h *SegmentHandler) Create(c *gin.Context) {
	var req dto.CreateSegmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	seg := &segment.Segment{
		Key:                       req.Key,
		Name:                      req.Name,
		Description:               req.Description,
		Metadata:                  req.Metadata,
		MembershipCacheEnabled:    req.MembershipCacheEnabled,
		MembershipCacheTTLSeconds: req.MembershipCacheTTLSeconds,
		RecordCacheEnabled:        req.RecordCacheEnabled,
		RecordCacheTTLSeconds:     req.RecordCacheTTLSeconds,
		CreatedBy:                 middleware.GetUserEmail(c),
		UpdatedBy:                 middleware.GetUserEmail(c),
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

// Update godoc
// @Summary      Update segment
// @Description  Updates an existing segment by key
// @Tags         segments
// @Accept       json
// @Produce      json
// @Param        key   path      string                    true  "Segment key"
// @Param        body  body      dto.UpdateSegmentRequest   true  "Segment update payload"
// @Success      200   {object}  dto.SegmentResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/segments/{key} [put]
func (h *SegmentHandler) Update(c *gin.Context) {
	key := c.Param("key")
	var req dto.UpdateSegmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	seg := &segment.Segment{
		Key:                       key,
		Name:                      req.Name,
		Description:               req.Description,
		Metadata:                  req.Metadata,
		MembershipCacheEnabled:    req.MembershipCacheEnabled,
		MembershipCacheTTLSeconds: req.MembershipCacheTTLSeconds,
		RecordCacheEnabled:        req.RecordCacheEnabled,
		RecordCacheTTLSeconds:     req.RecordCacheTTLSeconds,
		UpdatedBy:                 middleware.GetUserEmail(c),
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

// Delete godoc
// @Summary      Delete segment
// @Description  Deletes a segment and all its records
// @Tags         segments
// @Produce      json
// @Param        key  path      string  true  "Segment key"
// @Success      200  {object}  dto.MessageResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/segments/{key} [delete]
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

// GetSchema godoc
// @Summary      Get segment schema
// @Description  Returns the stored schema metadata for a segment
// @Tags         segments
// @Produce      json
// @Param        key  path      string  true  "Segment key"
// @Success      200  {object}  dto.SegmentSchemaResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/segments/{key}/schema [get]
func (h *SegmentHandler) GetSchema(c *gin.Context) {
	key := c.Param("key")
	seg, err := h.svc.GetByKey(c.Request.Context(), key)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToSegmentSchemaResponse(seg))
}

// ListRecords godoc
// @Summary      List segment records
// @Description  Returns a paginated list of records for a segment
// @Tags         segments
// @Produce      json
// @Param        key       path      string  true   "Segment key"
// @Param        q         query     string  false  "Search query"
// @Param        page      query     int     false  "Page number"  default(1)
// @Param        pageSize  query     int     false  "Page size"    default(20)
// @Success      200       {object}  dto.ListResponse[dto.SegmentRecordResponse]
// @Failure      404       {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/segments/{key}/records [get]
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

// ImportData godoc
// @Summary      Import segment data
// @Description  Imports a full dataset version into a segment (replace mode)
// @Tags         segments
// @Accept       json
// @Produce      json
// @Param        key   path      string                       true  "Segment key"
// @Param        body  body      dto.ImportSegmentDataRequest  true  "Import payload"
// @Success      200   {object}  dto.ImportResultResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/segments/{key}/data/import [post]
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

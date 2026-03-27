package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/audit"
	evalmetrics "github.com/rendis/feature-evaluator/internal/domain/metrics"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
)

// FeatureObservabilityHandler handles observability endpoints for a feature.
type FeatureObservabilityHandler struct {
	svc *evalmetrics.ObservabilityService
}

// NewFeatureObservabilityHandler creates a new observability handler.
func NewFeatureObservabilityHandler(svc *evalmetrics.ObservabilityService) *FeatureObservabilityHandler {
	return &FeatureObservabilityHandler{svc: svc}
}

// Overview godoc
// @Summary Feature observability overview
// @Description Returns aggregated evaluation metrics for a feature
// @Tags features
// @Produce json
// @Param key path string true "Feature key"
// @Success 200 {object} dto.ObservabilityOverviewResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/features/{key}/observability/overview [get]
func (h *FeatureObservabilityHandler) Overview(c *gin.Context) {
	overview, err := h.svc.Overview(c.Request.Context(), c.Param("key"))
	if err != nil {
		slog.Error("loading feature observability overview", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ToObservabilityOverviewResponse(overview))
}

// Rules godoc
// @Summary Feature rule observability
// @Description Returns aggregated metrics for rules within a feature
// @Tags features
// @Produce json
// @Param key path string true "Feature key"
// @Success 200 {object} dto.DataResponse[[]dto.ObservabilityRuleResponse]
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/features/{key}/observability/rules [get]
func (h *FeatureObservabilityHandler) Rules(c *gin.Context) {
	rules, err := h.svc.Rules(c.Request.Context(), c.Param("key"))
	if err != nil {
		slog.Error("loading feature observability rules", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}
	data := make([]dto.ObservabilityRuleResponse, 0, len(rules))
	for i := range rules {
		data = append(data, dto.ToObservabilityRuleResponse(rules[i]))
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

// Rule godoc
// @Summary Feature rule observability detail
// @Description Returns aggregated metrics for a single rule
// @Tags features
// @Produce json
// @Param key path string true "Feature key"
// @Param ruleId path string true "Rule ID"
// @Success 200 {object} dto.ObservabilityRuleResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/features/{key}/observability/rules/{ruleId} [get]
func (h *FeatureObservabilityHandler) Rule(c *gin.Context) {
	rule, err := h.svc.Rule(c.Request.Context(), c.Param("key"), c.Param("ruleId"))
	if err != nil {
		slog.Error("loading feature observability rule", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ToObservabilityRuleResponse(*rule))
}

// Traces godoc
// @Summary Feature evaluation traces
// @Description Returns persisted sanitized traces for a feature
// @Tags features
// @Produce json
// @Param key path string true "Feature key"
// @Param ruleId query string false "Filter by rule ID"
// @Param search query string false "Search request ID"
// @Param cacheStatus query string false "Filter by cache status"
// @Param usedRedis query bool false "Filter by Redis usage"
// @Param from query string false "Start date (RFC3339)"
// @Param to query string false "End date (RFC3339)"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Success 200 {object} dto.ListResponse[dto.ObservabilityTraceResponse]
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/features/{key}/observability/traces [get]
func (h *FeatureObservabilityHandler) Traces(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	var usedRedis *bool
	if raw := c.Query("usedRedis"); raw != "" {
		if parsed, err := strconv.ParseBool(raw); err == nil {
			usedRedis = &parsed
		}
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	result, err := h.svc.Traces(c.Request.Context(), audit.TraceListParams{
		FeatureKey:  c.Param("key"),
		RuleID:      c.Query("ruleId"),
		Search:      c.Query("search"),
		CacheStatus: c.Query("cacheStatus"),
		UsedRedis:   usedRedis,
		From:        c.Query("from"),
		To:          c.Query("to"),
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		slog.Error("loading feature observability traces", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}
	data := make([]dto.ObservabilityTraceResponse, 0, len(result.Data))
	for i := range result.Data {
		data = append(data, dto.ToObservabilityTraceResponse(result.Data[i]))
	}
	c.JSON(http.StatusOK, dto.ListResponse[dto.ObservabilityTraceResponse]{
		Data: data,
		Pagination: dto.PaginationResponse{
			Page:       result.Page,
			PageSize:   result.PageSize,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

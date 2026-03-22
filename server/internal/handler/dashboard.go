package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/audit"
	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/domain/segment"
	"github.com/rendis/feature-evaluator/internal/dto"
)

// swagger type resolution
var _ dto.DashboardStatsResponse

type dashboardMetricsReader interface {
	DailyCount(ctx context.Context, day string) int64
	ReasonBreakdown(ctx context.Context, day string) map[string]int64
	CacheHits(ctx context.Context, day string) int64
	CacheMisses(ctx context.Context, day string) int64
	RateLimitRejects(ctx context.Context, day string) int64
	ExternalLatencyPercentiles(ctx context.Context, day string) (float64, float64)
	CircuitBreakerCounts(ctx context.Context, day string) (int64, int64)
}

// DashboardHandler handles dashboard aggregate endpoints.
type DashboardHandler struct {
	featureSvc *feature.Service
	segmentSvc *segment.Service
	auditSvc   *audit.Service
	metrics    dashboardMetricsReader
	probe      *dependencyProbe
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(
	featureSvc *feature.Service,
	segmentSvc *segment.Service,
	auditSvc *audit.Service,
	metrics dashboardMetricsReader,
	postgres postgresPinger,
	redis redisDependencyChecker,
) *DashboardHandler {
	return &DashboardHandler{
		featureSvc: featureSvc,
		segmentSvc: segmentSvc,
		auditSvc:   auditSvc,
		metrics:    metrics,
		probe:      newDependencyProbe(postgres, redis),
	}
}

// Stats godoc
// @Summary Dashboard stats
// @Description Returns aggregate counts of features, active features, segments, and segment members
// @Tags dashboard
// @Produce json
// @Success 200 {object} dto.DashboardStatsResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/dashboard/stats [get]
func (h *DashboardHandler) Stats(c *gin.Context) {
	ctx := c.Request.Context()

	allFeatures, err := h.featureSvc.List(ctx, feature.ListParams{Page: 1, PageSize: 1})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count features"})
		return
	}

	enabled := true
	activeFeatures, err := h.featureSvc.List(ctx, feature.ListParams{Page: 1, PageSize: 1, Enabled: &enabled})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count active features"})
		return
	}

	allSegments, err := h.segmentSvc.List(ctx, segment.ListParams{Page: 1, PageSize: 1})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count segments"})
		return
	}

	var totalMembers int64

	c.JSON(http.StatusOK, gin.H{
		"totalFeatures":       allFeatures.Total,
		"activeFeatures":      activeFeatures.Total,
		"totalSegments":       allSegments.Total,
		"totalSegmentMembers": totalMembers,
	})
}

// Activity godoc
// @Summary Dashboard activity
// @Description Returns the 10 most recently updated features for the activity feed
// @Tags dashboard
// @Produce json
// @Success 200 {array} dto.DashboardActivityItem
// @Security BearerAuth
// @Router /admin/dashboard/activity [get]
func (h *DashboardHandler) Activity(c *gin.Context) {
	ctx := c.Request.Context()

	recentFeatures, err := h.featureSvc.List(ctx, feature.ListParams{
		Page:      1,
		PageSize:  10,
		SortBy:    "updatedAt",
		SortOrder: "desc",
	})
	if err != nil {
		c.JSON(http.StatusOK, []any{})
		return
	}

	activities := make([]gin.H, 0, len(recentFeatures.Data))
	for _, f := range recentFeatures.Data {
		activities = append(activities, gin.H{
			"id":          f.ID,
			"type":        "feature_updated",
			"featureKey":  f.Key,
			"description": f.Name,
			"createdBy":   f.UpdatedBy,
			"createdAt":   f.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, activities)
}

// ErrorSummary godoc
// @Summary Dashboard error summary
// @Description Returns aggregated evaluation error counts for the dashboard
// @Tags dashboard
// @Produce json
// @Success 200 {object} dto.DashboardErrorSummaryResponse
// @Security BearerAuth
// @Router /admin/dashboard/error-summary [get]
func (h *DashboardHandler) ErrorSummary(c *gin.Context) {
	ctx := c.Request.Context()

	errors, err := h.auditSvc.List(ctx, audit.ListParams{Page: 1, PageSize: 1})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"total":  0,
			"byType": []any{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":  errors.Total,
		"byType": []any{},
	})
}

// Operations godoc
// @Summary Dashboard operations
// @Description Returns the current system status, dependency health, and runtime metrics snapshot
// @Tags dashboard
// @Produce json
// @Success 200 {object} dto.DashboardOperationsResponse
// @Security BearerAuth
// @Router /admin/dashboard/operations [get]
func (h *DashboardHandler) Operations(c *gin.Context) {
	ctx := c.Request.Context()
	snapshot := h.probe.Check(ctx)
	today := time.Now().UTC().Format("2006-01-02")

	hits := h.metrics.CacheHits(ctx, today)
	misses := h.metrics.CacheMisses(ctx, today)
	cacheHitRatio := 0.0
	if totalCacheAccess := hits + misses; totalCacheAccess > 0 {
		cacheHitRatio = float64(hits) / float64(totalCacheAccess)
	}

	externalP50, externalP95 := h.metrics.ExternalLatencyPercentiles(ctx, today)
	cbOpenEvents, cbCloseEvents := h.metrics.CircuitBreakerCounts(ctx, today)
	reasons := h.metrics.ReasonBreakdown(ctx, today)

	c.JSON(http.StatusOK, gin.H{
		"checkedAt":     snapshot.CheckedAt,
		"overallStatus": snapshot.OverallStatus,
		"services": gin.H{
			"postgresql": snapshot.PostgreSQL,
			"redis":      snapshot.Redis,
		},
		"metrics": gin.H{
			"evaluationsToday":          h.metrics.DailyCount(ctx, today),
			"errorsToday":               reasons["error"],
			"cacheHitRatio":             cacheHitRatio,
			"rateLimitRejectsToday":     h.metrics.RateLimitRejects(ctx, today),
			"externalP50Ms":             externalP50,
			"externalP95Ms":             externalP95,
			"circuitBreakerOpenEvents":  cbOpenEvents,
			"circuitBreakerCloseEvents": cbCloseEvents,
		},
	})
}

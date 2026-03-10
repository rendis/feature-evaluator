package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/audit"
	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/domain/segment"
)

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

// Stats returns aggregate counts for the dashboard.
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

// Activity returns recent feature changes for the dashboard.
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

// ErrorSummary returns aggregated error info for the dashboard.
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

// Operations returns the current system status and runtime snapshot for the dashboard.
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

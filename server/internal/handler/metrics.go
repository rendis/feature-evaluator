package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	evalmetrics "github.com/rendis/feature-evaluator/internal/domain/metrics"
)

// MetricsHandler handles metrics dashboard endpoints.
type MetricsHandler struct {
	collector *evalmetrics.Collector
}

// NewMetricsHandler creates a new MetricsHandler.
func NewMetricsHandler(collector *evalmetrics.Collector) *MetricsHandler {
	return &MetricsHandler{collector: collector}
}

func parseDays(c *gin.Context) int {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days < 1 {
		days = 1
	}
	if days > 30 {
		days = 30
	}
	return days
}

func dateRange(days int) []string {
	now := time.Now().UTC()
	dates := make([]string, days)
	for i := range days {
		dates[i] = now.AddDate(0, 0, -i).Format("2006-01-02")
	}
	return dates
}

// Overview returns today's totals and a 7-day trend.
func (h *MetricsHandler) Overview(c *gin.Context) {
	ctx := c.Request.Context()
	days := parseDays(c)
	dates := dateRange(days)

	today := dates[0]
	totalToday := h.collector.DailyCount(ctx, today)
	reasons := h.collector.ReasonBreakdown(ctx, today)
	hits := h.collector.CacheHits(ctx, today)
	misses := h.collector.CacheMisses(ctx, today)

	var cacheHitRatio float64
	if total := hits + misses; total > 0 {
		cacheHitRatio = float64(hits) / float64(total)
	}

	type trendEntry struct {
		Date   string `json:"date"`
		Total  int64  `json:"total"`
		Errors int64  `json:"errors"`
	}

	trend := make([]trendEntry, 0, len(dates))
	for _, d := range dates {
		total := h.collector.DailyCount(ctx, d)
		dr := h.collector.ReasonBreakdown(ctx, d)
		trend = append(trend, trendEntry{
			Date:   d,
			Total:  total,
			Errors: dr["error"],
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"today": gin.H{
			"total":         totalToday,
			"errors":        reasons["error"],
			"cacheHitRatio": cacheHitRatio,
		},
		"trend": trend,
	})
}

// Features returns top features by evaluation count.
func (h *MetricsHandler) Features(c *gin.Context) {
	ctx := c.Request.Context()
	days := parseDays(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 {
		limit = 10
	}
	dates := dateRange(days)

	// Aggregate across days
	agg := make(map[string]float64)
	for _, d := range dates {
		for _, mc := range h.collector.TopFeatures(ctx, d, 0) {
			agg[mc.Member] += mc.Count
		}
	}

	sorted := sortMapDesc(agg, limit)
	type entry struct {
		FeatureKey string `json:"featureKey"`
		Count      int64  `json:"count"`
	}
	result := make([]entry, 0, len(sorted))
	for _, kv := range sorted {
		result = append(result, entry{FeatureKey: kv.key, Count: int64(kv.val)})
	}

	c.JSON(http.StatusOK, result)
}

// Reasons returns reason breakdown aggregated over the date range.
func (h *MetricsHandler) Reasons(c *gin.Context) {
	ctx := c.Request.Context()
	days := parseDays(c)
	dates := dateRange(days)

	agg := make(map[string]int64)
	for _, d := range dates {
		for reason, count := range h.collector.ReasonBreakdown(ctx, d) {
			agg[reason] += count
		}
	}

	c.JSON(http.StatusOK, agg)
}

// Tenants returns top tenants by evaluation count.
func (h *MetricsHandler) Tenants(c *gin.Context) {
	ctx := c.Request.Context()
	days := parseDays(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 {
		limit = 10
	}
	dates := dateRange(days)

	agg := make(map[string]float64)
	for _, d := range dates {
		for _, mc := range h.collector.TopTenants(ctx, d, 0) {
			agg[mc.Member] += mc.Count
		}
	}

	sorted := sortMapDesc(agg, limit)
	type entry struct {
		TenantID string `json:"tenantId"`
		Count    int64  `json:"count"`
	}
	result := make([]entry, 0, len(sorted))
	for _, kv := range sorted {
		result = append(result, entry{TenantID: kv.key, Count: int64(kv.val)})
	}

	c.JSON(http.StatusOK, result)
}

// Environments returns environment breakdown.
func (h *MetricsHandler) Environments(c *gin.Context) {
	ctx := c.Request.Context()
	days := parseDays(c)
	dates := dateRange(days)

	agg := make(map[string]int64)
	for _, d := range dates {
		for env, count := range h.collector.EnvironmentBreakdown(ctx, d) {
			agg[env] += count
		}
	}

	c.JSON(http.StatusOK, agg)
}

// Cache returns cache hit/miss stats.
func (h *MetricsHandler) Cache(c *gin.Context) {
	ctx := c.Request.Context()
	days := parseDays(c)
	dates := dateRange(days)

	var totalHits, totalMisses int64
	for _, d := range dates {
		totalHits += h.collector.CacheHits(ctx, d)
		totalMisses += h.collector.CacheMisses(ctx, d)
	}

	var ratio float64
	if total := totalHits + totalMisses; total > 0 {
		ratio = float64(totalHits) / float64(total)
	}

	c.JSON(http.StatusOK, gin.H{
		"hits":   totalHits,
		"misses": totalMisses,
		"ratio":  ratio,
	})
}

// External returns external call latency percentiles and circuit breaker stats.
func (h *MetricsHandler) External(c *gin.Context) {
	ctx := c.Request.Context()
	days := parseDays(c)
	dates := dateRange(days)

	// Use today's latency samples for percentiles
	p50, p95 := h.collector.ExternalLatencyPercentiles(ctx, dates[0])

	var openEvents, closeEvents int64
	for _, d := range dates {
		o, cl := h.collector.CircuitBreakerCounts(ctx, d)
		openEvents += o
		closeEvents += cl
	}

	c.JSON(http.StatusOK, gin.H{
		"p50Ms":         p50,
		"p95Ms":         p95,
		"cbOpenEvents":  openEvents,
		"cbCloseEvents": closeEvents,
	})
}

type kv struct {
	key string
	val float64
}

func sortMapDesc(m map[string]float64, limit int) []kv {
	items := make([]kv, 0, len(m))
	for k, v := range m {
		items = append(items, kv{k, v})
	}
	// Sort descending by value
	for i := range items {
		for j := i + 1; j < len(items); j++ {
			if items[j].val > items[i].val {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return items
}

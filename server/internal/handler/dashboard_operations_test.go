package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	redisclient "github.com/rendis/feature-evaluator/internal/storage/redis"
)

type fakePostgresPinger struct {
	delay time.Duration
	err   error
}

func (f fakePostgresPinger) Ping(ctx context.Context) error {
	return runPing(ctx, f.delay, f.err)
}

type fakeRedisChecker struct {
	delay time.Duration
	err   error
	state redisclient.CircuitState
}

func (f fakeRedisChecker) Ping(ctx context.Context) error {
	return runPing(ctx, f.delay, f.err)
}

func (f fakeRedisChecker) CircuitState() redisclient.CircuitState {
	return f.state
}

type fakeDashboardMetrics struct {
	dailyCount    int64
	reasons       map[string]int64
	cacheHits     int64
	cacheMisses   int64
	rateLimitHits int64
	externalP50   float64
	externalP95   float64
	cbOpen        int64
	cbClose       int64
}

func (f fakeDashboardMetrics) DailyCount(_ context.Context, _ string) int64 {
	return f.dailyCount
}

func (f fakeDashboardMetrics) ReasonBreakdown(_ context.Context, _ string) map[string]int64 {
	return f.reasons
}

func (f fakeDashboardMetrics) CacheHits(_ context.Context, _ string) int64 {
	return f.cacheHits
}

func (f fakeDashboardMetrics) CacheMisses(_ context.Context, _ string) int64 {
	return f.cacheMisses
}

func (f fakeDashboardMetrics) RateLimitRejects(_ context.Context, _ string) int64 {
	return f.rateLimitHits
}

func (f fakeDashboardMetrics) ExternalLatencyPercentiles(_ context.Context, _ string) (float64, float64) {
	return f.externalP50, f.externalP95
}

func (f fakeDashboardMetrics) CircuitBreakerCounts(_ context.Context, _ string) (int64, int64) {
	return f.cbOpen, f.cbClose
}

type operationsResponse struct {
	CheckedAt     time.Time `json:"checkedAt"`
	OverallStatus string    `json:"overallStatus"`
	Services      struct {
		PostgreSQL struct {
			Status    string `json:"status"`
			LatencyMs *int64 `json:"latencyMs"`
		} `json:"postgresql"`
		Redis struct {
			Status      string     `json:"status"`
			LatencyMs   *int64     `json:"latencyMs"`
			CircuitOpen bool       `json:"circuitOpen"`
			OpenUntil   *time.Time `json:"openUntil"`
		} `json:"redis"`
	} `json:"services"`
	Metrics struct {
		EvaluationsToday          int64   `json:"evaluationsToday"`
		ErrorsToday               int64   `json:"errorsToday"`
		CacheHitRatio             float64 `json:"cacheHitRatio"`
		RateLimitRejectsToday     int64   `json:"rateLimitRejectsToday"`
		ExternalP50Ms             float64 `json:"externalP50Ms"`
		ExternalP95Ms             float64 `json:"externalP95Ms"`
		CircuitBreakerOpenEvents  int64   `json:"circuitBreakerOpenEvents"`
		CircuitBreakerCloseEvents int64   `json:"circuitBreakerCloseEvents"`
	} `json:"metrics"`
}

type readinessResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

func TestDashboardOperationsHealthy(t *testing.T) {
	t.Parallel()

	handler := NewDashboardHandler(
		nil,
		nil,
		nil,
		fakeDashboardMetrics{
			dailyCount:    120,
			reasons:       map[string]int64{"error": 3},
			cacheHits:     90,
			cacheMisses:   30,
			rateLimitHits: 4,
			externalP50:   18,
			externalP95:   42,
			cbOpen:        1,
			cbClose:       1,
		},
		fakePostgresPinger{},
		fakeRedisChecker{},
	)

	recorder := performRequest(t, handler.Operations)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	var response operationsResponse
	decodeResponse(t, recorder, &response)

	if response.OverallStatus != "healthy" {
		t.Fatalf("overallStatus = %q, want healthy", response.OverallStatus)
	}
	if response.CheckedAt.IsZero() {
		t.Fatal("expected checkedAt to be populated")
	}
	if response.Services.PostgreSQL.Status != "healthy" || response.Services.PostgreSQL.LatencyMs == nil {
		t.Fatalf("unexpected postgres status: %+v", response.Services.PostgreSQL)
	}
	if response.Services.Redis.Status != "healthy" || response.Services.Redis.LatencyMs == nil {
		t.Fatalf("unexpected redis status: %+v", response.Services.Redis)
	}
	if response.Metrics.EvaluationsToday != 120 {
		t.Fatalf("evaluationsToday = %d, want 120", response.Metrics.EvaluationsToday)
	}
	if response.Metrics.ErrorsToday != 3 {
		t.Fatalf("errorsToday = %d, want 3", response.Metrics.ErrorsToday)
	}
	if response.Metrics.CacheHitRatio != 0.75 {
		t.Fatalf("cacheHitRatio = %v, want 0.75", response.Metrics.CacheHitRatio)
	}
	if response.Metrics.RateLimitRejectsToday != 4 {
		t.Fatalf("rateLimitRejectsToday = %d, want 4", response.Metrics.RateLimitRejectsToday)
	}
	if response.Metrics.ExternalP50Ms != 18 || response.Metrics.ExternalP95Ms != 42 {
		t.Fatalf("unexpected external percentiles: %+v", response.Metrics)
	}
}

func TestDashboardOperationsDegradedWhenRedisCircuitIsOpen(t *testing.T) {
	t.Parallel()

	openUntil := time.Now().Add(30 * time.Second).UTC()
	handler := NewDashboardHandler(
		nil,
		nil,
		nil,
		fakeDashboardMetrics{dailyCount: 10},
		fakePostgresPinger{},
		fakeRedisChecker{
			err: errors.New("redis unavailable"),
			state: redisclient.CircuitState{
				Open:      true,
				OpenUntil: openUntil,
			},
		},
	)

	recorder := performRequest(t, handler.Operations)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	var response operationsResponse
	decodeResponse(t, recorder, &response)

	if response.OverallStatus != "degraded" {
		t.Fatalf("overallStatus = %q, want degraded", response.OverallStatus)
	}
	if response.Services.Redis.Status != "degraded" {
		t.Fatalf("redis.status = %q, want degraded", response.Services.Redis.Status)
	}
	if !response.Services.Redis.CircuitOpen {
		t.Fatal("expected redis circuitOpen to be true")
	}
	if response.Services.Redis.OpenUntil == nil || !response.Services.Redis.OpenUntil.Equal(openUntil) {
		t.Fatalf("openUntil = %v, want %v", response.Services.Redis.OpenUntil, openUntil)
	}
	if response.Metrics.EvaluationsToday != 10 {
		t.Fatalf("evaluationsToday = %d, want 10", response.Metrics.EvaluationsToday)
	}
}

func TestDashboardOperationsUnhealthyWhenPostgresFails(t *testing.T) {
	t.Parallel()

	handler := NewDashboardHandler(
		nil,
		nil,
		nil,
		fakeDashboardMetrics{},
		fakePostgresPinger{err: errors.New("postgres down")},
		fakeRedisChecker{},
	)

	recorder := performRequest(t, handler.Operations)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	var response operationsResponse
	decodeResponse(t, recorder, &response)

	if response.OverallStatus != "unhealthy" {
		t.Fatalf("overallStatus = %q, want unhealthy", response.OverallStatus)
	}
	if response.Services.PostgreSQL.Status != "unhealthy" {
		t.Fatalf("postgres.status = %q, want unhealthy", response.Services.PostgreSQL.Status)
	}
	if response.Services.PostgreSQL.LatencyMs != nil {
		t.Fatalf("postgres.latencyMs = %v, want nil", response.Services.PostgreSQL.LatencyMs)
	}
}

func TestDashboardOperationsWithoutMetricsData(t *testing.T) {
	t.Parallel()

	handler := NewDashboardHandler(
		nil,
		nil,
		nil,
		fakeDashboardMetrics{},
		fakePostgresPinger{},
		fakeRedisChecker{},
	)

	recorder := performRequest(t, handler.Operations)

	var response operationsResponse
	decodeResponse(t, recorder, &response)

	if response.Metrics.CacheHitRatio != 0 {
		t.Fatalf("cacheHitRatio = %v, want 0", response.Metrics.CacheHitRatio)
	}
	if response.Metrics.ExternalP50Ms != 0 || response.Metrics.ExternalP95Ms != 0 {
		t.Fatalf("expected zero latencies, got p50=%v p95=%v", response.Metrics.ExternalP50Ms, response.Metrics.ExternalP95Ms)
	}
}

func TestHealthReadinessRedisDegradedKeepsReadyShape(t *testing.T) {
	t.Parallel()

	handler := NewHealthHandler(
		fakePostgresPinger{},
		fakeRedisChecker{
			err: errors.New("redis unavailable"),
			state: redisclient.CircuitState{
				Open:      true,
				OpenUntil: time.Now().Add(30 * time.Second).UTC(),
			},
		},
	)

	recorder := performRequest(t, handler.Readiness)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	var response readinessResponse
	decodeResponse(t, recorder, &response)

	if response.Status != "ready" {
		t.Fatalf("status = %q, want ready", response.Status)
	}
	if response.Checks["postgresql"] != "healthy" {
		t.Fatalf("checks.postgresql = %q, want healthy", response.Checks["postgresql"])
	}
	if response.Checks["redis"] != "unhealthy (degraded)" {
		t.Fatalf("checks.redis = %q, want unhealthy (degraded)", response.Checks["redis"])
	}
}

func TestHealthReadinessPostgresFailureReturnsNotReady(t *testing.T) {
	t.Parallel()

	handler := NewHealthHandler(
		fakePostgresPinger{err: errors.New("postgres down")},
		fakeRedisChecker{},
	)

	recorder := performRequest(t, handler.Readiness)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}

	var response readinessResponse
	decodeResponse(t, recorder, &response)

	if response.Status != "not ready" {
		t.Fatalf("status = %q, want not ready", response.Status)
	}
	if response.Checks["postgresql"] != "unhealthy" {
		t.Fatalf("checks.postgresql = %q, want unhealthy", response.Checks["postgresql"])
	}
}

func performRequest(t *testing.T, handlerFunc gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	handlerFunc(ctx)
	return recorder
}

func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder, target any) {
	t.Helper()

	if err := json.Unmarshal(recorder.Body.Bytes(), target); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
}

func runPing(ctx context.Context, delay time.Duration, err error) error {
	if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}

	return err
}

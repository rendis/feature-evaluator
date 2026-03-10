package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
)

func TestRateLimiterSkipsRedisDuringCooldown(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	now := time.Date(2026, time.March, 8, 12, 0, 0, 0, time.UTC)
	allowCalls := 0

	rl := &RateLimiter{
		now:      func() time.Time { return now },
		cooldown: 30 * time.Second,
	}
	rl.allow = func(_ *gin.Context, _ string, _ int) (*redis_rate.Result, error) {
		allowCalls++
		if allowCalls == 1 {
			return nil, errors.New("redis unavailable")
		}
		return &redis_rate.Result{Allowed: 1}, nil
	}

	router := gin.New()
	router.Use(rl.Limit("fe:rl:test:", 10, func(_ *gin.Context) string { return "tester" }))
	router.GET("/limited", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("first response status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if allowCalls != 1 {
		t.Fatalf("allow calls after first request = %d, want 1", allowCalls)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/limited", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("second response status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if allowCalls != 1 {
		t.Fatalf("allow calls during cooldown = %d, want 1", allowCalls)
	}

	now = now.Add(31 * time.Second)

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/limited", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("third response status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if allowCalls != 2 {
		t.Fatalf("allow calls after cooldown = %d, want 2", allowCalls)
	}
}

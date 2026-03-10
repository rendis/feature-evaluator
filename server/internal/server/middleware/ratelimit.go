package middleware

import (
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	goredis "github.com/redis/go-redis/v9"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

const defaultRateLimiterCooldown = 30 * time.Second

// RateLimitRejectFunc is called when a request is rate-limited.
type RateLimitRejectFunc func()

// RateLimiter provides rate limiting middleware using redis_rate GCRA.
type RateLimiter struct {
	limiter  *redis_rate.Limiter
	onReject RateLimitRejectFunc
	allow    func(ctx *gin.Context, key string, perSecond int) (*redis_rate.Result, error)
	now      func() time.Time
	cooldown time.Duration

	mu        sync.RWMutex
	skipUntil time.Time
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter(rdb *goredis.Client) *RateLimiter {
	rl := &RateLimiter{
		limiter:  redis_rate.NewLimiter(rdb),
		now:      time.Now,
		cooldown: defaultRateLimiterCooldown,
	}
	rl.allow = func(c *gin.Context, key string, perSecond int) (*redis_rate.Result, error) {
		return rl.limiter.Allow(c.Request.Context(), key, redis_rate.PerSecond(perSecond))
	}
	return rl
}

// SetOnReject sets the callback invoked on each rate limit rejection.
func (rl *RateLimiter) SetOnReject(fn RateLimitRejectFunc) {
	rl.onReject = fn
}

// Limit returns a middleware that rate-limits requests.
// keyPrefix is used to namespace the rate limit key (e.g. "fe:rl:eval:" or "fe:rl:admin:").
// perSecond is the allowed rate per second.
// identifierFn extracts the identifier from the request (e.g. API key or user email).
func (rl *RateLimiter) Limit(keyPrefix string, perSecond int, identifierFn func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rl.shouldSkip(rl.now()) {
			c.Next()
			return
		}

		identifier := identifierFn(c)
		if identifier == "" {
			identifier = c.ClientIP()
		}

		key := keyPrefix + identifier
		result, err := rl.allow(c, key, perSecond)
		if err != nil {
			rl.openCooldown(err, key)
			c.Next()
			return
		}
		rl.clearCooldown()

		if result.Allowed == 0 {
			if rl.onReject != nil {
				rl.onReject()
			}

			retryAfter := result.RetryAfter.Seconds()
			c.Header("Retry-After", strconv.FormatFloat(retryAfter, 'f', 0, 64))

			apiErr := apierror.NewTooManyRequests("rate limit exceeded", "error.rateLimitExceeded")
			apiErr.RequestID = GetRequestID(c)
			c.AbortWithStatusJSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
			return
		}

		c.Next()
	}
}

func (rl *RateLimiter) shouldSkip(now time.Time) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return now.Before(rl.skipUntil)
}

func (rl *RateLimiter) openCooldown(err error, key string) {
	now := rl.now()
	until := now.Add(rl.cooldown)

	rl.mu.Lock()
	shouldLog := !now.Before(rl.skipUntil)
	rl.skipUntil = until
	rl.mu.Unlock()

	if shouldLog {
		slog.Warn(
			"rate limiter redis unavailable, bypassing temporarily",
			"error", err,
			"key", key,
			"cooldown", rl.cooldown.String(),
		)
	}
}

func (rl *RateLimiter) clearCooldown() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.skipUntil = time.Time{}
}

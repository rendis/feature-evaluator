package redis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps the Redis client with circuit breaker for fail-open behavior.
type Client struct {
	rdb *redis.Client

	mu          sync.RWMutex
	circuitOpen bool
	errorCount  int
	lastError   time.Time
	openUntil   time.Time

	// OnCacheAccess is an optional hook called on cache Get for feature keys.
	// hit is true when a value was found.
	OnCacheAccess func(hit bool)
}

// CircuitState exposes the current circuit breaker status.
type CircuitState struct {
	Open      bool
	OpenUntil time.Time
}

const (
	maxErrors      = 3
	cooldownPeriod = 30 * time.Second
	errorWindow    = 10 * time.Second
)

var errCircuitBreakerOpen = errors.New("redis circuit breaker open")

// NewClient creates a new Redis client and verifies the connection.
func NewClient(ctx context.Context, addr, password string) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     20,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Warn("Redis unavailable, operating in degraded mode", "error", err)
		return &Client{rdb: rdb, circuitOpen: true, openUntil: time.Now().Add(cooldownPeriod)}, nil
	}

	slog.Info("connected to Redis", "addr", addr)
	return &Client{rdb: rdb}, nil
}

// Available returns true if Redis circuit is closed and available.
func (c *Client) Available() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.circuitOpen {
		return true
	}
	return time.Now().After(c.openUntil)
}

// recordError tracks errors and opens circuit if threshold reached.
func (c *Client) recordError() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if now.Sub(c.lastError) > errorWindow {
		c.errorCount = 0
	}
	c.errorCount++
	c.lastError = now

	if c.errorCount >= maxErrors {
		c.circuitOpen = true
		c.openUntil = now.Add(cooldownPeriod)
		slog.Warn("Redis circuit breaker opened", "cooldown", cooldownPeriod)
	}
}

// recordSuccess resets error tracking.
func (c *Client) recordSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errorCount = 0
	c.circuitOpen = false
}

// tryRecover attempts to close the circuit if the cooldown has passed.
func (c *Client) tryRecover(ctx context.Context) bool {
	c.mu.Lock()
	if !c.circuitOpen || time.Now().Before(c.openUntil) {
		c.mu.Unlock()
		return !c.circuitOpen
	}
	c.mu.Unlock()

	if err := c.rdb.Ping(ctx).Err(); err != nil {
		c.mu.Lock()
		c.openUntil = time.Now().Add(cooldownPeriod)
		c.mu.Unlock()
		return false
	}

	c.recordSuccess()
	slog.Info("Redis circuit breaker recovered")
	return true
}

// Get retrieves a value from Redis. Returns empty string if unavailable.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if !c.Available() {
		if !c.tryRecover(ctx) {
			return "", nil
		}
	}

	val, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		if c.OnCacheAccess != nil && strings.HasPrefix(key, featureKeyPrefix) {
			c.OnCacheAccess(false)
		}
		return "", nil
	}
	if err != nil {
		c.recordError()
		return "", nil // fail-open
	}
	c.recordSuccess()
	if c.OnCacheAccess != nil && strings.HasPrefix(key, featureKeyPrefix) {
		c.OnCacheAccess(true)
	}
	return val, nil
}

// Set stores a value in Redis with TTL. Silently fails if unavailable.
func (c *Client) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if !c.Available() {
		if !c.tryRecover(ctx) {
			return nil
		}
	}

	if err := c.rdb.Set(ctx, key, value, ttl).Err(); err != nil {
		c.recordError()
		return nil // fail-open
	}
	c.recordSuccess()
	return nil
}

// Del deletes one or more keys. Silently fails if unavailable.
func (c *Client) Del(ctx context.Context, keys ...string) error {
	if !c.Available() {
		if !c.tryRecover(ctx) {
			return nil
		}
	}

	if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
		c.recordError()
		return nil // fail-open
	}
	c.recordSuccess()
	return nil
}

// DelPattern deletes keys matching a pattern. Silently fails if unavailable.
func (c *Client) DelPattern(ctx context.Context, pattern string) error {
	if !c.Available() {
		if !c.tryRecover(ctx) {
			return nil
		}
	}

	iter := c.rdb.Scan(ctx, 0, pattern, 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		c.recordError()
		return nil
	}

	if len(keys) > 0 {
		if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
			c.recordError()
			return nil
		}
	}

	c.recordSuccess()
	return nil
}

// Underlying returns the raw redis.Client for libraries that need it (e.g. redis_rate).
func (c *Client) Underlying() *redis.Client {
	return c.rdb
}

// CircuitState returns the current circuit breaker state.
func (c *Client) CircuitState() CircuitState {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CircuitState{
		Open:      c.circuitOpen,
		OpenUntil: c.openUntil,
	}
}

// Ping checks if Redis is reachable.
func (c *Client) Ping(ctx context.Context) error {
	if c.CircuitState().Open {
		if !c.tryRecover(ctx) {
			return errCircuitBreakerOpen
		}
		return nil
	}

	if err := c.rdb.Ping(ctx).Err(); err != nil {
		c.recordError()
		return err
	}
	c.recordSuccess()
	return nil
}

// Close disconnects the Redis client.
func (c *Client) Close() error {
	slog.Info("disconnecting from Redis")
	return c.rdb.Close()
}

// Keys for the cache.
const (
	// FeatureKey returns the Redis key for a feature config cache.
	featureKeyPrefix = "fe:feature:"
	// MemberKey returns the Redis key for a member cache.
	memberKeyPrefix = "fe:member:"
	// SegmentMemberKey prefix for segment membership cache.
	segmentMemberPrefix = "fe:seg:"
	// SegmentRecordKey prefix for segment record cache.
	segmentRecordPrefix = "fe:segrec:"
	// ExternalCallKey prefix for external call result cache.
	externalCallPrefix = "fe:ext:"
	// AuthProfileValidationKey prefix for auth profile validation cache.
	authProfileValidationPrefix = "fe:authv:"
	// AuthProfileTokenKey prefix for auth profile token/header cache.
	//nolint:gosec // Redis key prefix, not a token or credential.
	authProfileTokenPrefix = "fe:authtok:"
)

// FeatureKey returns the Redis key for caching a feature.
func FeatureKey(key string) string {
	return fmt.Sprintf("%s%s", featureKeyPrefix, key)
}

// FeatureWorkspaceKey returns the Redis key for caching a feature within a workspace.
func FeatureWorkspaceKey(workspaceKey, key string) string {
	return fmt.Sprintf("%s%s:%s", featureKeyPrefix, workspaceKey, key)
}

// MemberKey returns the Redis key for caching a member.
func MemberKey(email string) string {
	return fmt.Sprintf("%s%s", memberKeyPrefix, email)
}

// SegmentMemberKey returns the Redis key for caching segment membership.
func SegmentMemberKey(segmentKey, userID, tenantID string) string {
	return fmt.Sprintf("%s%s:%s:%s", segmentMemberPrefix, segmentKey, userID, tenantID)
}

// SegmentRecordKey returns the Redis key for caching a segment record lookup.
func SegmentRecordKey(segmentKey, datasetVersion, recordKey string) string {
	return fmt.Sprintf("%s%s:%s:%s", segmentRecordPrefix, segmentKey, datasetVersion, recordKey)
}

// ExternalCallKey returns the Redis key for caching external call results.
func ExternalCallKey(hash string) string {
	return fmt.Sprintf("%s%s", externalCallPrefix, hash)
}

// AuthProfileValidationKey returns the Redis key for auth profile validation cache.
func AuthProfileValidationKey(workspaceKey, profileKey string, version int, fingerprint string) string {
	return fmt.Sprintf("%s%s:%s:%d:%s", authProfileValidationPrefix, workspaceKey, profileKey, version, fingerprint)
}

// AuthProfileTokenKey returns the Redis key for auth profile token/header cache.
func AuthProfileTokenKey(workspaceKey, profileKey string, version int) string {
	return fmt.Sprintf("%s%s:%s:%d", authProfileTokenPrefix, workspaceKey, profileKey, version)
}

// SegmentPattern returns the pattern for deleting all segment membership cache entries.
func SegmentPattern(segmentKey string) string {
	return fmt.Sprintf("%s%s:*", segmentMemberPrefix, segmentKey)
}

// SegmentRecordPattern returns the pattern for deleting all segment record cache entries.
func SegmentRecordPattern(segmentKey string) string {
	return fmt.Sprintf("%s%s:*", segmentRecordPrefix, segmentKey)
}

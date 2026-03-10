package evalmetrics

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	redisclient "github.com/rendis/feature-evaluator/internal/storage/redis"
)

const (
	keyPrefix     = "fe:m:"
	metricTTL     = 7 * 24 * time.Hour // 604800s
	chanCapacity  = 1000
	batchSize     = 100
	flushInterval = 1 * time.Second
	latencyCap    = 1000 // max latency samples per day
	warnThrottle  = 1 * time.Minute
)

// Event represents a single evaluation metric event.
type Event struct {
	FeatureKey  string
	Reason      string
	TenantID    string
	Environment string
	HasError    bool
	CacheHit    bool
	CacheMiss   bool
	SegmentKey  string
}

// latencyEvent records external call latency.
type latencyEvent struct {
	ms float64
}

// cbEvent records circuit breaker state changes.
type cbEvent struct {
	open bool
}

// rlEvent records rate limit rejections.
type rlEvent struct{}

// Collector collects evaluation metrics and writes them to Redis in batches.
type Collector struct {
	rdb *redisclient.Client

	events    chan Event
	latencies chan latencyEvent
	cbEvents  chan cbEvent
	rlEvents  chan rlEvent

	stop chan struct{}
	done chan struct{}

	mu           sync.Mutex
	lastWarnTime time.Time
	ttlSet       map[string]bool // tracks keys that already have TTL set today
	ttlDay       string          // date string for ttlSet
}

// NewCollector creates a new metrics Collector.
func NewCollector(rdb *redisclient.Client) *Collector {
	return &Collector{
		rdb:       rdb,
		events:    make(chan Event, chanCapacity),
		latencies: make(chan latencyEvent, chanCapacity),
		cbEvents:  make(chan cbEvent, chanCapacity),
		rlEvents:  make(chan rlEvent, chanCapacity),
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
		ttlSet:    make(map[string]bool),
	}
}

// Start begins the background flush goroutine.
func (c *Collector) Start() {
	go c.loop()
}

// Stop signals the background goroutine to drain and exit.
func (c *Collector) Stop() {
	close(c.stop)
	<-c.done
}

// Record enqueues an evaluation event. Non-blocking; drops if buffer full.
func (c *Collector) Record(e Event) {
	select {
	case c.events <- e:
	default:
		c.warnDrop()
	}
}

// RecordCacheAccess records a cache hit or miss.
func (c *Collector) RecordCacheAccess(hit bool) {
	e := Event{CacheHit: hit, CacheMiss: !hit}
	select {
	case c.events <- e:
	default:
		c.warnDrop()
	}
}

// RecordRateLimitReject records a rate limit rejection.
func (c *Collector) RecordRateLimitReject() {
	select {
	case c.rlEvents <- rlEvent{}:
	default:
		c.warnDrop()
	}
}

// RecordExternalLatency records an external call latency sample.
func (c *Collector) RecordExternalLatency(d time.Duration) {
	select {
	case c.latencies <- latencyEvent{ms: float64(d.Milliseconds())}:
	default:
		c.warnDrop()
	}
}

// RecordCircuitBreakerEvent records a circuit breaker open or close.
func (c *Collector) RecordCircuitBreakerEvent(opened bool) {
	select {
	case c.cbEvents <- cbEvent{open: opened}:
	default:
		c.warnDrop()
	}
}

// RecordSegmentLookup records a segment lookup.
func (c *Collector) RecordSegmentLookup(segmentKey string) {
	select {
	case c.events <- Event{SegmentKey: segmentKey}:
	default:
		c.warnDrop()
	}
}

func (c *Collector) warnDrop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.lastWarnTime) > warnThrottle {
		slog.Warn("metrics collector: buffer full, dropping event")
		c.lastWarnTime = time.Now()
	}
}

//nolint:cyclop // The event loop intentionally centralizes channel draining and timed flushing.
func (c *Collector) loop() {
	defer close(c.done)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	var (
		evalBuf []Event
		latBuf  []latencyEvent
		cbBuf   []cbEvent
		rlCount int
	)

	flush := func() {
		if len(evalBuf) == 0 && len(latBuf) == 0 && len(cbBuf) == 0 && rlCount == 0 {
			return
		}
		c.flush(evalBuf, latBuf, cbBuf, rlCount)
		evalBuf = evalBuf[:0]
		latBuf = latBuf[:0]
		cbBuf = cbBuf[:0]
		rlCount = 0
	}

	for {
		select {
		case e := <-c.events:
			evalBuf = append(evalBuf, e)
			if len(evalBuf) >= batchSize {
				flush()
			}
		case l := <-c.latencies:
			latBuf = append(latBuf, l)
		case cb := <-c.cbEvents:
			cbBuf = append(cbBuf, cb)
		case <-c.rlEvents:
			rlCount++
		case <-ticker.C:
			flush()
		case <-c.stop:
			// Drain remaining
		drainLoop:
			for {
				select {
				case e := <-c.events:
					evalBuf = append(evalBuf, e)
				case l := <-c.latencies:
					latBuf = append(latBuf, l)
				case cb := <-c.cbEvents:
					cbBuf = append(cbBuf, cb)
				case <-c.rlEvents:
					rlCount++
				default:
					break drainLoop
				}
			}
			flush()
			return
		}
	}
}

//nolint:cyclop,gocognit // This is a single Redis batch writer that keeps metric aggregation in one place.
func (c *Collector) flush(events []Event, latencies []latencyEvent, cbEvents []cbEvent, rlCount int) {
	if !c.rdb.Available() {
		c.warnDrop()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	day := time.Now().UTC().Format("2006-01-02")
	c.resetTTLSetIfNewDay(day)

	pipe := c.rdb.Underlying().Pipeline()

	for _, e := range events {
		// Pure cache-only events (from RecordCacheAccess)
		if e.CacheHit && e.FeatureKey == "" {
			c.incrWithTTL(pipe, day, "cache:hit", 1)
			continue
		}
		if e.CacheMiss && e.FeatureKey == "" {
			c.incrWithTTL(pipe, day, "cache:miss", 1)
			continue
		}

		// Segment-only events (from RecordSegmentLookup)
		if e.SegmentKey != "" && e.FeatureKey == "" {
			c.zincrWithTTL(pipe, day, "seg:lookup", e.SegmentKey, 1)
			continue
		}

		// Full evaluation event
		if e.FeatureKey != "" {
			// Tier 1
			c.incrWithTTL(pipe, day, "eval:daily", 1)
			c.zincrWithTTL(pipe, day, "eval:feature", e.FeatureKey, 1)
			if e.Reason != "" {
				c.hincrWithTTL(pipe, day, "eval:reason", e.Reason, 1)
			}

			// Tier 2
			if e.TenantID != "" {
				c.zincrWithTTL(pipe, day, "eval:tenant", e.TenantID, 1)
			}
			if e.Environment != "" {
				c.hincrWithTTL(pipe, day, "eval:env", e.Environment, 1)
			}
		}
	}

	// Latency samples
	if len(latencies) > 0 {
		latKey := c.key(day, "ext:latency")
		for _, l := range latencies {
			pipe.RPush(ctx, latKey, l.ms)
		}
		pipe.LTrim(ctx, latKey, -int64(latencyCap), -1)
		c.ensureTTL(pipe, day, "ext:latency")
	}

	// Circuit breaker events
	for _, cb := range cbEvents {
		field := "close"
		if cb.open {
			field = "open"
		}
		c.hincrWithTTL(pipe, day, "ext:cb", field, 1)
	}

	// Rate limit rejections
	if rlCount > 0 {
		c.incrWithTTL(pipe, day, "rl:reject", int64(rlCount))
	}

	if _, err := pipe.Exec(ctx); err != nil {
		c.mu.Lock()
		if time.Since(c.lastWarnTime) > warnThrottle {
			slog.Warn("metrics collector: redis pipeline error", "error", err)
			c.lastWarnTime = time.Now()
		}
		c.mu.Unlock()
	}
}

func (c *Collector) key(day, suffix string) string {
	return fmt.Sprintf("%s%s:{%s}", keyPrefix, suffix, day)
}

func (c *Collector) incrWithTTL(pipe redis.Pipeliner, day, suffix string, val int64) {
	k := c.key(day, suffix)
	pipe.IncrBy(context.Background(), k, val)
	c.ensureTTL(pipe, day, suffix)
}

func (c *Collector) zincrWithTTL(pipe redis.Pipeliner, day, suffix, member string, val float64) {
	k := c.key(day, suffix)
	pipe.ZIncrBy(context.Background(), k, val, member)
	c.ensureTTL(pipe, day, suffix)
}

func (c *Collector) hincrWithTTL(pipe redis.Pipeliner, day, suffix, field string, val int64) {
	k := c.key(day, suffix)
	pipe.HIncrBy(context.Background(), k, field, val)
	c.ensureTTL(pipe, day, suffix)
}

func (c *Collector) ensureTTL(pipe redis.Pipeliner, day, suffix string) {
	ttlKey := day + ":" + suffix
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ttlSet[ttlKey] {
		return
	}
	c.ttlSet[ttlKey] = true
	pipe.Expire(context.Background(), c.key(day, suffix), metricTTL)
}

func (c *Collector) resetTTLSetIfNewDay(day string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ttlDay != day {
		c.ttlSet = make(map[string]bool)
		c.ttlDay = day
	}
}

// --- Read helpers (used by handler) ---

// DailyCount returns the total eval count for a date.
func (c *Collector) DailyCount(ctx context.Context, day string) int64 {
	val, err := c.rdb.Underlying().Get(ctx, c.key(day, "eval:daily")).Int64()
	if err != nil {
		return 0
	}
	return val
}

// ReasonBreakdown returns the reason counts for a date.
func (c *Collector) ReasonBreakdown(ctx context.Context, day string) map[string]int64 {
	return c.hgetAllInt(ctx, c.key(day, "eval:reason"))
}

// TopFeatures returns the top N features by eval count for a date.
func (c *Collector) TopFeatures(ctx context.Context, day string, limit int) []MemberCount {
	return c.zrevrangeWithScores(ctx, c.key(day, "eval:feature"), limit)
}

// TopTenants returns the top N tenants by eval count for a date.
func (c *Collector) TopTenants(ctx context.Context, day string, limit int) []MemberCount {
	return c.zrevrangeWithScores(ctx, c.key(day, "eval:tenant"), limit)
}

// EnvironmentBreakdown returns environment counts for a date.
func (c *Collector) EnvironmentBreakdown(ctx context.Context, day string) map[string]int64 {
	return c.hgetAllInt(ctx, c.key(day, "eval:env"))
}

// SegmentLookupTop returns the top N segments by lookup frequency for a date.
func (c *Collector) SegmentLookupTop(ctx context.Context, day string, limit int) []MemberCount {
	return c.zrevrangeWithScores(ctx, c.key(day, "seg:lookup"), limit)
}

// CacheHits returns cache hits for a date.
func (c *Collector) CacheHits(ctx context.Context, day string) int64 {
	val, err := c.rdb.Underlying().Get(ctx, c.key(day, "cache:hit")).Int64()
	if err != nil {
		return 0
	}
	return val
}

// CacheMisses returns cache misses for a date.
func (c *Collector) CacheMisses(ctx context.Context, day string) int64 {
	val, err := c.rdb.Underlying().Get(ctx, c.key(day, "cache:miss")).Int64()
	if err != nil {
		return 0
	}
	return val
}

// RateLimitRejects returns rate limit rejections for a date.
func (c *Collector) RateLimitRejects(ctx context.Context, day string) int64 {
	val, err := c.rdb.Underlying().Get(ctx, c.key(day, "rl:reject")).Int64()
	if err != nil {
		return 0
	}
	return val
}

// ExternalLatencyPercentiles returns p50 and p95 latency (ms) for a date.
func (c *Collector) ExternalLatencyPercentiles(ctx context.Context, day string) (p50, p95 float64) {
	vals, err := c.rdb.Underlying().LRange(ctx, c.key(day, "ext:latency"), 0, -1).Result()
	if err != nil || len(vals) == 0 {
		return 0, 0
	}
	samples := make([]float64, 0, len(vals))
	for _, v := range vals {
		f, err := strconv.ParseFloat(v, 64)
		if err == nil {
			samples = append(samples, f)
		}
	}
	if len(samples) == 0 {
		return 0, 0
	}
	sort.Float64s(samples)
	p50 = percentile(samples, 50)
	p95 = percentile(samples, 95)
	return p50, p95
}

// CircuitBreakerCounts returns open/close event counts for a date.
func (c *Collector) CircuitBreakerCounts(ctx context.Context, day string) (openCount, closeCount int64) {
	m := c.hgetAllInt(ctx, c.key(day, "ext:cb"))
	return m["open"], m["close"]
}

// MemberCount holds a sorted set member and its score.
type MemberCount struct {
	Member string  `json:"member"`
	Count  float64 `json:"count"`
}

func (c *Collector) zrevrangeWithScores(ctx context.Context, key string, limit int) []MemberCount {
	results, err := c.rdb.Underlying().ZRevRangeWithScores(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		return nil
	}
	out := make([]MemberCount, 0, len(results))
	for _, z := range results {
		out = append(out, MemberCount{
			Member: z.Member.(string),
			Count:  z.Score,
		})
	}
	return out
}

func (c *Collector) hgetAllInt(ctx context.Context, key string) map[string]int64 {
	m, err := c.rdb.Underlying().HGetAll(ctx, key).Result()
	if err != nil {
		return nil
	}
	out := make(map[string]int64, len(m))
	for k, v := range m {
		n, _ := strconv.ParseInt(v, 10, 64)
		out[k] = n
	}
	return out
}

func percentile(sorted []float64, pct float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := (pct / 100) * float64(len(sorted)-1)
	lower := int(math.Floor(idx))
	upper := int(math.Ceil(idx))
	if lower == upper || upper >= len(sorted) {
		return sorted[lower]
	}
	frac := idx - float64(lower)
	return sorted[lower]*(1-frac) + sorted[upper]*frac
}

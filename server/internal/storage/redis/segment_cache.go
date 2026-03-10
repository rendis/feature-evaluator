package redis

import (
	"context"
	"time"
)

const segmentMemberTTL = 300 * time.Second

// SegmentCache implements segment.Cache using the Redis client.
type SegmentCache struct {
	client *Client
}

// NewSegmentCache creates a new segment membership cache backed by Redis.
func NewSegmentCache(client *Client) *SegmentCache {
	return &SegmentCache{client: client}
}

// GetMembership returns the cached membership result.
// Returns (isMember, found). On Redis errors it returns (false, false) to fail-open.
func (sc *SegmentCache) GetMembership(ctx context.Context, segmentKey, userID, tenantID string) (bool, bool) {
	val, _ := sc.client.Get(ctx, SegmentMemberKey(segmentKey, userID, tenantID))
	if val == "" {
		return false, false
	}
	return val == "1", true
}

// SetMembership caches the membership result with a fixed TTL.
func (sc *SegmentCache) SetMembership(ctx context.Context, segmentKey, userID, tenantID string, isMember bool) {
	val := "0"
	if isMember {
		val = "1"
	}
	_ = sc.client.Set(ctx, SegmentMemberKey(segmentKey, userID, tenantID), val, segmentMemberTTL)
}

// InvalidateSegment removes all cached membership entries for a segment.
func (sc *SegmentCache) InvalidateSegment(ctx context.Context, segmentKey string) {
	_ = sc.client.DelPattern(ctx, SegmentPattern(segmentKey))
}

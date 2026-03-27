package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/segment"
)

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

// SetMembership caches the membership result with a TTL.
func (sc *SegmentCache) SetMembership(ctx context.Context, segmentKey, userID, tenantID string, isMember bool, ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	val := "0"
	if isMember {
		val = "1"
	}
	_ = sc.client.Set(ctx, SegmentMemberKey(segmentKey, userID, tenantID), val, ttl)
}

// GetRecord returns the cached record lookup.
// Returns (record, found). On Redis errors it returns (nil, false) to fail-open.
func (sc *SegmentCache) GetRecord(ctx context.Context, segmentKey, datasetVersion, recordKey string) (*segment.Record, bool) {
	val, _ := sc.client.Get(ctx, SegmentRecordKey(segmentKey, datasetVersion, recordKey))
	if val == "" {
		return nil, false
	}
	var record segment.Record
	if err := json.Unmarshal([]byte(val), &record); err != nil {
		return nil, false
	}
	return &record, true
}

// SetRecord caches a record lookup with a TTL.
func (sc *SegmentCache) SetRecord(ctx context.Context, segmentKey, datasetVersion, recordKey string, record *segment.Record, ttl time.Duration) {
	if ttl <= 0 || record == nil {
		return
	}
	data, err := json.Marshal(record)
	if err != nil {
		return
	}
	_ = sc.client.Set(ctx, SegmentRecordKey(segmentKey, datasetVersion, recordKey), string(data), ttl)
}

// InvalidateSegment removes all cached membership and record entries for a segment.
func (sc *SegmentCache) InvalidateSegment(ctx context.Context, segmentKey string) {
	_ = sc.client.DelPattern(ctx, SegmentPattern(segmentKey))
	_ = sc.client.DelPattern(ctx, SegmentRecordPattern(segmentKey))
}

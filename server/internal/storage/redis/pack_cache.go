package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const (
	packTargetPrefix = "fe:pack:targets:"
	packTargetTTL    = 300 * time.Second
)

// PackCache implements pack.Cache using the Redis client.
type PackCache struct {
	client *Client
}

// NewPackCache creates a new pack cache backed by Redis.
func NewPackCache(client *Client) *PackCache {
	return &PackCache{client: client}
}

// PackTargetKey returns the Redis key for caching active feature keys for a target.
func PackTargetKey(tenantID, campusID, programID string) string {
	return fmt.Sprintf("%s%s:%s:%s", packTargetPrefix, tenantID, campusID, programID)
}

// GetActiveFeatureKeys returns the cached set of active feature keys.
// Returns (keys, found). On Redis errors it returns (nil, false) to fail-open.
func (pc *PackCache) GetActiveFeatureKeys(ctx context.Context, tenantID, campusID, programID string) ([]string, bool) {
	val, _ := pc.client.Get(ctx, PackTargetKey(tenantID, campusID, programID))
	if val == "" {
		return nil, false
	}
	var keys []string
	if err := json.Unmarshal([]byte(val), &keys); err != nil {
		return nil, false
	}
	return keys, true
}

// SetActiveFeatureKeys caches the set of active feature keys with a fixed TTL.
func (pc *PackCache) SetActiveFeatureKeys(ctx context.Context, tenantID, campusID, programID string, keys []string) {
	if keys == nil {
		keys = []string{}
	}
	data, err := json.Marshal(keys)
	if err != nil {
		return
	}
	_ = pc.client.Set(ctx, PackTargetKey(tenantID, campusID, programID), string(data), packTargetTTL)
}

// InvalidateAll removes all pack target cache entries.
func (pc *PackCache) InvalidateAll(ctx context.Context) {
	_ = pc.client.DelPattern(ctx, packTargetPrefix+"*")
}

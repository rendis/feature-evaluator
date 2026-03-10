package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/experiment"
)

const (
	experimentPrefix = "fe:exp:running:"
	experimentTTL    = 60 * time.Second
)

// ExperimentCache caches running experiments by feature key.
type ExperimentCache struct {
	client *Client
}

// NewExperimentCache creates a new experiment cache backed by Redis.
func NewExperimentCache(client *Client) *ExperimentCache {
	return &ExperimentCache{client: client}
}

func experimentKey(workspaceKey, featureKey string) string {
	return fmt.Sprintf("%s%s:%s", experimentPrefix, workspaceKey, featureKey)
}

// GetRunning returns the cached running experiment for a feature.
// Returns (experiment, found). On miss or error returns (nil, false).
func (ec *ExperimentCache) GetRunning(ctx context.Context, workspaceKey, featureKey string) (*experiment.Experiment, bool) {
	val, _ := ec.client.Get(ctx, experimentKey(workspaceKey, featureKey))
	if val == "" {
		return nil, false
	}
	// Sentinel value for "no running experiment"
	if val == "null" {
		return nil, true
	}
	var exp experiment.Experiment
	if err := json.Unmarshal([]byte(val), &exp); err != nil {
		return nil, false
	}
	return &exp, true
}

// SetRunning caches a running experiment for a feature key.
// Pass nil to cache the absence of a running experiment.
func (ec *ExperimentCache) SetRunning(ctx context.Context, workspaceKey, featureKey string, exp *experiment.Experiment) {
	var data []byte
	if exp == nil {
		data = []byte("null")
	} else {
		var err error
		data, err = json.Marshal(exp)
		if err != nil {
			return
		}
	}
	_ = ec.client.Set(ctx, experimentKey(workspaceKey, featureKey), string(data), experimentTTL)
}

// Invalidate removes cached experiment for a feature key.
func (ec *ExperimentCache) Invalidate(ctx context.Context, workspaceKey, featureKey string) {
	_ = ec.client.Del(ctx, experimentKey(workspaceKey, featureKey))
}

// InvalidateAll removes all cached experiment entries.
func (ec *ExperimentCache) InvalidateAll(ctx context.Context) {
	_ = ec.client.DelPattern(ctx, experimentPrefix+"*")
}

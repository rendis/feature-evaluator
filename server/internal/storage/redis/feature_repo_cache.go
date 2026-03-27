package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/domain/observability"
	"github.com/rendis/feature-evaluator/internal/domain/workspace"
)

// CachedFeatureRepo decorates a feature repository with Redis-backed snapshot caching.
type CachedFeatureRepo struct {
	base  feature.Repository
	cache *Client
}

// NewCachedFeatureRepo creates a Redis-backed feature repository decorator.
func NewCachedFeatureRepo(base feature.Repository, cache *Client) *CachedFeatureRepo {
	return &CachedFeatureRepo{base: base, cache: cache}
}

// Create implements feature.Repository.
func (r *CachedFeatureRepo) Create(ctx context.Context, feat *feature.Feature) error {
	if err := r.base.Create(ctx, feat); err != nil {
		return err
	}
	r.invalidate(ctx, feat.Key)
	return nil
}

// GetByKey implements feature.Repository.
func (r *CachedFeatureRepo) GetByKey(ctx context.Context, key string) (*feature.Feature, error) {
	start := time.Now()
	if cachedFeature, ok := r.loadCachedFeature(ctx, key); ok {
		r.recordLookup(ctx, *cachedFeature, observability.CacheStatusHit, time.Since(start), "cached")
		if trace, ok := observability.TraceRecorderFromContext(ctx); ok && trace != nil {
			trace.MarkUsedRedis()
		}
		return cachedFeature, nil
	}

	feat, err := r.base.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	status := observability.CacheStatusDisabled
	if r.cache != nil && feat.EvalCacheEnabled && feat.EvalCacheTTLSeconds > 0 {
		payload, marshalErr := json.Marshal(feat)
		if marshalErr == nil {
			_ = r.cache.Set(
				ctx,
				FeatureWorkspaceKey(workspace.KeyFromContext(ctx), feat.Key),
				payload,
				time.Duration(feat.EvalCacheTTLSeconds)*time.Second,
			)
		}
		status = observability.CacheStatusMiss
	} else if feat.EvalCacheEnabled && feat.EvalCacheTTLSeconds > 0 {
		status = observability.CacheStatusComputed
	}

	r.recordLookup(ctx, *feat, status, time.Since(start), "loaded")
	return feat, nil
}

func (r *CachedFeatureRepo) loadCachedFeature(ctx context.Context, key string) (*feature.Feature, bool) {
	if r.cache == nil || !r.cache.Available() {
		return nil, false
	}
	cacheKey := FeatureWorkspaceKey(workspace.KeyFromContext(ctx), key)
	cached, _ := r.cache.Get(ctx, cacheKey)
	if cached == "" {
		return nil, false
	}
	var feat feature.Feature
	if err := json.Unmarshal([]byte(cached), &feat); err != nil {
		_ = r.cache.Del(ctx, cacheKey)
		return nil, false
	}
	return &feat, true
}

// Update implements feature.Repository.
func (r *CachedFeatureRepo) Update(ctx context.Context, feat *feature.Feature) error {
	if err := r.base.Update(ctx, feat); err != nil {
		return err
	}
	r.invalidate(ctx, feat.Key)
	return nil
}

// Delete implements feature.Repository.
func (r *CachedFeatureRepo) Delete(ctx context.Context, key string) error {
	if err := r.base.Delete(ctx, key); err != nil {
		return err
	}
	r.invalidate(ctx, key)
	return nil
}

// List implements feature.Repository.
func (r *CachedFeatureRepo) List(ctx context.Context, params feature.ListParams) (*feature.ListResult, error) {
	return r.base.List(ctx, params)
}

// ListEnabled implements feature.Repository.
func (r *CachedFeatureRepo) ListEnabled(ctx context.Context) ([]feature.Feature, error) {
	return r.base.ListEnabled(ctx)
}

// Toggle implements feature.Repository.
func (r *CachedFeatureRepo) Toggle(ctx context.Context, key string, enabled bool, updatedBy string) error {
	if err := r.base.Toggle(ctx, key, enabled, updatedBy); err != nil {
		return err
	}
	r.invalidate(ctx, key)
	return nil
}

// AddRule implements feature.Repository.
func (r *CachedFeatureRepo) AddRule(ctx context.Context, featureKey string, rule *feature.Rule) error {
	if err := r.base.AddRule(ctx, featureKey, rule); err != nil {
		return err
	}
	r.invalidate(ctx, featureKey)
	return nil
}

// UpdateRule implements feature.Repository.
func (r *CachedFeatureRepo) UpdateRule(ctx context.Context, featureKey string, rule *feature.Rule) error {
	if err := r.base.UpdateRule(ctx, featureKey, rule); err != nil {
		return err
	}
	r.invalidate(ctx, featureKey)
	return nil
}

// DeleteRule implements feature.Repository.
func (r *CachedFeatureRepo) DeleteRule(ctx context.Context, featureKey string, ruleID string) error {
	if err := r.base.DeleteRule(ctx, featureKey, ruleID); err != nil {
		return err
	}
	r.invalidate(ctx, featureKey)
	return nil
}

// ReorderRules implements feature.Repository.
func (r *CachedFeatureRepo) ReorderRules(ctx context.Context, featureKey string, ruleIDs []string) error {
	if err := r.base.ReorderRules(ctx, featureKey, ruleIDs); err != nil {
		return err
	}
	r.invalidate(ctx, featureKey)
	return nil
}

func (r *CachedFeatureRepo) invalidate(ctx context.Context, key string) {
	if r.cache == nil {
		return
	}
	_ = r.cache.Del(ctx, FeatureWorkspaceKey(workspace.KeyFromContext(ctx), key))
}

func (r *CachedFeatureRepo) recordLookup(
	ctx context.Context,
	feat feature.Feature,
	status observability.CacheStatus,
	duration time.Duration,
	outcome string,
) {
	trace, ok := observability.TraceRecorderFromContext(ctx)
	if !ok || trace == nil {
		return
	}
	trace.RecordComponent(observability.ComponentTrace{
		Name:         "feature_lookup",
		CacheBackend: observability.CacheBackendRedis,
		CacheEnabled: feat.EvalCacheEnabled,
		CacheStatus:  status,
		TTLSeconds:   feat.EvalCacheTTLSeconds,
		DurationMs:   duration.Milliseconds(),
		Outcome:      outcome,
	})
}

var _ feature.Repository = (*CachedFeatureRepo)(nil)

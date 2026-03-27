package experiment

import (
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/domain/observability"
	"github.com/rendis/feature-evaluator/internal/domain/workspace"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

const z95 = 1.96

// TxManager coordinates multi-step persistence operations under one transaction.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(context.Context) error) error
}

// Service handles experiment business logic.
type Service struct {
	repo       Repository
	expRepo    ExposureRepository
	convRepo   ConversionRepository
	featureSvc *feature.Service
	cache      Cache
	txManager  TxManager
}

// NewService creates a new experiment service.
func NewService(repo Repository, expRepo ExposureRepository, convRepo ConversionRepository, featureSvc *feature.Service, cache Cache) *Service {
	return &Service{
		repo:       repo,
		expRepo:    expRepo,
		convRepo:   convRepo,
		featureSvc: featureSvc,
		cache:      cache,
	}
}

// SetTxManager configures transactional execution for multi-step state changes.
func (s *Service) SetTxManager(txManager TxManager) {
	s.txManager = txManager
}

// Create creates a new experiment after validation.
func (s *Service) Create(ctx context.Context, exp *Experiment) error {
	if exp.Name == "" {
		return apierror.NewBadRequest("experiment name is required", "error.experimentNameRequired")
	}
	if exp.FeatureKey == "" {
		return apierror.NewBadRequest("feature key is required", "error.featureKeyRequired")
	}
	if len(exp.Variants) < 2 {
		return apierror.NewBadRequest("at least 2 variants are required", "error.experimentMinVariants")
	}
	if err := validateVariants(exp.Variants); err != nil {
		return err
	}
	exp.NormalizeCacheConfig()

	// Verify feature exists.
	if _, err := s.featureSvc.GetByKey(ctx, exp.FeatureKey); err != nil {
		return err
	}

	// Check no running experiment for this feature.
	existing, err := s.repo.FindRunningByFeatureKey(ctx, exp.FeatureKey)
	if err != nil {
		return fmt.Errorf("checking running experiments: %w", err)
	}
	if existing != nil {
		return apierror.NewConflict(
			fmt.Sprintf("feature %q already has a running experiment", exp.FeatureKey),
			"error.experimentAlreadyRunning",
		)
	}

	now := time.Now().UTC()
	exp.Status = StatusDraft
	exp.CreatedAt = now
	exp.UpdatedAt = now

	if exp.Metrics == nil {
		exp.Metrics = []Metric{}
	}

	return s.repo.Create(ctx, exp)
}

// GetByID retrieves an experiment by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Experiment, error) {
	exp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting experiment %s: %w", id, err)
	}
	return exp, nil
}

// List returns all experiments.
func (s *Service) List(ctx context.Context) ([]Experiment, error) {
	return s.repo.List(ctx)
}

// Update updates a draft experiment.
func (s *Service) Update(ctx context.Context, exp *Experiment) error {
	existing, err := s.repo.GetByID(ctx, exp.ID)
	if err != nil {
		return err
	}
	if existing.Status != StatusDraft {
		return apierror.NewBadRequest("only draft experiments can be updated", "error.experimentNotDraft")
	}
	if exp.Name == "" {
		return apierror.NewBadRequest("experiment name is required", "error.experimentNameRequired")
	}
	if len(exp.Variants) < 2 {
		return apierror.NewBadRequest("at least 2 variants are required", "error.experimentMinVariants")
	}
	if err := validateVariants(exp.Variants); err != nil {
		return err
	}

	existing.Name = exp.Name
	existing.Description = exp.Description
	existing.Variants = exp.Variants
	existing.Metrics = exp.Metrics
	existing.LookupCacheEnabled = exp.LookupCacheEnabled
	existing.LookupCacheTTLSeconds = exp.LookupCacheTTLSeconds
	if existing.Metrics == nil {
		existing.Metrics = []Metric{}
	}
	existing.NormalizeCacheConfig()
	existing.UpdatedAt = time.Now().UTC()

	return s.repo.Update(ctx, existing)
}

// Start transitions an experiment from draft/paused to running.
func (s *Service) Start(ctx context.Context, id string) error {
	exp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if exp.Status != StatusDraft && exp.Status != StatusPaused {
		return apierror.NewBadRequest(
			fmt.Sprintf("cannot start experiment in %q status", exp.Status),
			"error.experimentInvalidTransition",
		)
	}

	// Check no other running experiment for this feature.
	running, err := s.repo.FindRunningByFeatureKey(ctx, exp.FeatureKey)
	if err != nil {
		return fmt.Errorf("checking running experiments: %w", err)
	}
	if running != nil && running.ID != id {
		return apierror.NewConflict(
			fmt.Sprintf("feature %q already has a running experiment", exp.FeatureKey),
			"error.experimentAlreadyRunning",
		)
	}

	now := time.Now().UTC()
	exp.Status = StatusRunning
	if exp.StartedAt == nil {
		exp.StartedAt = &now
	}
	exp.UpdatedAt = now

	if err := s.repo.Update(ctx, exp); err != nil {
		return err
	}
	s.invalidateCache(ctx, exp.FeatureKey)
	return nil
}

// Pause transitions a running experiment to paused.
func (s *Service) Pause(ctx context.Context, id string) error {
	exp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if exp.Status != StatusRunning {
		return apierror.NewBadRequest("only running experiments can be paused", "error.experimentNotRunning")
	}

	exp.Status = StatusPaused
	exp.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, exp); err != nil {
		return err
	}
	s.invalidateCache(ctx, exp.FeatureKey)
	return nil
}

// Complete transitions a running/paused experiment to completed.
func (s *Service) Complete(ctx context.Context, id string) error {
	exp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if exp.Status != StatusRunning && exp.Status != StatusPaused {
		return apierror.NewBadRequest("only running or paused experiments can be completed", "error.experimentInvalidTransition")
	}

	now := time.Now().UTC()
	exp.Status = StatusCompleted
	exp.CompletedAt = &now
	exp.UpdatedAt = now

	if err := s.repo.Update(ctx, exp); err != nil {
		return err
	}
	s.invalidateCache(ctx, exp.FeatureKey)
	return nil
}

// DeclareWinner declares a winning variant and applies its value as the feature's defaultValue.
func (s *Service) DeclareWinner(ctx context.Context, id, variantKey string) error {
	var featureKey string
	apply := func(txCtx context.Context) error {
		exp, err := s.repo.GetByID(txCtx, id)
		if err != nil {
			return err
		}
		if exp.Status != StatusCompleted {
			return apierror.NewBadRequest("experiment must be completed before declaring a winner", "error.experimentNotCompleted")
		}

		var winnerValue any
		found := false
		for _, v := range exp.Variants {
			if v.Key == variantKey {
				winnerValue = v.Value
				found = true
				break
			}
		}
		if !found {
			return apierror.NewBadRequest(
				fmt.Sprintf("variant %q not found in experiment", variantKey),
				"error.experimentVariantNotFound",
			)
		}

		f, err := s.featureSvc.GetByKey(txCtx, exp.FeatureKey)
		if err != nil {
			return fmt.Errorf("getting feature for winner application: %w", err)
		}
		f.DefaultValue = winnerValue
		if err := s.featureSvc.Update(txCtx, f); err != nil {
			return fmt.Errorf("applying winner value to feature: %w", err)
		}

		exp.WinnerKey = variantKey
		exp.UpdatedAt = time.Now().UTC()
		featureKey = exp.FeatureKey

		return s.repo.Update(txCtx, exp)
	}

	if s.txManager != nil {
		if err := s.txManager.WithinTx(ctx, apply); err != nil {
			return err
		}
	} else if err := apply(ctx); err != nil {
		return err
	}

	s.invalidateCache(ctx, featureKey)
	return nil
}

// AssignVariant deterministically assigns a user to a variant using FNV-1a.
func AssignVariant(experimentID, userID string, variants []Variant) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(experimentID + ":" + userID))
	hashValue := h.Sum32()

	bucket := hashValue % 100
	var cumulative uint32
	for _, v := range variants {
		weight, ok := safeVariantWeight(v.Weight)
		if !ok {
			continue
		}
		cumulative += weight
		if bucket < cumulative {
			return v.Key
		}
	}
	return variants[len(variants)-1].Key
}

func safeVariantWeight(weight int) (uint32, bool) {
	if weight < 0 || weight > 100 {
		return 0, false
	}

	return uint32(weight), true
}

// RecordExposure records that a user was exposed to a variant.
func (s *Service) RecordExposure(ctx context.Context, exp *Exposure) error {
	exp.CreatedAt = time.Now().UTC()
	return s.expRepo.Upsert(ctx, exp)
}

// RecordConversion records a conversion event for a user.
func (s *Service) RecordConversion(ctx context.Context, conv *Conversion) error {
	// Verify experiment exists and is running or completed.
	exp, err := s.repo.GetByID(ctx, conv.ExperimentID)
	if err != nil {
		return err
	}
	if exp.Status != StatusRunning && exp.Status != StatusCompleted {
		return apierror.NewBadRequest("experiment is not running", "error.experimentNotRunning")
	}

	// Verify metric key exists in experiment.
	metricFound := false
	for _, m := range exp.Metrics {
		if m.Key == conv.MetricKey {
			metricFound = true
			break
		}
	}
	if !metricFound {
		return apierror.NewBadRequest(
			fmt.Sprintf("metric %q not found in experiment", conv.MetricKey),
			"error.experimentMetricNotFound",
		)
	}

	// Resolve variant key from exposure record.
	exposure, err := s.expRepo.Find(ctx, conv.ExperimentID, conv.UserID)
	if err != nil {
		return fmt.Errorf("finding exposure for conversion: %w", err)
	}
	if exposure == nil {
		return apierror.NewBadRequest("user has no exposure for this experiment", "error.experimentNoExposure")
	}
	conv.VariantKey = exposure.VariantKey

	conv.CreatedAt = time.Now().UTC()
	return s.convRepo.Create(ctx, conv)
}

// GetResults computes experiment results with variant stats.
func (s *Service) GetResults(ctx context.Context, id string) (*Results, error) {
	exp, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	exposureCounts, err := s.expRepo.CountByVariant(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("counting exposures: %w", err)
	}

	// Use the first metric for conversion counting. If no metrics, use exposures only.
	var conversionCounts map[string]int64
	if len(exp.Metrics) > 0 {
		conversionCounts, err = s.convRepo.CountByVariant(ctx, id, exp.Metrics[0].Key)
		if err != nil {
			return nil, fmt.Errorf("counting conversions: %w", err)
		}
	}

	var totalExposures, totalConversions int64
	variants := make([]VariantStats, 0, len(exp.Variants))

	for _, v := range exp.Variants {
		exposures := exposureCounts[v.Key]
		conversions := int64(0)
		if conversionCounts != nil {
			conversions = conversionCounts[v.Key]
		}

		totalExposures += exposures
		totalConversions += conversions

		rate := float64(0)
		if exposures > 0 {
			rate = float64(conversions) / float64(exposures)
		}

		low, high := WilsonScore(conversions, exposures, z95)

		variants = append(variants, VariantStats{
			VariantKey:     v.Key,
			Exposures:      exposures,
			Conversions:    conversions,
			ConversionRate: rate,
			ConfidenceLow:  low,
			ConfidenceHigh: high,
		})
	}

	// Simple significance check: do any two variant CIs not overlap?
	significant := false
	for i := 0; i < len(variants); i++ {
		for j := i + 1; j < len(variants); j++ {
			if variants[i].ConfidenceHigh < variants[j].ConfidenceLow ||
				variants[j].ConfidenceHigh < variants[i].ConfidenceLow {
				significant = true
				break
			}
		}
		if significant {
			break
		}
	}

	return &Results{
		ExperimentID:     id,
		TotalExposures:   totalExposures,
		TotalConversions: totalConversions,
		Variants:         variants,
		IsSignificant:    significant,
	}, nil
}

// FindRunningByFeatureKey returns the running experiment for a feature, if any.
// Uses Redis cache to avoid hitting the database on every evaluation.
func (s *Service) FindRunningByFeatureKey(ctx context.Context, featureKey string) (*Experiment, error) {
	trace, _ := observability.TraceRecorderFromContext(ctx)
	stepStart := time.Now()
	wsKey := workspace.KeyFromContext(ctx)
	if exp, found := s.cachedRunningExperiment(ctx, wsKey, featureKey, trace, stepStart); found {
		return exp, nil
	}

	exp, err := s.repo.FindRunningByFeatureKey(ctx, featureKey)
	if err != nil {
		s.recordExperimentLookupTrace(trace, featureKey, s.cache != nil, 0, stepStart, "error")
		return nil, err
	}

	s.cacheRunningExperiment(ctx, wsKey, featureKey, exp)
	s.recordExperimentLookupTrace(trace, featureKey, exp != nil && exp.LookupCacheEnabled, experimentLookupTTL(exp), stepStart, "resolved")
	return exp, nil
}

func (s *Service) cachedRunningExperiment(
	ctx context.Context,
	wsKey, featureKey string,
	trace observability.TraceRecorder,
	stepStart time.Time,
) (*Experiment, bool) {
	if s.cache == nil {
		return nil, false
	}
	exp, found := s.cache.GetRunning(ctx, wsKey, featureKey)
	if !found || !canUseCachedRunningExperiment(exp) {
		return nil, false
	}
	s.recordExperimentLookupTrace(trace, featureKey, true, experimentLookupTTL(exp), stepStart, "cached")
	return exp, true
}

func (s *Service) cacheRunningExperiment(ctx context.Context, wsKey, featureKey string, exp *Experiment) {
	if s.cache == nil || !canCacheRunningExperiment(exp) {
		return
	}
	s.cache.SetRunning(ctx, wsKey, featureKey, exp, time.Duration(exp.LookupCacheTTLSeconds)*time.Second)
}

func (s *Service) recordExperimentLookupTrace(
	trace observability.TraceRecorder,
	featureKey string,
	cacheEnabled bool,
	ttlSeconds int,
	stepStart time.Time,
	outcome string,
) {
	if trace == nil {
		return
	}
	cacheStatus := observability.CacheStatusMiss
	if outcome == "cached" {
		cacheStatus = observability.CacheStatusHit
		trace.MarkUsedRedis()
	} else if !cacheEnabled {
		cacheStatus = observability.CacheStatusNotApplicable
	} else if outcome == "error" {
		cacheStatus = observability.CacheStatusComputed
	}
	trace.RecordComponent(observability.ComponentTrace{
		Name:         "experiment_lookup:" + featureKey,
		CacheBackend: observability.CacheBackendRedis,
		CacheEnabled: cacheEnabled,
		CacheStatus:  cacheStatus,
		TTLSeconds:   ttlSeconds,
		DurationMs:   time.Since(stepStart).Milliseconds(),
		Outcome:      outcome,
	})
}

func canUseCachedRunningExperiment(exp *Experiment) bool {
	return exp == nil || (exp.LookupCacheEnabled && exp.LookupCacheTTLSeconds > 0)
}

func canCacheRunningExperiment(exp *Experiment) bool {
	return exp != nil && exp.LookupCacheEnabled && exp.LookupCacheTTLSeconds > 0
}

func experimentLookupTTL(exp *Experiment) int {
	if exp == nil {
		return 0
	}
	return exp.LookupCacheTTLSeconds
}

func (s *Service) invalidateCache(ctx context.Context, featureKey string) {
	if s.cache != nil {
		wsKey := workspace.KeyFromContext(ctx)
		s.cache.Invalidate(ctx, wsKey, featureKey)
	}
}

func validateVariants(variants []Variant) error {
	totalWeight := 0
	keys := make(map[string]bool, len(variants))
	for _, v := range variants {
		if v.Key == "" {
			return apierror.NewBadRequest("variant key is required", "error.experimentVariantKeyRequired")
		}
		if keys[v.Key] {
			return apierror.NewBadRequest(
				fmt.Sprintf("duplicate variant key: %s", v.Key),
				"error.experimentDuplicateVariant",
			)
		}
		keys[v.Key] = true
		if v.Weight < 0 || v.Weight > 100 {
			return apierror.NewBadRequest(
				fmt.Sprintf("variant %q weight must be between 0 and 100", v.Key),
				"error.experimentInvalidWeight",
			)
		}
		totalWeight += v.Weight
	}
	if totalWeight != 100 {
		return apierror.NewBadRequest(
			fmt.Sprintf("variant weights must sum to 100, got %d", totalWeight),
			"error.experimentWeightSum",
		)
	}
	return nil
}

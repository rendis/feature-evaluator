package pack

import (
	"context"
	"fmt"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/domain/resourcekey"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// Cache defines the interface for pack feature key caching.
type Cache interface {
	GetActiveFeatureKeys(ctx context.Context, tenantID, campusID, programID string) ([]string, bool)
	SetActiveFeatureKeys(ctx context.Context, tenantID, campusID, programID string, keys []string)
	InvalidateAll(ctx context.Context)
}

// Service handles pack business logic.
type Service struct {
	repo           Repository
	activationRepo ActivationRepository
	featureRepo    feature.Repository
	cache          Cache
}

// NewService creates a new pack service.
func NewService(repo Repository, activationRepo ActivationRepository, featureRepo feature.Repository, cache Cache) *Service {
	return &Service{
		repo:           repo,
		activationRepo: activationRepo,
		featureRepo:    featureRepo,
		cache:          cache,
	}
}

// Create creates a new pack after validation.
func (s *Service) Create(ctx context.Context, p *Pack) error {
	p.Key = resourcekey.Normalize(p.Key)
	if !resourcekey.IsValid(p.Key) {
		return apierror.NewBadRequest(
			fmt.Sprintf("invalid pack key format: %s", p.Key),
			"error.invalidPackKey",
		)
	}
	if p.Name == "" {
		return apierror.NewBadRequest("pack name is required", "error.packNameRequired")
	}
	if len(p.FeatureKeys) > MaxFeaturesPerPack {
		return apierror.NewBadRequest(
			fmt.Sprintf("a pack can have at most %d features", MaxFeaturesPerPack),
			"error.tooManyPackFeatures",
		)
	}

	if len(p.FeatureKeys) > 0 {
		if err := s.validateFeatureKeys(ctx, p.FeatureKeys); err != nil {
			return err
		}
	}

	if len(p.InheritsFrom) > 0 {
		if err := s.validateInheritance(ctx, p.Key, p.InheritsFrom); err != nil {
			return err
		}
	}

	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	if p.FeatureKeys == nil {
		p.FeatureKeys = []string{}
	}
	if p.InheritsFrom == nil {
		p.InheritsFrom = []string{}
	}

	return s.repo.Create(ctx, p)
}

// GetByKey retrieves a pack by its unique key.
func (s *Service) GetByKey(ctx context.Context, key string) (*Pack, error) {
	p, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("getting pack %s: %w", key, err)
	}
	return p, nil
}

// Update updates an existing pack.
func (s *Service) Update(ctx context.Context, p *Pack) error {
	if p.Name == "" {
		return apierror.NewBadRequest("pack name is required", "error.packNameRequired")
	}
	if len(p.FeatureKeys) > MaxFeaturesPerPack {
		return apierror.NewBadRequest(
			fmt.Sprintf("a pack can have at most %d features", MaxFeaturesPerPack),
			"error.tooManyPackFeatures",
		)
	}

	if len(p.FeatureKeys) > 0 {
		if err := s.validateFeatureKeys(ctx, p.FeatureKeys); err != nil {
			return err
		}
	}

	if len(p.InheritsFrom) > 0 {
		if err := s.validateInheritance(ctx, p.Key, p.InheritsFrom); err != nil {
			return err
		}
	}

	if p.InheritsFrom == nil {
		p.InheritsFrom = []string{}
	}

	p.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, p); err != nil {
		return err
	}

	s.cache.InvalidateAll(ctx)
	return nil
}

// Delete removes a pack by key.
func (s *Service) Delete(ctx context.Context, key string) error {
	if err := s.repo.Delete(ctx, key); err != nil {
		return err
	}
	s.cache.InvalidateAll(ctx)
	return nil
}

// List returns all packs.
func (s *Service) List(ctx context.Context) ([]Pack, error) {
	return s.repo.List(ctx)
}

// Toggle enables or disables a pack.
func (s *Service) Toggle(ctx context.Context, key string, enabled bool, updatedBy string) error {
	if err := s.repo.Toggle(ctx, key, enabled, updatedBy); err != nil {
		return err
	}
	s.cache.InvalidateAll(ctx)
	return nil
}

// FindByFeatureKey returns all packs that contain a given feature key.
func (s *Service) FindByFeatureKey(ctx context.Context, featureKey string) ([]Pack, error) {
	return s.repo.FindByFeatureKey(ctx, featureKey)
}

// Activate creates a pack activation for a target.
func (s *Service) Activate(ctx context.Context, a *Activation) error {
	if !ValidTargetType(string(a.TargetType)) {
		return apierror.NewBadRequest(
			fmt.Sprintf("invalid target type: %s", a.TargetType),
			"error.invalidTargetType",
		)
	}
	if a.TargetID == "" {
		return apierror.NewBadRequest("target ID is required", "error.targetIdRequired")
	}

	// Verify pack exists
	if _, err := s.repo.GetByKey(ctx, a.PackKey); err != nil {
		return err
	}

	a.ActivatedAt = time.Now().UTC()

	if err := s.activationRepo.Create(ctx, a); err != nil {
		return err
	}

	s.cache.InvalidateAll(ctx)
	return nil
}

// Deactivate removes a pack activation for a target.
func (s *Service) Deactivate(ctx context.Context, packKey string, targetType TargetType, targetID string) error {
	if err := s.activationRepo.Delete(ctx, packKey, targetType, targetID); err != nil {
		return err
	}
	s.cache.InvalidateAll(ctx)
	return nil
}

// ListActivations returns all activations for a pack.
func (s *Service) ListActivations(ctx context.Context, packKey string) ([]Activation, error) {
	// Verify pack exists
	if _, err := s.repo.GetByKey(ctx, packKey); err != nil {
		return nil, err
	}
	return s.activationRepo.ListByPack(ctx, packKey)
}

// FindByTarget returns all activations for a given target.
func (s *Service) FindByTarget(ctx context.Context, targetType TargetType, targetID string) ([]Activation, error) {
	return s.activationRepo.FindByTarget(ctx, targetType, targetID)
}

// FindActiveFeatureKeys returns all feature keys granted via pack activations for a target.
// Uses Redis cache with a 300s TTL.
func (s *Service) FindActiveFeatureKeys(ctx context.Context, tenantID, campusID, programID string) ([]string, error) {
	if keys, found := s.cache.GetActiveFeatureKeys(ctx, tenantID, campusID, programID); found {
		return keys, nil
	}

	keys, err := s.activationRepo.FindActiveFeatureKeys(ctx, tenantID, campusID, programID)
	if err != nil {
		return nil, err
	}

	s.cache.SetActiveFeatureKeys(ctx, tenantID, campusID, programID, keys)
	return keys, nil
}

// ResolveFeatureKeys returns all feature keys for a pack, including inherited ones.
func (s *Service) ResolveFeatureKeys(ctx context.Context, packKey string) ([]string, error) {
	p, err := s.repo.GetByKey(ctx, packKey)
	if err != nil {
		return nil, err
	}
	return s.resolveFeatureKeysRecursive(ctx, p, make(map[string]bool))
}

func (s *Service) resolveFeatureKeysRecursive(ctx context.Context, p *Pack, visited map[string]bool) ([]string, error) {
	if visited[p.Key] {
		return nil, nil
	}
	visited[p.Key] = true

	keys := make([]string, len(p.FeatureKeys))
	copy(keys, p.FeatureKeys)

	for _, parentKey := range p.InheritsFrom {
		parent, err := s.repo.GetByKey(ctx, parentKey)
		if err != nil {
			continue
		}
		if !parent.Enabled {
			continue
		}
		parentKeys, err := s.resolveFeatureKeysRecursive(ctx, parent, visited)
		if err != nil {
			continue
		}
		keys = append(keys, parentKeys...)
	}

	return deduplicate(keys), nil
}

// ResolveTierKeysForFeature returns tier keys from all packs that contain the given feature.
func (s *Service) ResolveTierKeysForFeature(ctx context.Context, featureKey string) []string {
	packs, err := s.repo.FindByFeatureKey(ctx, featureKey)
	if err != nil {
		return nil
	}
	tierKeys := make([]string, 0)
	for _, p := range packs {
		if p.TierKey != nil && *p.TierKey != "" {
			tierKeys = append(tierKeys, *p.TierKey)
		}
	}
	return deduplicate(tierKeys)
}

// validateInheritance checks parent packs exist and that no cycle would be created.
func (s *Service) validateInheritance(ctx context.Context, packKey string, parentKeys []string) error {
	for _, key := range parentKeys {
		if _, err := s.repo.GetByKey(ctx, key); err != nil {
			return apierror.NewBadRequest(
				fmt.Sprintf("parent pack %q does not exist", key),
				"error.packNotFound",
			)
		}
	}

	allInheritance, err := s.repo.ListAllInheritance(ctx)
	if err != nil {
		return fmt.Errorf("loading inheritance graph: %w", err)
	}

	if err := DetectCycle(packKey, parentKeys, allInheritance); err != nil {
		return apierror.NewBadRequest(err.Error(), "error.packInheritanceCycle")
	}

	return nil
}

// validateFeatureKeys checks that all feature keys exist.
func (s *Service) validateFeatureKeys(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if _, err := s.featureRepo.GetByKey(ctx, key); err != nil {
			return apierror.NewBadRequest(
				fmt.Sprintf("feature %q does not exist", key),
				"error.featureNotFound",
			)
		}
	}
	return nil
}

// deduplicate removes duplicate strings preserving order.
func deduplicate(keys []string) []string {
	seen := make(map[string]bool, len(keys))
	result := make([]string, 0, len(keys))
	for _, k := range keys {
		if !seen[k] {
			seen[k] = true
			result = append(result, k)
		}
	}
	return result
}

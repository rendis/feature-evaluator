package tag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/resourcekey"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// Slugify converts a name to a URL-friendly key.
func Slugify(name string) string {
	return resourcekey.Normalize(name)
}

// Repository defines the persistence interface for tags.
type Repository interface {
	Create(ctx context.Context, t *Tag) error
	Update(ctx context.Context, t *Tag) error
	Delete(ctx context.Context, key string) error
	FindByKey(ctx context.Context, key string) (*Tag, error)
	FindByKeys(ctx context.Context, keys []string) ([]Tag, error)
	List(ctx context.Context, search string) ([]Tag, error)
	CountFeaturesByTag(ctx context.Context, tagKey string) (int64, error)
}

// Service handles tag business logic.
type Service struct {
	repo Repository
}

// NewService creates a new tag service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new tag with an auto-generated key from the name.
func (s *Service) Create(ctx context.Context, name, color, createdBy string) (*Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apierror.NewBadRequest("tag name is required", "error.tagNameRequired")
	}
	if color == "" {
		return nil, apierror.NewBadRequest("tag color is required", "error.tagColorRequired")
	}

	key := Slugify(name)
	if key == "" {
		return nil, apierror.NewBadRequest("tag name produces empty key", "error.tagNameInvalid")
	}

	now := time.Now().UTC()
	t := &Tag{
		Key:       key,
		Name:      name,
		Color:     color,
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: createdBy,
	}

	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Update updates a tag's name and color. The key is immutable.
func (s *Service) Update(ctx context.Context, key, name, color string) (*Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apierror.NewBadRequest("tag name is required", "error.tagNameRequired")
	}
	if color == "" {
		return nil, apierror.NewBadRequest("tag color is required", "error.tagColorRequired")
	}

	t, err := s.repo.FindByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	t.Name = name
	t.Color = color
	t.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Delete removes a tag, rejecting if any feature uses it.
func (s *Service) Delete(ctx context.Context, key string) error {
	count, err := s.repo.CountFeaturesByTag(ctx, key)
	if err != nil {
		return fmt.Errorf("counting features for tag %q: %w", key, err)
	}
	if count > 0 {
		return apierror.NewConflict(
			fmt.Sprintf("tag %q is used by %d feature(s)", key, count),
			"error.tagInUse",
		)
	}
	return s.repo.Delete(ctx, key)
}

// List returns all tags, optionally filtered by search.
func (s *Service) List(ctx context.Context, search string) ([]Tag, error) {
	return s.repo.List(ctx, search)
}

// FindByKeys returns tags matching the given keys.
func (s *Service) FindByKeys(ctx context.Context, keys []string) ([]Tag, error) {
	if len(keys) == 0 {
		return []Tag{}, nil
	}
	return s.repo.FindByKeys(ctx, keys)
}

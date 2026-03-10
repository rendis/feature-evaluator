package tier

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

// Repository defines the persistence interface for tiers.
type Repository interface {
	Create(ctx context.Context, t *Tier) error
	Update(ctx context.Context, t *Tier) error
	Delete(ctx context.Context, key string) error
	FindByKey(ctx context.Context, key string) (*Tier, error)
	FindByKeys(ctx context.Context, keys []string) ([]Tier, error)
	List(ctx context.Context, search string) ([]Tier, error)
	CountPacksByTier(ctx context.Context, tierKey string) (int64, error)
}

// IconRepository defines the persistence interface for custom tier icons.
type IconRepository interface {
	CreateIcon(ctx context.Context, icon *TierIcon) error
	DeleteIcon(ctx context.Context, id string) error
	FindIconByID(ctx context.Context, id string) (*TierIcon, error)
	ListIcons(ctx context.Context) ([]TierIcon, error)
}

// Service handles tier business logic.
type Service struct {
	repo     Repository
	iconRepo IconRepository
}

// NewService creates a new tier service.
func NewService(repo Repository, iconRepo IconRepository) *Service {
	return &Service{repo: repo, iconRepo: iconRepo}
}

// Create creates a new tier with an auto-generated key from the name.
func (s *Service) Create(ctx context.Context, name string, level int, color, icon, createdBy string) (*Tier, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apierror.NewBadRequest("tier name is required", "error.tierNameRequired")
	}
	if level <= 0 {
		return nil, apierror.NewBadRequest("tier level must be greater than 0", "error.tierLevelInvalid")
	}
	if err := validateIcon(icon); err != nil {
		return nil, err
	}

	key := Slugify(name)
	if key == "" {
		return nil, apierror.NewBadRequest("tier name produces empty key", "error.tierNameInvalid")
	}

	now := time.Now().UTC()
	t := &Tier{
		Key:       key,
		Name:      name,
		Level:     level,
		Color:     color,
		Icon:      icon,
		CreatedAt: now,
		UpdatedAt: now,
		CreatedBy: createdBy,
	}

	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Update updates a tier's name, level, color, and icon. The key is immutable.
func (s *Service) Update(ctx context.Context, key, name string, level int, color, icon string) (*Tier, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apierror.NewBadRequest("tier name is required", "error.tierNameRequired")
	}
	if level <= 0 {
		return nil, apierror.NewBadRequest("tier level must be greater than 0", "error.tierLevelInvalid")
	}
	if err := validateIcon(icon); err != nil {
		return nil, err
	}

	t, err := s.repo.FindByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	t.Name = name
	t.Level = level
	t.Color = color
	t.Icon = icon
	t.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// Delete removes a tier, rejecting if any pack uses it.
func (s *Service) Delete(ctx context.Context, key string) error {
	count, err := s.repo.CountPacksByTier(ctx, key)
	if err != nil {
		return fmt.Errorf("counting packs for tier %q: %w", key, err)
	}
	if count > 0 {
		return apierror.NewConflict(
			fmt.Sprintf("tier %q is used by %d pack(s)", key, count),
			"error.tierInUse",
		)
	}
	return s.repo.Delete(ctx, key)
}

// List returns all tiers, optionally filtered by search.
func (s *Service) List(ctx context.Context, search string) ([]Tier, error) {
	return s.repo.List(ctx, search)
}

// FindByKey returns a single tier by its key.
func (s *Service) FindByKey(ctx context.Context, key string) (*Tier, error) {
	return s.repo.FindByKey(ctx, key)
}

// FindByKeys returns tiers matching the given keys.
func (s *Service) FindByKeys(ctx context.Context, keys []string) ([]Tier, error) {
	if len(keys) == 0 {
		return []Tier{}, nil
	}
	return s.repo.FindByKeys(ctx, keys)
}

// UploadIcon stores a custom tier icon.
func (s *Service) UploadIcon(ctx context.Context, name, contentType string, data []byte, createdBy string) (*TierIcon, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apierror.NewBadRequest("icon name is required", "error.iconNameRequired")
	}
	if contentType != "image/svg+xml" && contentType != "image/png" {
		return nil, apierror.NewBadRequest(
			"icon content type must be image/svg+xml or image/png",
			"error.iconContentTypeInvalid",
		)
	}
	if len(data) > MaxIconSize {
		return nil, apierror.NewBadRequest(
			fmt.Sprintf("icon size exceeds maximum of %d bytes", MaxIconSize),
			"error.iconSizeTooLarge",
		)
	}

	icon := &TierIcon{
		Name:        name,
		ContentType: contentType,
		Data:        data,
		CreatedAt:   time.Now().UTC(),
		CreatedBy:   createdBy,
	}

	if err := s.iconRepo.CreateIcon(ctx, icon); err != nil {
		return nil, err
	}
	return icon, nil
}

// DeleteIcon removes a custom tier icon by ID.
func (s *Service) DeleteIcon(ctx context.Context, id string) error {
	return s.iconRepo.DeleteIcon(ctx, id)
}

// ListIcons returns all custom tier icons.
func (s *Service) ListIcons(ctx context.Context) ([]TierIcon, error) {
	return s.iconRepo.ListIcons(ctx)
}

// validateIcon checks that the icon value starts with "builtin:" or "custom:".
func validateIcon(icon string) error {
	if !strings.HasPrefix(icon, "builtin:") && !strings.HasPrefix(icon, "custom:") {
		return apierror.NewBadRequest(
			"icon must start with \"builtin:\" or \"custom:\"",
			"error.tierIconInvalid",
		)
	}
	return nil
}

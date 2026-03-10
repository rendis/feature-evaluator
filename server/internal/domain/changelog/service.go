package changelog

import (
	"context"
	"fmt"
	"time"
)

// Service handles changelog business logic.
type Service struct {
	repo Repository
}

// NewService creates a new changelog service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Record persists a change entry. Intended to be called in a fire-and-forget goroutine.
func (s *Service) Record(ctx context.Context, entry *ChangeEntry) error {
	entry.CreatedAt = time.Now().UTC()
	if err := s.repo.Create(ctx, entry); err != nil {
		return fmt.Errorf("recording changelog for %s/%s: %w", entry.EntityType, entry.EntityKey, err)
	}
	return nil
}

// List returns a paginated, filtered list of changelog entries.
func (s *Service) List(ctx context.Context, params ListParams) (*ListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	return s.repo.List(ctx, params)
}

// ListByEntity returns changelog entries for a specific entity.
func (s *Service) ListByEntity(ctx context.Context, entityType, entityKey string, params ListParams) (*ListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	return s.repo.ListByEntity(ctx, entityType, entityKey, params)
}

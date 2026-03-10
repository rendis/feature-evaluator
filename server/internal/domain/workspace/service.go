package workspace

import (
	"context"
	"fmt"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/resourcekey"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// Service handles workspace business logic.
type Service struct {
	repo Repository
}

// NewService creates a new workspace service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new workspace after validation.
func (s *Service) Create(ctx context.Context, w *Workspace) error {
	w.Key = resourcekey.Normalize(w.Key)
	if !resourcekey.IsValid(w.Key) {
		return apierror.NewBadRequest(
			fmt.Sprintf("invalid workspace key format: %s", w.Key),
			"error.invalidWorkspaceKey",
		)
	}
	if w.Name == "" {
		return apierror.NewBadRequest("workspace name is required", "error.workspaceNameRequired")
	}

	now := time.Now().UTC()
	w.CreatedAt = now
	w.UpdatedAt = now

	return s.repo.Create(ctx, w)
}

// GetByKey retrieves a workspace by its key.
func (s *Service) GetByKey(ctx context.Context, key string) (*Workspace, error) {
	w, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("getting workspace %s: %w", key, err)
	}
	return w, nil
}

// Update updates an existing workspace.
func (s *Service) Update(ctx context.Context, w *Workspace) error {
	if w.Name == "" {
		return apierror.NewBadRequest("workspace name is required", "error.workspaceNameRequired")
	}
	w.UpdatedAt = time.Now().UTC()
	return s.repo.Update(ctx, w)
}

// Archive marks a workspace as archived.
func (s *Service) Archive(ctx context.Context, key, archivedBy string) error {
	return s.repo.Archive(ctx, key, archivedBy)
}

// Restore marks a workspace as active again.
func (s *Service) Restore(ctx context.Context, key string) error {
	return s.repo.Restore(ctx, key)
}

// Delete is a compatibility alias for Archive.
func (s *Service) Delete(ctx context.Context, key, archivedBy string) error {
	return s.Archive(ctx, key, archivedBy)
}

// List returns workspaces, excluding archived ones by default.
func (s *Service) List(ctx context.Context, includeArchived bool) ([]Workspace, error) {
	return s.repo.List(ctx, includeArchived)
}

// CountActive returns the number of active workspaces.
func (s *Service) CountActive(ctx context.Context) (int64, error) {
	return s.repo.CountActive(ctx)
}

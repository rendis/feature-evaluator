package member

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// Service handles member business logic.
type Service struct {
	repo Repository
}

// NewService creates a new member service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create registers a new team member.
func (s *Service) Create(ctx context.Context, m *Member) error {
	if !m.Role.Valid() {
		return fmt.Errorf("invalid role: %s", m.Role)
	}
	m.Email = strings.ToLower(strings.TrimSpace(m.Email))
	now := time.Now().UTC()
	m.CreatedAt = now
	m.UpdatedAt = now
	return s.repo.Create(ctx, m)
}

// GetByID retrieves a member by their ID.
func (s *Service) GetByID(ctx context.Context, id string) (*Member, error) {
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting member %s: %w", id, err)
	}
	return m, nil
}

// GetByEmail retrieves a member by email.
func (s *Service) GetByEmail(ctx context.Context, email string) (*Member, error) {
	m, err := s.repo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return nil, fmt.Errorf("getting member by email: %w", err)
	}
	return m, nil
}

// List returns all team members.
func (s *Service) List(ctx context.Context) ([]Member, error) {
	return s.repo.List(ctx)
}

// UpdateRole changes a member's role with ownership protections.
func (s *Service) UpdateRole(ctx context.Context, id string, newRole Role, actorEmail string) error {
	if !newRole.Valid() {
		return fmt.Errorf("invalid role: %s", newRole)
	}

	target, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("getting target member %s: %w", id, err)
	}

	// Self-demotion prevention
	if target.Email == actorEmail && target.Role == RoleOwner && newRole != RoleOwner {
		return apierror.NewSelfDemotion()
	}

	// Last owner protection
	if target.Role == RoleOwner && newRole != RoleOwner {
		count, err := s.repo.CountByRole(ctx, RoleOwner)
		if err != nil {
			return fmt.Errorf("counting owners: %w", err)
		}
		if count <= 1 {
			return apierror.NewLastOwner()
		}
	}

	return s.repo.UpdateRole(ctx, id, newRole)
}

// Delete removes a member with ownership protections.
func (s *Service) Delete(ctx context.Context, id string) error {
	target, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("getting member %s for deletion: %w", id, err)
	}

	if target.Role == RoleOwner {
		count, err := s.repo.CountByRole(ctx, RoleOwner)
		if err != nil {
			return fmt.Errorf("counting owners: %w", err)
		}
		if count <= 1 {
			return apierror.NewLastOwner()
		}
	}

	return s.repo.Delete(ctx, id)
}

// ClaimOwnership creates the first member as owner when the members collection is empty.
// If the collection is not empty, it returns a 403 Forbidden error.
// If a race condition causes a duplicate key error on insert, it also returns 403.
func (s *Service) ClaimOwnership(ctx context.Context, email, displayName string) (*Member, error) {
	count, err := s.repo.CountAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("counting members: %w", err)
	}
	if count > 0 {
		return nil, apierror.NewForbidden("access denied: workspace already has members", "error.accessDenied")
	}

	email = strings.ToLower(strings.TrimSpace(email))
	if displayName == "" {
		displayName = email
	}

	now := time.Now().UTC()
	m := &Member{
		Email:       email,
		Role:        RoleOwner,
		DisplayName: displayName,
		AddedBy:     "system",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, m); err != nil {
		// Race condition: another request created a member between count and insert.
		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) && apiErr.Code == apierror.CodeConflict {
			return nil, apierror.NewForbidden("access denied: workspace already has members", "error.accessDenied")
		}
		return nil, fmt.Errorf("creating owner member: %w", err)
	}

	slog.Info("first user claimed ownership", "email", email)
	return m, nil
}

// TransferOwnership transfers owner role from one member to another.
func (s *Service) TransferOwnership(ctx context.Context, fromID, toID string) error {
	from, err := s.repo.GetByID(ctx, fromID)
	if err != nil {
		return fmt.Errorf("getting source member %s: %w", fromID, err)
	}
	if from.Role != RoleOwner {
		return apierror.NewForbidden("only owners can transfer ownership", "error.notOwner")
	}

	if _, err := s.repo.GetByID(ctx, toID); err != nil {
		return fmt.Errorf("getting target member %s: %w", toID, err)
	}

	return s.repo.TransferOwnership(ctx, fromID, toID)
}

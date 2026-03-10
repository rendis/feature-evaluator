package schedule

import (
	"context"
	"fmt"
	"time"

	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// Service handles scheduled change business logic.
type Service struct {
	repo Repository
}

// NewService creates a new schedule service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new scheduled change after validation.
func (s *Service) Create(ctx context.Context, sc *ScheduledChange) error {
	if sc.FeatureKey == "" {
		return apierror.NewBadRequest("featureKey is required", "error.featureKeyRequired")
	}
	if sc.ScheduledAt.IsZero() {
		return apierror.NewBadRequest("scheduledAt is required", "error.scheduledAtRequired")
	}
	if sc.ScheduledAt.Before(time.Now().UTC()) {
		return apierror.NewBadRequest("scheduledAt must be in the future", "error.scheduledAtPast")
	}
	if sc.ChangeType == "" {
		return apierror.NewBadRequest("changeType is required", "error.changeTypeRequired")
	}
	if !validChangeType(sc.ChangeType) {
		return apierror.NewBadRequest(
			fmt.Sprintf("invalid changeType: %s", sc.ChangeType),
			"error.invalidChangeType",
		)
	}

	sc.Status = StatusPending
	sc.CreatedAt = time.Now().UTC()

	return s.repo.Create(ctx, sc)
}

// GetByID retrieves a scheduled change by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*ScheduledChange, error) {
	return s.repo.GetByID(ctx, id)
}

// Cancel cancels a pending scheduled change by deleting it.
func (s *Service) Cancel(ctx context.Context, id string) error {
	sc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if sc.Status != StatusPending {
		return apierror.NewBadRequest(
			fmt.Sprintf("cannot cancel schedule in %s status", sc.Status),
			"error.scheduleNotCancellable",
		)
	}
	return s.repo.Delete(ctx, id)
}

// ListByFeature returns all scheduled changes for a feature.
func (s *Service) ListByFeature(ctx context.Context, featureKey string) ([]ScheduledChange, error) {
	return s.repo.ListByFeature(ctx, featureKey)
}

func validChangeType(ct ChangeType) bool {
	switch ct {
	case ChangeToggle, ChangeUpdate, ChangeDefaultVal, ChangeEnvironment:
		return true
	default:
		return false
	}
}

package audit

import (
	"context"
	"fmt"
	"time"
)

// Service handles audit/evaluation error logic.
type Service struct {
	repo Repository
}

// NewService creates a new audit service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// LogError records an evaluation error asynchronously (caller should use goroutine).
func (s *Service) LogError(ctx context.Context, evalErr *EvalError) error {
	evalErr.CreatedAt = time.Now().UTC()
	if err := s.repo.Create(ctx, evalErr); err != nil {
		return fmt.Errorf("logging eval error for feature %s: %w", evalErr.FeatureKey, err)
	}
	return nil
}

// List returns a paginated list of evaluation errors.
func (s *Service) List(ctx context.Context, params ListParams) (*ListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}
	return s.repo.List(ctx, params)
}

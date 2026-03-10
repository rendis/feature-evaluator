package schedule

import "context"

// Repository defines persistence for scheduled changes.
type Repository interface {
	Create(ctx context.Context, sc *ScheduledChange) error
	GetByID(ctx context.Context, id string) (*ScheduledChange, error)
	Delete(ctx context.Context, id string) error
	ListByFeature(ctx context.Context, featureKey string) ([]ScheduledChange, error)

	// ClaimNextPending atomically finds one pending change whose scheduledAt <= now
	// and sets its status to "executing". Returns nil, nil if none available.
	ClaimNextPending(ctx context.Context) (*ScheduledChange, error)

	// SetCompleted marks a change as completed.
	SetCompleted(ctx context.Context, id string) error

	// SetFailed marks a change as failed with an error message.
	SetFailed(ctx context.Context, id string, errMsg string) error
}

package experiment

import (
	"context"
	"time"
)

// Cache defines the interface for caching running experiments.
type Cache interface {
	GetRunning(ctx context.Context, workspaceKey, featureKey string) (*Experiment, bool)
	SetRunning(ctx context.Context, workspaceKey, featureKey string, exp *Experiment, ttl time.Duration)
	Invalidate(ctx context.Context, workspaceKey, featureKey string)
}

// Repository defines the persistence interface for experiments.
type Repository interface {
	Create(ctx context.Context, exp *Experiment) error
	GetByID(ctx context.Context, id string) (*Experiment, error)
	Update(ctx context.Context, exp *Experiment) error
	List(ctx context.Context) ([]Experiment, error)
	FindRunningByFeatureKey(ctx context.Context, featureKey string) (*Experiment, error)
}

// ExposureRepository defines the persistence interface for exposures.
type ExposureRepository interface {
	Upsert(ctx context.Context, exp *Exposure) error
	Find(ctx context.Context, experimentID, userID string) (*Exposure, error)
	CountByVariant(ctx context.Context, experimentID string) (map[string]int64, error)
}

// ConversionRepository defines the persistence interface for conversions.
type ConversionRepository interface {
	Create(ctx context.Context, conv *Conversion) error
	CountByVariant(ctx context.Context, experimentID, metricKey string) (map[string]int64, error)
}

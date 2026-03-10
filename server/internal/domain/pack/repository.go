package pack

import "context"

// Repository defines the persistence interface for packs.
type Repository interface {
	Create(ctx context.Context, p *Pack) error
	GetByKey(ctx context.Context, key string) (*Pack, error)
	Update(ctx context.Context, p *Pack) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context) ([]Pack, error)
	Toggle(ctx context.Context, key string, enabled bool, updatedBy string) error
	FindByFeatureKey(ctx context.Context, featureKey string) ([]Pack, error)
	ListEnabled(ctx context.Context) ([]Pack, error)
	ListAllInheritance(ctx context.Context) (map[string][]string, error)
}

// ActivationRepository defines the persistence interface for pack activations.
type ActivationRepository interface {
	Create(ctx context.Context, a *Activation) error
	Delete(ctx context.Context, packKey string, targetType TargetType, targetID string) error
	ListByPack(ctx context.Context, packKey string) ([]Activation, error)
	FindByTarget(ctx context.Context, targetType TargetType, targetID string) ([]Activation, error)
	FindActiveFeatureKeys(ctx context.Context, tenantID, campusID, programID string) ([]string, error)
}

package member

import "context"

// Repository defines the persistence interface for team members.
type Repository interface {
	Create(ctx context.Context, m *Member) error
	GetByID(ctx context.Context, id string) (*Member, error)
	GetByEmail(ctx context.Context, email string) (*Member, error)
	Update(ctx context.Context, m *Member) error
	UpdateRole(ctx context.Context, id string, role Role) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]Member, error)
	CountAll(ctx context.Context) (int64, error)
	CountByRole(ctx context.Context, role Role) (int64, error)
	TransferOwnership(ctx context.Context, fromID, toID string) error
}

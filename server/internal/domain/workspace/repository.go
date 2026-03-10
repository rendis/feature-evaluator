package workspace

import "context"

// Repository defines the persistence interface for workspaces.
type Repository interface {
	Create(ctx context.Context, w *Workspace) error
	GetByKey(ctx context.Context, key string) (*Workspace, error)
	Update(ctx context.Context, w *Workspace) error
	Archive(ctx context.Context, key, archivedBy string) error
	Restore(ctx context.Context, key string) error
	List(ctx context.Context, includeArchived bool) ([]Workspace, error)
	CountActive(ctx context.Context) (int64, error)
}

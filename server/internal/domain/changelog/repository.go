package changelog

import "context"

// ListParams holds filtering/pagination for changelog queries.
type ListParams struct {
	EntityType string
	EntityKey  string
	Actor      string
	Action     string
	From       string
	To         string
	Page       int
	PageSize   int
}

// ListResult holds a paginated list of change entries.
type ListResult struct {
	Data       []ChangeEntry
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

// Repository defines the persistence interface for changelog entries.
type Repository interface {
	Create(ctx context.Context, entry *ChangeEntry) error
	List(ctx context.Context, params ListParams) (*ListResult, error)
	ListByEntity(ctx context.Context, entityType, entityKey string, params ListParams) (*ListResult, error)
}

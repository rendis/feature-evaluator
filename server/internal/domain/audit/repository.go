package audit

import "context"

// ListParams holds parameters for querying evaluation errors.
type ListParams struct {
	FeatureKey string
	TenantID   string
	ErrorType  string
	From       string
	To         string
	Page       int
	PageSize   int
}

// ListResult holds a paginated list of evaluation errors.
type ListResult struct {
	Data       []EvalError
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

// Repository defines the persistence interface for evaluation errors.
type Repository interface {
	Create(ctx context.Context, err *EvalError) error
	List(ctx context.Context, params ListParams) (*ListResult, error)
}

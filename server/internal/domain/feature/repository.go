package feature

import "context"

type ListView string

const (
	ListViewDefault ListView = ""
	ListViewSummary ListView = "summary"
)

// ListParams holds parameters for listing features.
type ListParams struct {
	Search      string
	Enabled     *bool
	ValueType   *ValueType
	Tags        []string
	Environment string
	Page        int
	PageSize    int
	SortBy      string
	SortOrder   string
	View        ListView
}

// ListResult holds a paginated list of features.
type ListResult struct {
	Data       []Feature
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

// Repository defines the persistence interface for features.
type Repository interface {
	Create(ctx context.Context, feature *Feature) error
	GetByKey(ctx context.Context, key string) (*Feature, error)
	Update(ctx context.Context, feature *Feature) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, params ListParams) (*ListResult, error)
	ListEnabled(ctx context.Context) ([]Feature, error)
	Toggle(ctx context.Context, key string, enabled bool, updatedBy string) error

	// Rule operations
	AddRule(ctx context.Context, featureKey string, rule *Rule) error
	UpdateRule(ctx context.Context, featureKey string, rule *Rule) error
	DeleteRule(ctx context.Context, featureKey string, ruleID string) error
	ReorderRules(ctx context.Context, featureKey string, ruleIDs []string) error
}

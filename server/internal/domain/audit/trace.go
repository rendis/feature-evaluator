package audit

import (
	"context"

	"github.com/rendis/feature-evaluator/internal/domain/observability"
)

// EvalTrace is the persisted sanitized evaluation trace payload.
type EvalTrace = observability.EvaluationTrace

// TraceListParams controls listing persisted traces.
type TraceListParams struct {
	FeatureKey  string
	RuleID      string
	Search      string
	CacheStatus string
	UsedRedis   *bool
	From        string
	To          string
	Page        int
	PageSize    int
}

// TraceListResult is a paginated trace list.
type TraceListResult struct {
	Data       []EvalTrace
	Total      int64
	Page       int
	PageSize   int
	TotalPages int
}

// TraceRepository persists sanitized evaluation traces.
type TraceRepository interface {
	Create(ctx context.Context, trace *EvalTrace) error
	List(ctx context.Context, params TraceListParams) (*TraceListResult, error)
}

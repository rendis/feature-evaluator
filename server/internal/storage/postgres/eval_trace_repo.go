package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/audit"
)

// EvalTraceRepo persists evaluation traces in PostgreSQL.
type EvalTraceRepo struct {
	client *Client
}

// NewEvalTraceRepo creates a new trace repository.
func NewEvalTraceRepo(client *Client) *EvalTraceRepo {
	return &EvalTraceRepo{client: client}
}

// Create inserts a sanitized evaluation trace.
func (r *EvalTraceRepo) Create(ctx context.Context, trace *audit.EvalTrace) error {
	if trace == nil {
		return nil
	}
	id, err := newID()
	if err != nil {
		return err
	}
	if trace.CreatedAt.IsZero() {
		trace.CreatedAt = time.Now().UTC()
	}
	trace.ID = id
	trace.FeatureKey = strings.TrimSpace(trace.FeatureKey)
	trace.RequestID = strings.TrimSpace(trace.RequestID)
	payload, err := json.Marshal(trace)
	if err != nil {
		return fmt.Errorf("marshal evaluation trace: %w", err)
	}
	_, err = r.client.db(ctx).Exec(ctx, `
		INSERT INTO evaluation_traces (
			id, workspace_key, feature_key, request_id, rule_id, used_redis,
			cache_status, total_duration_ms, result_reason, error_code, trace, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`,
		id,
		wsKey(ctx),
		trace.FeatureKey,
		trace.RequestID,
		extractMatchedRuleID(trace),
		trace.UsedRedis,
		trace.CacheStatus,
		trace.TotalDurationMs,
		trace.ResultReason,
		trace.ErrorCode,
		payload,
		trace.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert evaluation trace: %w", err)
	}
	return nil
}

// List returns paginated traces for a feature.
func (r *EvalTraceRepo) List(ctx context.Context, params audit.TraceListParams) (*audit.TraceListResult, error) { //nolint:funlen,cyclop // query builder
	pageSize := params.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	page := params.Page
	if page < 1 {
		page = 1
	}
	where := []string{"workspace_key = $1"}
	args := []any{wsKey(ctx)}
	arg := 2

	if params.FeatureKey != "" {
		where = append(where, fmt.Sprintf("feature_key = $%d", arg))
		args = append(args, params.FeatureKey)
		arg++
	}
	if params.RuleID != "" {
		where = append(where, fmt.Sprintf("rule_id = $%d", arg))
		args = append(args, params.RuleID)
		arg++
	}
	if search := sanitizeSearch(params.Search); search != "" {
		where = append(where, fmt.Sprintf("request_id ILIKE $%d", arg))
		args = append(args, "%"+search+"%")
		arg++
	}
	if params.CacheStatus != "" {
		where = append(where, fmt.Sprintf("cache_status = $%d", arg))
		args = append(args, params.CacheStatus)
		arg++
	}
	if params.UsedRedis != nil {
		where = append(where, fmt.Sprintf("used_redis = $%d", arg))
		args = append(args, *params.UsedRedis)
		arg++
	}

	fromPtr, toPtr := parseCreatedAtFilters(params.From, params.To)
	if fromPtr != nil {
		where = append(where, fmt.Sprintf("created_at >= $%d", arg))
		args = append(args, *fromPtr)
		arg++
	}
	if toPtr != nil {
		where = append(where, fmt.Sprintf("created_at <= $%d", arg))
		args = append(args, *toPtr)
		arg++
	}

	predicate := strings.Join(where, " AND ")
	var total int64
	if err := r.client.db(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM evaluation_traces WHERE `+predicate, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count evaluation traces: %w", err)
	}

	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, trace
		FROM evaluation_traces
		WHERE `+predicate+`
		ORDER BY created_at DESC
		LIMIT $`+fmt.Sprint(arg)+` OFFSET $`+fmt.Sprint(arg+1), args...)
	if err != nil {
		return nil, fmt.Errorf("list evaluation traces: %w", err)
	}
	defer rows.Close()

	data := make([]audit.EvalTrace, 0)
	for rows.Next() {
		var id string
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			return nil, fmt.Errorf("scan evaluation trace: %w", err)
		}
		var trace audit.EvalTrace
		if err := json.Unmarshal(raw, &trace); err != nil {
			return nil, fmt.Errorf("unmarshal evaluation trace: %w", err)
		}
		if trace.ID == "" {
			trace.ID = id
		}
		data = append(data, trace)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evaluation traces: %w", err)
	}

	return &audit.TraceListResult{
		Data:       data,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: calcTotalPages(total, pageSize),
	}, nil
}

func extractMatchedRuleID(trace *audit.EvalTrace) string {
	for _, rule := range trace.Rules {
		if rule.Matched {
			return rule.RuleID
		}
	}
	return ""
}

package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/rendis/feature-evaluator/internal/domain/audit"
)

// EvalErrorRepo implements audit.Repository using PostgreSQL.
type EvalErrorRepo struct {
	client *Client
}

// NewEvalErrorRepo creates a new EvalErrorRepo.
func NewEvalErrorRepo(client *Client) *EvalErrorRepo {
	return &EvalErrorRepo{client: client}
}

// Create inserts an evaluation error.
func (r *EvalErrorRepo) Create(ctx context.Context, evalErr *audit.EvalError) error {
	if evalErr.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		evalErr.ID = id
	}
	evalErr.WorkspaceKey = wsKey(ctx)

	_, err := r.client.db(ctx).Exec(ctx, `
		INSERT INTO evaluation_errors (
			id, workspace_key, feature_key, rule_id, error_type, message,
			tenant_id, campus_id, program_id, request_id, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11
		)
	`,
		evalErr.ID,
		evalErr.WorkspaceKey,
		evalErr.FeatureKey,
		evalErr.RuleID,
		evalErr.ErrorType,
		evalErr.Message,
		evalErr.TenantID,
		evalErr.CampusID,
		evalErr.ProgramID,
		evalErr.RequestID,
		evalErr.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert evaluation error: %w", err)
	}

	return nil
}

// List returns paginated evaluation errors.
func (r *EvalErrorRepo) List(ctx context.Context, params audit.ListParams) (*audit.ListResult, error) { //nolint:funlen // builds complex query
	where := []string{"workspace_key = $1"}
	args := []any{wsKey(ctx)}
	arg := 2

	if params.FeatureKey != "" {
		where = append(where, fmt.Sprintf("feature_key = $%d", arg))
		args = append(args, params.FeatureKey)
		arg++
	}
	if params.TenantID != "" {
		where = append(where, fmt.Sprintf("tenant_id = $%d", arg))
		args = append(args, params.TenantID)
		arg++
	}
	if params.ErrorType != "" {
		where = append(where, fmt.Sprintf("error_type = $%d", arg))
		args = append(args, params.ErrorType)
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
	if err := r.client.db(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM evaluation_errors WHERE `+predicate, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count evaluation errors: %w", err)
	}

	args = append(args, params.PageSize, (params.Page-1)*params.PageSize)
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, workspace_key, feature_key, rule_id, error_type, message,
		       tenant_id, campus_id, program_id, request_id, created_at
		FROM evaluation_errors
		WHERE `+predicate+`
		ORDER BY created_at DESC
		LIMIT $`+fmt.Sprint(arg)+` OFFSET $`+fmt.Sprint(arg+1), args...)
	if err != nil {
		return nil, fmt.Errorf("list evaluation errors: %w", err)
	}
	defer rows.Close()

	errors := make([]audit.EvalError, 0)
	for rows.Next() {
		var item audit.EvalError
		if err := rows.Scan(
			&item.ID,
			&item.WorkspaceKey,
			&item.FeatureKey,
			&item.RuleID,
			&item.ErrorType,
			&item.Message,
			&item.TenantID,
			&item.CampusID,
			&item.ProgramID,
			&item.RequestID,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan evaluation error: %w", err)
		}
		errors = append(errors, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evaluation errors: %w", err)
	}

	return &audit.ListResult{
		Data:       errors,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: calcTotalPages(total, params.PageSize),
	}, nil
}

package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rendis/feature-evaluator/internal/domain/experiment"
)

// ConversionRepo implements experiment.ConversionRepository using PostgreSQL.
type ConversionRepo struct {
	client *Client
}

// NewConversionRepo creates a new ConversionRepo.
func NewConversionRepo(client *Client) *ConversionRepo {
	return &ConversionRepo{client: client}
}

// Create inserts a conversion row.
func (r *ConversionRepo) Create(ctx context.Context, conv *experiment.Conversion) error {
	if conv.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		conv.ID = id
	}
	conv.WorkspaceKey = wsKey(ctx)

	experimentID, err := uuid.Parse(conv.ExperimentID)
	if err != nil {
		return fmt.Errorf("invalid experiment id: %w", err)
	}

	_, err = r.client.db(ctx).Exec(ctx, `
		INSERT INTO experiment_conversions (
			id, workspace_key, experiment_id, user_id, variant_key, metric_key, value, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
	`,
		conv.ID,
		conv.WorkspaceKey,
		experimentID,
		conv.UserID,
		conv.VariantKey,
		conv.MetricKey,
		conv.Value,
		conv.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert conversion: %w", err)
	}

	return nil
}

// CountByVariant aggregates conversions by variant for a metric.
func (r *ConversionRepo) CountByVariant(ctx context.Context, experimentID, metricKey string) (map[string]int64, error) {
	parsed, err := uuid.Parse(experimentID)
	if err != nil {
		return map[string]int64{}, nil
	}

	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT variant_key, COUNT(*)
		FROM experiment_conversions
		WHERE workspace_key = $1 AND experiment_id = $2 AND metric_key = $3
		GROUP BY variant_key
	`, wsKey(ctx), parsed, metricKey)
	if err != nil {
		return nil, fmt.Errorf("count conversions by variant: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var key string
		var count int64
		if err := rows.Scan(&key, &count); err != nil {
			return nil, fmt.Errorf("scan conversion aggregate: %w", err)
		}
		result[key] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversion aggregates: %w", err)
	}

	return result, nil
}

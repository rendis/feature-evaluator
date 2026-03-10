package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rendis/feature-evaluator/internal/domain/experiment"
)

// ExposureRepo implements experiment.ExposureRepository using PostgreSQL.
type ExposureRepo struct {
	client *Client
}

// NewExposureRepo creates a new ExposureRepo.
func NewExposureRepo(client *Client) *ExposureRepo {
	return &ExposureRepo{client: client}
}

// Upsert inserts or updates an exposure.
func (r *ExposureRepo) Upsert(ctx context.Context, exp *experiment.Exposure) error {
	if exp.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		exp.ID = id
	}
	exp.WorkspaceKey = wsKey(ctx)

	experimentID, err := uuid.Parse(exp.ExperimentID)
	if err != nil {
		return fmt.Errorf("invalid experiment id: %w", err)
	}
	featureID, err := r.featureIDByKey(ctx, exp.FeatureKey)
	if err != nil {
		return err
	}

	_, err = r.client.db(ctx).Exec(ctx, `
		INSERT INTO experiment_exposures (
			id, workspace_key, experiment_id, feature_id, user_id, variant_key, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
		ON CONFLICT (experiment_id, user_id)
		DO UPDATE SET variant_key = EXCLUDED.variant_key, feature_id = EXCLUDED.feature_id
	`,
		exp.ID,
		exp.WorkspaceKey,
		experimentID,
		featureID,
		exp.UserID,
		exp.VariantKey,
		exp.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert exposure: %w", err)
	}

	return nil
}

// Find finds an exposure by experiment and user.
func (r *ExposureRepo) Find(ctx context.Context, experimentID, userID string) (*experiment.Exposure, error) {
	parsed, err := uuid.Parse(experimentID)
	if err != nil {
		return nil, nil
	}

	row := r.client.db(ctx).QueryRow(ctx, `
		SELECT ee.id, ee.workspace_key, ee.experiment_id::text, f.key, ee.user_id, ee.variant_key, ee.created_at
		FROM experiment_exposures ee
		JOIN features f ON f.id = ee.feature_id
		WHERE ee.workspace_key = $1 AND ee.experiment_id = $2 AND ee.user_id = $3
	`, wsKey(ctx), parsed, userID)

	var exp experiment.Exposure
	if err := row.Scan(&exp.ID, &exp.WorkspaceKey, &exp.ExperimentID, &exp.FeatureKey, &exp.UserID, &exp.VariantKey, &exp.CreatedAt); err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("find exposure: %w", err)
	}

	return &exp, nil
}

// CountByVariant aggregates exposures by variant.
func (r *ExposureRepo) CountByVariant(ctx context.Context, experimentID string) (map[string]int64, error) {
	parsed, err := uuid.Parse(experimentID)
	if err != nil {
		return map[string]int64{}, nil
	}

	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT variant_key, COUNT(*)
		FROM experiment_exposures
		WHERE workspace_key = $1 AND experiment_id = $2
		GROUP BY variant_key
	`, wsKey(ctx), parsed)
	if err != nil {
		return nil, fmt.Errorf("count exposures by variant: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var key string
		var count int64
		if err := rows.Scan(&key, &count); err != nil {
			return nil, fmt.Errorf("scan exposure aggregate: %w", err)
		}
		result[key] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate exposure aggregates: %w", err)
	}

	return result, nil
}

func (r *ExposureRepo) featureIDByKey(ctx context.Context, featureKey string) (uuid.UUID, error) {
	var featureID uuid.UUID
	if err := r.client.db(ctx).QueryRow(ctx, `
		SELECT id
		FROM features
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), featureKey).Scan(&featureID); err != nil {
		return uuid.Nil, fmt.Errorf("find feature id for exposure: %w", err)
	}

	return featureID, nil
}

package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rendis/feature-evaluator/internal/domain/experiment"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// ExperimentRepo implements experiment.Repository using PostgreSQL.
type ExperimentRepo struct {
	client *Client
}

// NewExperimentRepo creates a new ExperimentRepo.
func NewExperimentRepo(client *Client) *ExperimentRepo {
	return &ExperimentRepo{client: client}
}

// Create inserts a new experiment.
func (r *ExperimentRepo) Create(ctx context.Context, exp *experiment.Experiment) error {
	if exp.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		exp.ID = id
	}
	exp.WorkspaceKey = wsKey(ctx)

	featureID, err := r.featureIDByKey(ctx, exp.FeatureKey)
	if err != nil {
		return err
	}
	variantsJSON, err := jsonBytes(exp.Variants, "[]")
	if err != nil {
		return fmt.Errorf("marshal experiment variants: %w", err)
	}
	metricsJSON, err := jsonBytes(exp.Metrics, "[]")
	if err != nil {
		return fmt.Errorf("marshal experiment metrics: %w", err)
	}

	_, err = r.client.db(ctx).Exec(ctx, `
		INSERT INTO experiments (
			id, workspace_key, feature_id, name, description, variants, metrics, status,
			winner_key, started_at, completed_at, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6::jsonb, $7::jsonb, $8,
			$9, $10, $11, $12, $13, $14
		)
	`,
		exp.ID,
		exp.WorkspaceKey,
		featureID,
		exp.Name,
		exp.Description,
		variantsJSON,
		metricsJSON,
		exp.Status,
		exp.WinnerKey,
		exp.StartedAt,
		exp.CompletedAt,
		exp.CreatedBy,
		exp.CreatedAt,
		exp.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert experiment: %w", err)
	}

	return nil
}

// GetByID finds an experiment by ID.
func (r *ExperimentRepo) GetByID(ctx context.Context, id string) (*experiment.Experiment, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, apierror.NewNotFound(
			fmt.Sprintf("experiment %q not found", id),
			"error.experimentNotFound",
		)
	}

	row := r.client.db(ctx).QueryRow(ctx, `
		SELECT e.id, e.workspace_key, f.key, e.name, e.description, e.variants, e.metrics, e.status,
		       e.winner_key, e.started_at, e.completed_at, e.created_by, e.created_at, e.updated_at
		FROM experiments e
		JOIN features f ON f.id = e.feature_id
		WHERE e.workspace_key = $1 AND e.id = $2
	`, wsKey(ctx), parsed)

	exp, err := scanExperiment(row)
	if err != nil {
		if isNoRows(err) {
			return nil, apierror.NewNotFound(
				fmt.Sprintf("experiment %q not found", id),
				"error.experimentNotFound",
			)
		}
		return nil, fmt.Errorf("find experiment: %w", err)
	}

	return exp, nil
}

// Update updates an experiment.
func (r *ExperimentRepo) Update(ctx context.Context, exp *experiment.Experiment) error {
	parsed, err := uuid.Parse(exp.ID)
	if err != nil {
		return fmt.Errorf("invalid experiment ID: %w", err)
	}

	variantsJSON, err := jsonBytes(exp.Variants, "[]")
	if err != nil {
		return fmt.Errorf("marshal experiment variants: %w", err)
	}
	metricsJSON, err := jsonBytes(exp.Metrics, "[]")
	if err != nil {
		return fmt.Errorf("marshal experiment metrics: %w", err)
	}

	tag, err := r.client.db(ctx).Exec(ctx, `
		UPDATE experiments
		SET name = $3, description = $4, variants = $5::jsonb, metrics = $6::jsonb,
		    status = $7, winner_key = $8, started_at = $9, completed_at = $10, updated_at = $11
		WHERE workspace_key = $1 AND id = $2
	`,
		wsKey(ctx),
		parsed,
		exp.Name,
		exp.Description,
		variantsJSON,
		metricsJSON,
		exp.Status,
		exp.WinnerKey,
		exp.StartedAt,
		exp.CompletedAt,
		exp.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update experiment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound(
			fmt.Sprintf("experiment %q not found", exp.ID),
			"error.experimentNotFound",
		)
	}

	return nil
}

// List returns all experiments.
func (r *ExperimentRepo) List(ctx context.Context) ([]experiment.Experiment, error) {
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT e.id, e.workspace_key, f.key, e.name, e.description, e.variants, e.metrics, e.status,
		       e.winner_key, e.started_at, e.completed_at, e.created_by, e.created_at, e.updated_at
		FROM experiments e
		JOIN features f ON f.id = e.feature_id
		WHERE e.workspace_key = $1
		ORDER BY e.updated_at DESC
	`, wsKey(ctx))
	if err != nil {
		return nil, fmt.Errorf("list experiments: %w", err)
	}
	defer rows.Close()

	experiments := make([]experiment.Experiment, 0)
	for rows.Next() {
		exp, err := scanExperiment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan experiment: %w", err)
		}
		experiments = append(experiments, *exp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate experiments: %w", err)
	}

	if experiments == nil {
		return []experiment.Experiment{}, nil
	}
	return experiments, nil
}

// FindRunningByFeatureKey finds the running experiment for a feature.
func (r *ExperimentRepo) FindRunningByFeatureKey(ctx context.Context, featureKey string) (*experiment.Experiment, error) {
	row := r.client.db(ctx).QueryRow(ctx, `
		SELECT e.id, e.workspace_key, f.key, e.name, e.description, e.variants, e.metrics, e.status,
		       e.winner_key, e.started_at, e.completed_at, e.created_by, e.created_at, e.updated_at
		FROM experiments e
		JOIN features f ON f.id = e.feature_id
		WHERE e.workspace_key = $1 AND f.key = $2 AND e.status = $3
	`, wsKey(ctx), featureKey, experiment.StatusRunning)

	exp, err := scanExperiment(row)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("find running experiment: %w", err)
	}

	return exp, nil
}

func (r *ExperimentRepo) featureIDByKey(ctx context.Context, featureKey string) (uuid.UUID, error) {
	var featureID uuid.UUID
	if err := r.client.db(ctx).QueryRow(ctx, `
		SELECT id
		FROM features
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), featureKey).Scan(&featureID); err != nil {
		if isNoRows(err) {
			return uuid.Nil, apierror.NewNotFound(
				fmt.Sprintf("feature %q not found", featureKey),
				"error.featureNotFound",
			)
		}
		return uuid.Nil, fmt.Errorf("find experiment feature id: %w", err)
	}

	return featureID, nil
}

type experimentScanner interface {
	Scan(dest ...any) error
}

func scanExperiment(scanner experimentScanner) (*experiment.Experiment, error) {
	var exp experiment.Experiment
	var variantsJSON []byte
	var metricsJSON []byte
	if err := scanner.Scan(
		&exp.ID,
		&exp.WorkspaceKey,
		&exp.FeatureKey,
		&exp.Name,
		&exp.Description,
		&variantsJSON,
		&metricsJSON,
		&exp.Status,
		&exp.WinnerKey,
		&exp.StartedAt,
		&exp.CompletedAt,
		&exp.CreatedBy,
		&exp.CreatedAt,
		&exp.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if err := decodeJSON(variantsJSON, &exp.Variants); err != nil {
		return nil, err
	}
	if err := decodeJSON(metricsJSON, &exp.Metrics); err != nil {
		return nil, err
	}

	return &exp, nil
}

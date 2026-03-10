package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rendis/feature-evaluator/internal/domain/schedule"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// ScheduleRepo implements schedule.Repository using PostgreSQL.
type ScheduleRepo struct {
	client *Client
}

// NewScheduleRepo creates a new ScheduleRepo.
func NewScheduleRepo(client *Client) *ScheduleRepo {
	return &ScheduleRepo{client: client}
}

// Create inserts a scheduled change.
func (r *ScheduleRepo) Create(ctx context.Context, sc *schedule.ScheduledChange) error {
	if sc.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		sc.ID = id
	}
	sc.WorkspaceKey = wsKey(ctx)

	payloadJSON, err := jsonBytes(sc.Payload, "{}")
	if err != nil {
		return fmt.Errorf("marshal schedule payload: %w", err)
	}
	featureID, err := r.featureIDByKey(ctx, sc.FeatureKey)
	if err != nil {
		return err
	}

	_, err = r.client.db(ctx).Exec(ctx, `
		INSERT INTO schedules (
			id, workspace_key, feature_id, change_type, payload, scheduled_at,
			status, error, executed_at, created_by, created_at
		) VALUES (
			$1, $2, $3, $4, $5::jsonb, $6,
			$7, $8, $9, $10, $11
		)
	`,
		sc.ID,
		sc.WorkspaceKey,
		featureID,
		sc.ChangeType,
		payloadJSON,
		sc.ScheduledAt,
		sc.Status,
		sc.Error,
		sc.ExecutedAt,
		sc.CreatedBy,
		sc.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert schedule: %w", err)
	}

	return nil
}

// GetByID finds a scheduled change by ID.
func (r *ScheduleRepo) GetByID(ctx context.Context, id string) (*schedule.ScheduledChange, error) {
	parsed, err := parseUUID(id)
	if err != nil {
		return nil, apierror.NewBadRequest("invalid schedule ID", "error.invalidId")
	}

	row := r.client.db(ctx).QueryRow(ctx, `
		SELECT s.id, s.workspace_key, f.key, s.change_type, s.payload, s.scheduled_at,
		       s.status, s.error, s.executed_at, s.created_by, s.created_at
		FROM schedules s
		JOIN features f ON f.id = s.feature_id
		WHERE s.workspace_key = $1 AND s.id = $2
	`, wsKey(ctx), parsed)

	sc, err := scanSchedule(row)
	if err != nil {
		if isNoRows(err) {
			return nil, apierror.NewNotFound("scheduled change not found", "error.scheduleNotFound")
		}
		return nil, fmt.Errorf("find schedule: %w", err)
	}

	return sc, nil
}

// Delete removes a scheduled change.
func (r *ScheduleRepo) Delete(ctx context.Context, id string) error {
	parsed, err := parseUUID(id)
	if err != nil {
		return apierror.NewBadRequest("invalid schedule ID", "error.invalidId")
	}

	tag, err := r.client.db(ctx).Exec(ctx, `
		DELETE FROM schedules
		WHERE workspace_key = $1 AND id = $2
	`, wsKey(ctx), parsed)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound("scheduled change not found", "error.scheduleNotFound")
	}

	return nil
}

// ListByFeature lists scheduled changes for a feature.
func (r *ScheduleRepo) ListByFeature(ctx context.Context, featureKey string) ([]schedule.ScheduledChange, error) {
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT s.id, s.workspace_key, f.key, s.change_type, s.payload, s.scheduled_at,
		       s.status, s.error, s.executed_at, s.created_by, s.created_at
		FROM schedules s
		JOIN features f ON f.id = s.feature_id
		WHERE s.workspace_key = $1 AND f.key = $2
		ORDER BY s.scheduled_at ASC
	`, wsKey(ctx), featureKey)
	if err != nil {
		return nil, fmt.Errorf("list schedules by feature: %w", err)
	}
	defer rows.Close()

	schedules := make([]schedule.ScheduledChange, 0)
	for rows.Next() {
		sc, err := scanSchedule(rows)
		if err != nil {
			return nil, fmt.Errorf("scan schedule: %w", err)
		}
		schedules = append(schedules, *sc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schedules: %w", err)
	}

	if schedules == nil {
		return []schedule.ScheduledChange{}, nil
	}
	return schedules, nil
}

// ClaimNextPending atomically claims the next pending schedule across all workspaces.
func (r *ScheduleRepo) ClaimNextPending(ctx context.Context) (*schedule.ScheduledChange, error) {
	var claimed *schedule.ScheduledChange
	err := r.client.WithinTx(ctx, func(txCtx context.Context) error {
		row := r.client.db(txCtx).QueryRow(txCtx, `
			WITH next_schedule AS (
				SELECT id
				FROM schedules
				WHERE status = $1 AND scheduled_at <= $2
				ORDER BY scheduled_at ASC
				FOR UPDATE SKIP LOCKED
				LIMIT 1
			)
			UPDATE schedules s
			SET status = $3
			FROM next_schedule ns
			WHERE s.id = ns.id
			RETURNING s.id
		`, schedule.StatusPending, time.Now().UTC(), schedule.StatusExecuting)

		var scheduleID uuid.UUID
		if err := row.Scan(&scheduleID); err != nil {
			if isNoRows(err) {
				return nil
			}
			return fmt.Errorf("claim schedule: %w", err)
		}

		item, err := r.getByUUID(txCtx, scheduleID, false)
		if err != nil {
			return err
		}
		claimed = item
		return nil
	})
	if err != nil {
		return nil, err
	}

	return claimed, nil
}

// SetCompleted marks a schedule as completed.
func (r *ScheduleRepo) SetCompleted(ctx context.Context, id string) error {
	parsed, err := parseUUID(id)
	if err != nil {
		return fmt.Errorf("invalid schedule ID: %w", err)
	}

	_, err = r.client.db(ctx).Exec(ctx, `
		UPDATE schedules
		SET status = $2, executed_at = $3
		WHERE id = $1
	`, parsed, schedule.StatusCompleted, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("complete schedule: %w", err)
	}

	return nil
}

// SetFailed marks a schedule as failed.
func (r *ScheduleRepo) SetFailed(ctx context.Context, id string, errMsg string) error {
	parsed, err := parseUUID(id)
	if err != nil {
		return fmt.Errorf("invalid schedule ID: %w", err)
	}

	_, err = r.client.db(ctx).Exec(ctx, `
		UPDATE schedules
		SET status = $2, error = $3, executed_at = $4
		WHERE id = $1
	`, parsed, schedule.StatusFailed, errMsg, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("fail schedule: %w", err)
	}

	return nil
}

func (r *ScheduleRepo) featureIDByKey(ctx context.Context, featureKey string) (uuid.UUID, error) {
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
		return uuid.Nil, fmt.Errorf("find feature id for schedule: %w", err)
	}

	return featureID, nil
}

func (r *ScheduleRepo) getByUUID(ctx context.Context, id uuid.UUID, scoped bool) (*schedule.ScheduledChange, error) {
	query := `
		SELECT s.id, s.workspace_key, f.key, s.change_type, s.payload, s.scheduled_at,
		       s.status, s.error, s.executed_at, s.created_by, s.created_at
		FROM schedules s
		JOIN features f ON f.id = s.feature_id
		WHERE s.id = $1
	`
	args := []any{id}
	if scoped {
		query += ` AND s.workspace_key = $2`
		args = append(args, wsKey(ctx))
	}

	row := r.client.db(ctx).QueryRow(ctx, query, args...)
	sc, err := scanSchedule(row)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("find schedule by uuid: %w", err)
	}

	return sc, nil
}

type scheduleScanner interface {
	Scan(dest ...any) error
}

func scanSchedule(scanner scheduleScanner) (*schedule.ScheduledChange, error) {
	var sc schedule.ScheduledChange
	var payloadJSON []byte
	if err := scanner.Scan(
		&sc.ID,
		&sc.WorkspaceKey,
		&sc.FeatureKey,
		&sc.ChangeType,
		&payloadJSON,
		&sc.ScheduledAt,
		&sc.Status,
		&sc.Error,
		&sc.ExecutedAt,
		&sc.CreatedBy,
		&sc.CreatedAt,
	); err != nil {
		return nil, err
	}
	if err := decodeJSON(payloadJSON, &sc.Payload); err != nil {
		return nil, err
	}

	return &sc, nil
}

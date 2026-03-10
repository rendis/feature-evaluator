package postgres

import (
	"context"
	"fmt"

	"github.com/rendis/feature-evaluator/internal/domain/workspace"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// WorkspaceRepo implements workspace.Repository using PostgreSQL.
type WorkspaceRepo struct {
	client *Client
}

// NewWorkspaceRepo creates a new WorkspaceRepo.
func NewWorkspaceRepo(client *Client) *WorkspaceRepo {
	return &WorkspaceRepo{client: client}
}

// Create inserts a new workspace row.
func (r *WorkspaceRepo) Create(ctx context.Context, w *workspace.Workspace) error {
	if w.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		w.ID = id
	}

	metadata, err := jsonBytes(w.Metadata, "{}")
	if err != nil {
		return fmt.Errorf("marshal workspace metadata: %w", err)
	}

	_, err = r.client.db(ctx).Exec(ctx, `
		INSERT INTO workspaces (
			id, key, name, description, metadata, created_at, updated_at, created_by, archived_at, archived_by
		) VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9, $10)
	`,
		w.ID,
		w.Key,
		w.Name,
		w.Description,
		metadata,
		w.CreatedAt,
		w.UpdatedAt,
		w.CreatedBy,
		w.ArchivedAt,
		w.ArchivedBy,
	)
	if isUniqueViolation(err) {
		return apierror.NewConflict(
			fmt.Sprintf("workspace with key %q already exists", w.Key),
			"error.workspaceKeyExists",
		)
	}
	if err != nil {
		return fmt.Errorf("insert workspace: %w", err)
	}

	return nil
}

// GetByKey finds a workspace by key.
func (r *WorkspaceRepo) GetByKey(ctx context.Context, key string) (*workspace.Workspace, error) {
	row := r.client.db(ctx).QueryRow(ctx, `
		SELECT id, key, name, description, metadata, created_at, updated_at, created_by, archived_at, archived_by
		FROM workspaces
		WHERE key = $1
	`, key)

	var w workspace.Workspace
	var metadata []byte
	if err := row.Scan(
		&w.ID,
		&w.Key,
		&w.Name,
		&w.Description,
		&metadata,
		&w.CreatedAt,
		&w.UpdatedAt,
		&w.CreatedBy,
		&w.ArchivedAt,
		&w.ArchivedBy,
	); err != nil {
		if isNoRows(err) {
			return nil, apierror.NewNotFound(
				fmt.Sprintf("workspace %q not found", key),
				"error.workspaceNotFound",
			)
		}
		return nil, fmt.Errorf("find workspace: %w", err)
	}
	if err := decodeJSON(metadata, &w.Metadata); err != nil {
		return nil, fmt.Errorf("decode workspace metadata: %w", err)
	}

	return &w, nil
}

// Update updates mutable workspace fields.
func (r *WorkspaceRepo) Update(ctx context.Context, w *workspace.Workspace) error {
	metadata, err := jsonBytes(w.Metadata, "{}")
	if err != nil {
		return fmt.Errorf("marshal workspace metadata: %w", err)
	}

	tag, err := r.client.db(ctx).Exec(ctx, `
		UPDATE workspaces
		SET name = $2, description = $3, metadata = $4::jsonb, updated_at = $5
		WHERE key = $1
	`, w.Key, w.Name, w.Description, metadata, w.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update workspace: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound(
			fmt.Sprintf("workspace %q not found", w.Key),
			"error.workspaceNotFound",
		)
	}

	return nil
}

// Archive marks a workspace as archived.
func (r *WorkspaceRepo) Archive(ctx context.Context, key, archivedBy string) error {
	tag, err := r.client.db(ctx).Exec(ctx, `
		UPDATE workspaces
		SET archived_at = NOW(), archived_by = $2, updated_at = NOW()
		WHERE key = $1 AND archived_at IS NULL
	`, key, archivedBy)
	if err != nil {
		return fmt.Errorf("archive workspace: %w", err)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}

	existing, err := r.GetByKey(ctx, key)
	if err != nil {
		return err
	}
	if existing.IsArchived() {
		return apierror.NewConflict(
			fmt.Sprintf("workspace %q is already archived", key),
			"error.workspaceAlreadyArchived",
		)
	}

	return nil
}

// Restore marks an archived workspace as active.
func (r *WorkspaceRepo) Restore(ctx context.Context, key string) error {
	tag, err := r.client.db(ctx).Exec(ctx, `
		UPDATE workspaces
		SET archived_at = NULL, archived_by = '', updated_at = NOW()
		WHERE key = $1 AND archived_at IS NOT NULL
	`, key)
	if err != nil {
		return fmt.Errorf("restore workspace: %w", err)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}

	existing, err := r.GetByKey(ctx, key)
	if err != nil {
		return err
	}
	if !existing.IsArchived() {
		return apierror.NewConflict(
			fmt.Sprintf("workspace %q is already active", key),
			"error.workspaceAlreadyActive",
		)
	}

	return nil
}

// List returns all workspaces.
func (r *WorkspaceRepo) List(ctx context.Context, includeArchived bool) ([]workspace.Workspace, error) {
	query := `
		SELECT id, key, name, description, metadata, created_at, updated_at, created_by, archived_at, archived_by
		FROM workspaces
	`
	args := []any{}
	if !includeArchived {
		query += ` WHERE archived_at IS NULL`
	}
	query += ` ORDER BY key ASC`

	rows, err := r.client.db(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()

	workspaces := make([]workspace.Workspace, 0)
	for rows.Next() {
		var w workspace.Workspace
		var metadata []byte
		if err := rows.Scan(
			&w.ID,
			&w.Key,
			&w.Name,
			&w.Description,
			&metadata,
			&w.CreatedAt,
			&w.UpdatedAt,
			&w.CreatedBy,
			&w.ArchivedAt,
			&w.ArchivedBy,
		); err != nil {
			return nil, fmt.Errorf("scan workspace: %w", err)
		}
		if err := decodeJSON(metadata, &w.Metadata); err != nil {
			return nil, fmt.Errorf("decode workspace metadata: %w", err)
		}
		workspaces = append(workspaces, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspaces: %w", err)
	}

	if workspaces == nil {
		return []workspace.Workspace{}, nil
	}
	return workspaces, nil
}

// CountActive counts active workspaces.
func (r *WorkspaceRepo) CountActive(ctx context.Context) (int64, error) {
	var count int64
	if err := r.client.db(ctx).QueryRow(ctx, `
		SELECT COUNT(*)
		FROM workspaces
		WHERE archived_at IS NULL
	`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count active workspaces: %w", err)
	}

	return count, nil
}

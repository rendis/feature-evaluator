package postgres

import (
	"context"
	"fmt"

	"github.com/rendis/feature-evaluator/internal/domain/tag"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// TagRepo implements tag.Repository using PostgreSQL.
type TagRepo struct {
	client *Client
}

// NewTagRepo creates a new TagRepo.
func NewTagRepo(client *Client) *TagRepo {
	return &TagRepo{client: client}
}

// Create inserts a new tag.
func (r *TagRepo) Create(ctx context.Context, t *tag.Tag) error {
	if t.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		t.ID = id
	}
	t.WorkspaceKey = wsKey(ctx)

	_, err := r.client.db(ctx).Exec(ctx, `
		INSERT INTO tags (
			id, workspace_key, key, name, color, created_at, updated_at, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, t.ID, t.WorkspaceKey, t.Key, t.Name, t.Color, t.CreatedAt, t.UpdatedAt, t.CreatedBy)
	if isUniqueViolation(err) {
		return apierror.NewConflict(
			fmt.Sprintf("tag with key %q or name %q already exists", t.Key, t.Name),
			"error.tagExists",
		)
	}
	if err != nil {
		return fmt.Errorf("insert tag: %w", err)
	}

	return nil
}

// Update updates a tag by key.
func (r *TagRepo) Update(ctx context.Context, t *tag.Tag) error {
	tagResult, err := r.client.db(ctx).Exec(ctx, `
		UPDATE tags
		SET name = $3, color = $4, updated_at = $5
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), t.Key, t.Name, t.Color, t.UpdatedAt)
	if isUniqueViolation(err) {
		return apierror.NewConflict(
			fmt.Sprintf("tag with name %q already exists", t.Name),
			"error.tagNameExists",
		)
	}
	if err != nil {
		return fmt.Errorf("update tag: %w", err)
	}
	if tagResult.RowsAffected() == 0 {
		return apierror.NewNotFound("tag not found", "error.tagNotFound")
	}

	return nil
}

// Delete removes a tag by key.
func (r *TagRepo) Delete(ctx context.Context, key string) error {
	tagResult, err := r.client.db(ctx).Exec(ctx, `
		DELETE FROM tags
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), key)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	if tagResult.RowsAffected() == 0 {
		return apierror.NewNotFound("tag not found", "error.tagNotFound")
	}

	return nil
}

// FindByKey finds a tag by key.
func (r *TagRepo) FindByKey(ctx context.Context, key string) (*tag.Tag, error) {
	row := r.client.db(ctx).QueryRow(ctx, `
		SELECT id, workspace_key, key, name, color, created_at, updated_at, created_by
		FROM tags
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), key)

	var t tag.Tag
	if err := row.Scan(
		&t.ID,
		&t.WorkspaceKey,
		&t.Key,
		&t.Name,
		&t.Color,
		&t.CreatedAt,
		&t.UpdatedAt,
		&t.CreatedBy,
	); err != nil {
		if isNoRows(err) {
			return nil, apierror.NewNotFound(
				fmt.Sprintf("tag %q not found", key),
				"error.tagNotFound",
			)
		}
		return nil, fmt.Errorf("find tag: %w", err)
	}

	return &t, nil
}

// FindByKeys finds tags by keys.
func (r *TagRepo) FindByKeys(ctx context.Context, keys []string) ([]tag.Tag, error) {
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, workspace_key, key, name, color, created_at, updated_at, created_by
		FROM tags
		WHERE workspace_key = $1 AND key = ANY($2)
		ORDER BY name ASC
	`, wsKey(ctx), keys)
	if err != nil {
		return nil, fmt.Errorf("find tags by keys: %w", err)
	}
	defer rows.Close()

	tags := make([]tag.Tag, 0)
	for rows.Next() {
		var t tag.Tag
		if err := rows.Scan(
			&t.ID,
			&t.WorkspaceKey,
			&t.Key,
			&t.Name,
			&t.Color,
			&t.CreatedAt,
			&t.UpdatedAt,
			&t.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}

	if tags == nil {
		return []tag.Tag{}, nil
	}
	return tags, nil
}

// List returns all tags with optional search.
func (r *TagRepo) List(ctx context.Context, search string) ([]tag.Tag, error) {
	search = sanitizeSearch(search, 200)
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, workspace_key, key, name, color, created_at, updated_at, created_by
		FROM tags
		WHERE workspace_key = $1
		  AND ($2 = '' OR key ILIKE '%' || $2 || '%' OR name ILIKE '%' || $2 || '%')
		ORDER BY name ASC
	`, wsKey(ctx), search)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	tags := make([]tag.Tag, 0)
	for rows.Next() {
		var t tag.Tag
		if err := rows.Scan(
			&t.ID,
			&t.WorkspaceKey,
			&t.Key,
			&t.Name,
			&t.Color,
			&t.CreatedAt,
			&t.UpdatedAt,
			&t.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}

	if tags == nil {
		return []tag.Tag{}, nil
	}
	return tags, nil
}

// CountFeaturesByTag counts how many features reference the tag.
func (r *TagRepo) CountFeaturesByTag(ctx context.Context, tagKey string) (int64, error) {
	var count int64
	if err := r.client.db(ctx).QueryRow(ctx, `
		SELECT COUNT(*)
		FROM feature_tags ft
		JOIN features f ON f.id = ft.feature_id
		WHERE f.workspace_key = $1 AND ft.tag_key = $2
	`, wsKey(ctx), tagKey).Scan(&count); err != nil {
		return 0, fmt.Errorf("count features by tag: %w", err)
	}

	return count, nil
}

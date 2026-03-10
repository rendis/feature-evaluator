package postgres

import (
	"context"
	"fmt"

	"github.com/rendis/feature-evaluator/internal/domain/tier"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// TierRepo implements tier.Repository using PostgreSQL.
type TierRepo struct {
	client *Client
}

// NewTierRepo creates a new TierRepo.
func NewTierRepo(client *Client) *TierRepo {
	return &TierRepo{client: client}
}

// Create inserts a new tier.
func (r *TierRepo) Create(ctx context.Context, t *tier.Tier) error {
	if t.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		t.ID = id
	}
	t.WorkspaceKey = wsKey(ctx)

	_, err := r.client.db(ctx).Exec(ctx, `
		INSERT INTO tiers (
			id, workspace_key, key, name, level, color, icon, created_at, updated_at, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, t.ID, t.WorkspaceKey, t.Key, t.Name, t.Level, t.Color, t.Icon, t.CreatedAt, t.UpdatedAt, t.CreatedBy)
	if isUniqueViolation(err) {
		return apierror.NewConflict(
			"tier key already exists",
			"error.tierAlreadyExists",
		)
	}
	if err != nil {
		return fmt.Errorf("insert tier: %w", err)
	}

	return nil
}

// Update updates a tier by key.
func (r *TierRepo) Update(ctx context.Context, t *tier.Tier) error {
	tierResult, err := r.client.db(ctx).Exec(ctx, `
		UPDATE tiers
		SET name = $3, level = $4, color = $5, icon = $6, updated_at = $7
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), t.Key, t.Name, t.Level, t.Color, t.Icon, t.UpdatedAt)
	if isUniqueViolation(err) {
		return apierror.NewConflict(
			fmt.Sprintf("tier with name %q already exists", t.Name),
			"error.tierNameExists",
		)
	}
	if err != nil {
		return fmt.Errorf("update tier: %w", err)
	}
	if tierResult.RowsAffected() == 0 {
		return apierror.NewNotFound("tier not found", "error.tierNotFound")
	}

	return nil
}

// Delete removes a tier by key.
func (r *TierRepo) Delete(ctx context.Context, key string) error {
	tierResult, err := r.client.db(ctx).Exec(ctx, `
		DELETE FROM tiers
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), key)
	if err != nil {
		return fmt.Errorf("delete tier: %w", err)
	}
	if tierResult.RowsAffected() == 0 {
		return apierror.NewNotFound("tier not found", "error.tierNotFound")
	}

	return nil
}

// FindByKey finds a tier by key.
func (r *TierRepo) FindByKey(ctx context.Context, key string) (*tier.Tier, error) {
	row := r.client.db(ctx).QueryRow(ctx, `
		SELECT id, workspace_key, key, name, level, color, icon, created_at, updated_at, created_by
		FROM tiers
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), key)

	var t tier.Tier
	if err := row.Scan(
		&t.ID,
		&t.WorkspaceKey,
		&t.Key,
		&t.Name,
		&t.Level,
		&t.Color,
		&t.Icon,
		&t.CreatedAt,
		&t.UpdatedAt,
		&t.CreatedBy,
	); err != nil {
		if isNoRows(err) {
			return nil, apierror.NewNotFound(
				fmt.Sprintf("tier %q not found", key),
				"error.tierNotFound",
			)
		}
		return nil, fmt.Errorf("find tier: %w", err)
	}

	return &t, nil
}

// FindByKeys finds tiers by keys.
func (r *TierRepo) FindByKeys(ctx context.Context, keys []string) ([]tier.Tier, error) {
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, workspace_key, key, name, level, color, icon, created_at, updated_at, created_by
		FROM tiers
		WHERE workspace_key = $1 AND key = ANY($2)
		ORDER BY level ASC
	`, wsKey(ctx), keys)
	if err != nil {
		return nil, fmt.Errorf("find tiers by keys: %w", err)
	}
	defer rows.Close()

	tiers := make([]tier.Tier, 0)
	for rows.Next() {
		var t tier.Tier
		if err := rows.Scan(
			&t.ID,
			&t.WorkspaceKey,
			&t.Key,
			&t.Name,
			&t.Level,
			&t.Color,
			&t.Icon,
			&t.CreatedAt,
			&t.UpdatedAt,
			&t.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan tier: %w", err)
		}
		tiers = append(tiers, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tiers: %w", err)
	}

	if tiers == nil {
		return []tier.Tier{}, nil
	}
	return tiers, nil
}

// List returns all tiers with optional search.
func (r *TierRepo) List(ctx context.Context, search string) ([]tier.Tier, error) {
	search = sanitizeSearch(search, 200)
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, workspace_key, key, name, level, color, icon, created_at, updated_at, created_by
		FROM tiers
		WHERE workspace_key = $1
		  AND ($2 = '' OR key ILIKE '%' || $2 || '%' OR name ILIKE '%' || $2 || '%')
		ORDER BY level ASC
	`, wsKey(ctx), search)
	if err != nil {
		return nil, fmt.Errorf("list tiers: %w", err)
	}
	defer rows.Close()

	tiers := make([]tier.Tier, 0)
	for rows.Next() {
		var t tier.Tier
		if err := rows.Scan(
			&t.ID,
			&t.WorkspaceKey,
			&t.Key,
			&t.Name,
			&t.Level,
			&t.Color,
			&t.Icon,
			&t.CreatedAt,
			&t.UpdatedAt,
			&t.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan tier: %w", err)
		}
		tiers = append(tiers, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tiers: %w", err)
	}

	if tiers == nil {
		return []tier.Tier{}, nil
	}
	return tiers, nil
}

// CountPacksByTier counts how many packs reference the tier.
func (r *TierRepo) CountPacksByTier(ctx context.Context, tierKey string) (int64, error) {
	var count int64
	if err := r.client.db(ctx).QueryRow(ctx, `
		SELECT COUNT(*)
		FROM packs
		WHERE workspace_key = $1 AND tier_key = $2
	`, wsKey(ctx), tierKey).Scan(&count); err != nil {
		return 0, fmt.Errorf("count packs by tier: %w", err)
	}

	return count, nil
}

// TierIconRepo implements tier.IconRepository using PostgreSQL.
type TierIconRepo struct {
	client *Client
}

// NewTierIconRepo creates a new TierIconRepo.
func NewTierIconRepo(client *Client) *TierIconRepo {
	return &TierIconRepo{client: client}
}

// CreateIcon inserts a new custom tier icon.
func (r *TierIconRepo) CreateIcon(ctx context.Context, icon *tier.TierIcon) error {
	if icon.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		icon.ID = id
	}
	icon.WorkspaceKey = wsKey(ctx)

	_, err := r.client.db(ctx).Exec(ctx, `
		INSERT INTO tier_icons (
			id, workspace_key, name, content_type, data, created_at, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, icon.ID, icon.WorkspaceKey, icon.Name, icon.ContentType, icon.Data, icon.CreatedAt, icon.CreatedBy)
	if isUniqueViolation(err) {
		return apierror.NewConflict(
			fmt.Sprintf("icon with name %q already exists", icon.Name),
			"error.iconNameExists",
		)
	}
	if err != nil {
		return fmt.Errorf("insert tier icon: %w", err)
	}

	return nil
}

// DeleteIcon removes a custom tier icon by ID.
func (r *TierIconRepo) DeleteIcon(ctx context.Context, id string) error {
	iconResult, err := r.client.db(ctx).Exec(ctx, `
		DELETE FROM tier_icons
		WHERE id = $1 AND workspace_key = $2
	`, id, wsKey(ctx))
	if err != nil {
		return fmt.Errorf("delete tier icon: %w", err)
	}
	if iconResult.RowsAffected() == 0 {
		return apierror.NewNotFound("icon not found", "error.iconNotFound")
	}

	return nil
}

// FindIconByID finds a custom tier icon by ID.
func (r *TierIconRepo) FindIconByID(ctx context.Context, id string) (*tier.TierIcon, error) {
	row := r.client.db(ctx).QueryRow(ctx, `
		SELECT id, workspace_key, name, content_type, data, created_at, created_by
		FROM tier_icons
		WHERE id = $1 AND workspace_key = $2
	`, id, wsKey(ctx))

	var icon tier.TierIcon
	if err := row.Scan(
		&icon.ID,
		&icon.WorkspaceKey,
		&icon.Name,
		&icon.ContentType,
		&icon.Data,
		&icon.CreatedAt,
		&icon.CreatedBy,
	); err != nil {
		if isNoRows(err) {
			return nil, apierror.NewNotFound(
				fmt.Sprintf("icon %q not found", id),
				"error.iconNotFound",
			)
		}
		return nil, fmt.Errorf("find tier icon: %w", err)
	}

	return &icon, nil
}

// ListIcons returns all custom tier icons for the workspace.
func (r *TierIconRepo) ListIcons(ctx context.Context) ([]tier.TierIcon, error) {
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, workspace_key, name, content_type, data, created_at, created_by
		FROM tier_icons
		WHERE workspace_key = $1
		ORDER BY name ASC
	`, wsKey(ctx))
	if err != nil {
		return nil, fmt.Errorf("list tier icons: %w", err)
	}
	defer rows.Close()

	icons := make([]tier.TierIcon, 0)
	for rows.Next() {
		var icon tier.TierIcon
		if err := rows.Scan(
			&icon.ID,
			&icon.WorkspaceKey,
			&icon.Name,
			&icon.ContentType,
			&icon.Data,
			&icon.CreatedAt,
			&icon.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan tier icon: %w", err)
		}
		icons = append(icons, icon)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tier icons: %w", err)
	}

	if icons == nil {
		return []tier.TierIcon{}, nil
	}
	return icons, nil
}

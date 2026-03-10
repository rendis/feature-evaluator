package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rendis/feature-evaluator/internal/domain/pack"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// PackRepo implements pack.Repository using PostgreSQL.
type PackRepo struct {
	client *Client
}

// NewPackRepo creates a new PackRepo.
func NewPackRepo(client *Client) *PackRepo {
	return &PackRepo{client: client}
}

// Create inserts a new pack and its feature mappings.
func (r *PackRepo) Create(ctx context.Context, p *pack.Pack) error {
	if p.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		p.ID = id
	}
	p.WorkspaceKey = wsKey(ctx)

	metadataJSON, err := jsonBytes(p.Metadata, "{}")
	if err != nil {
		return fmt.Errorf("marshal pack metadata: %w", err)
	}

	return r.client.WithinTx(ctx, func(txCtx context.Context) error {
		_, err := r.client.db(txCtx).Exec(txCtx, `
			INSERT INTO packs (
				id, workspace_key, key, name, description, enabled, metadata,
				created_at, updated_at, created_by, updated_by, tier_key, trial_until
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7::jsonb,
				$8, $9, $10, $11, $12, $13
			)
		`,
			p.ID,
			p.WorkspaceKey,
			p.Key,
			p.Name,
			p.Description,
			p.Enabled,
			metadataJSON,
			p.CreatedAt,
			p.UpdatedAt,
			p.CreatedBy,
			p.UpdatedBy,
			p.TierKey,
			p.TrialUntil,
		)
		if isUniqueViolation(err) {
			return apierror.NewConflict(
				fmt.Sprintf("pack with key %q already exists", p.Key),
				"error.packKeyExists",
			)
		}
		if err != nil {
			return fmt.Errorf("insert pack: %w", err)
		}

		if err := r.replacePackFeatures(txCtx, p.ID, p.WorkspaceKey, p.FeatureKeys); err != nil {
			return err
		}

		return r.replacePackInheritance(txCtx, p.ID, p.WorkspaceKey, p.InheritsFrom)
	})
}

// GetByKey finds a pack by key.
func (r *PackRepo) GetByKey(ctx context.Context, key string) (*pack.Pack, error) {
	row := r.client.db(ctx).QueryRow(ctx, `
		SELECT id, workspace_key, key, name, description, enabled, metadata,
		       created_at, updated_at, created_by, updated_by, tier_key, trial_until
		FROM packs
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), key)

	p, err := scanPack(row)
	if err != nil {
		if isNoRows(err) {
			return nil, apierror.NewNotFound(
				fmt.Sprintf("pack %q not found", key),
				"error.packNotFound",
			)
		}
		return nil, fmt.Errorf("find pack: %w", err)
	}

	features, err := r.loadPackFeatureKeys(ctx, []string{p.ID})
	if err != nil {
		return nil, err
	}
	p.FeatureKeys = features[p.ID]

	inheritance, err := r.loadPackInheritance(ctx, []string{p.ID})
	if err != nil {
		return nil, err
	}
	p.InheritsFrom = inheritance[p.ID]

	return p, nil
}

// Update updates a pack and replaces its feature mappings.
func (r *PackRepo) Update(ctx context.Context, p *pack.Pack) error {
	metadataJSON, err := jsonBytes(p.Metadata, "{}")
	if err != nil {
		return fmt.Errorf("marshal pack metadata: %w", err)
	}

	return r.client.WithinTx(ctx, func(txCtx context.Context) error {
		var packID string
		tag, err := r.client.db(txCtx).Exec(txCtx, `
			UPDATE packs
			SET name = $3, description = $4, enabled = $5, metadata = $6::jsonb,
			    updated_at = $7, updated_by = $8, tier_key = $9, trial_until = $10
			WHERE workspace_key = $1 AND key = $2
		`, wsKey(txCtx), p.Key, p.Name, p.Description, p.Enabled, metadataJSON, p.UpdatedAt, p.UpdatedBy, p.TierKey, p.TrialUntil)
		if err != nil {
			return fmt.Errorf("update pack: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return apierror.NewNotFound(
				fmt.Sprintf("pack %q not found", p.Key),
				"error.packNotFound",
			)
		}

		if err := r.client.db(txCtx).QueryRow(txCtx, `
			SELECT id
			FROM packs
			WHERE workspace_key = $1 AND key = $2
		`, wsKey(txCtx), p.Key).Scan(&packID); err != nil {
			return fmt.Errorf("find updated pack id: %w", err)
		}

		p.ID = packID

		if err := r.replacePackFeatures(txCtx, packID, wsKey(txCtx), p.FeatureKeys); err != nil {
			return err
		}

		return r.replacePackInheritance(txCtx, packID, wsKey(txCtx), p.InheritsFrom)
	})
}

// Delete removes a pack by key.
func (r *PackRepo) Delete(ctx context.Context, key string) error {
	tag, err := r.client.db(ctx).Exec(ctx, `
		DELETE FROM packs
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), key)
	if err != nil {
		return fmt.Errorf("delete pack: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound(
			fmt.Sprintf("pack %q not found", key),
			"error.packNotFound",
		)
	}

	return nil
}

// List returns all packs for the workspace.
func (r *PackRepo) List(ctx context.Context) ([]pack.Pack, error) {
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, workspace_key, key, name, description, enabled, metadata,
		       created_at, updated_at, created_by, updated_by, tier_key, trial_until
		FROM packs
		WHERE workspace_key = $1
		ORDER BY updated_at DESC
	`, wsKey(ctx))
	if err != nil {
		return nil, fmt.Errorf("list packs: %w", err)
	}
	defer rows.Close()

	packs := make([]pack.Pack, 0)
	packIDs := make([]string, 0)
	for rows.Next() {
		item, err := scanPack(rows)
		if err != nil {
			return nil, fmt.Errorf("scan pack: %w", err)
		}
		packs = append(packs, *item)
		packIDs = append(packIDs, item.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate packs: %w", err)
	}

	featuresByPack, err := r.loadPackFeatureKeys(ctx, packIDs)
	if err != nil {
		return nil, err
	}
	for i := range packs {
		packs[i].FeatureKeys = featuresByPack[packs[i].ID]
	}

	inheritanceByPack, err := r.loadPackInheritance(ctx, packIDs)
	if err != nil {
		return nil, err
	}
	for i := range packs {
		packs[i].InheritsFrom = inheritanceByPack[packs[i].ID]
	}

	if packs == nil {
		return []pack.Pack{}, nil
	}
	return packs, nil
}

// Toggle enables or disables a pack.
func (r *PackRepo) Toggle(ctx context.Context, key string, enabled bool, updatedBy string) error {
	tag, err := r.client.db(ctx).Exec(ctx, `
		UPDATE packs
		SET enabled = $3, updated_by = $4, updated_at = NOW()
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), key, enabled, updatedBy)
	if err != nil {
		return fmt.Errorf("toggle pack: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound(
			fmt.Sprintf("pack %q not found", key),
			"error.packNotFound",
		)
	}

	return nil
}

// FindByFeatureKey finds packs containing the feature.
func (r *PackRepo) FindByFeatureKey(ctx context.Context, featureKey string) ([]pack.Pack, error) {
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT p.id, p.workspace_key, p.key, p.name, p.description, p.enabled, p.metadata,
		       p.created_at, p.updated_at, p.created_by, p.updated_by, p.tier_key, p.trial_until
		FROM packs p
		JOIN pack_features pf ON pf.pack_id = p.id
		JOIN features f ON f.id = pf.feature_id
		WHERE p.workspace_key = $1 AND f.key = $2
		ORDER BY p.updated_at DESC
	`, wsKey(ctx), featureKey)
	if err != nil {
		return nil, fmt.Errorf("find packs by feature key: %w", err)
	}
	defer rows.Close()

	packs := make([]pack.Pack, 0)
	packIDs := make([]string, 0)
	for rows.Next() {
		item, err := scanPack(rows)
		if err != nil {
			return nil, fmt.Errorf("scan pack: %w", err)
		}
		packs = append(packs, *item)
		packIDs = append(packIDs, item.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate packs by feature key: %w", err)
	}

	featuresByPack, err := r.loadPackFeatureKeys(ctx, packIDs)
	if err != nil {
		return nil, err
	}
	for i := range packs {
		packs[i].FeatureKeys = featuresByPack[packs[i].ID]
	}

	inheritanceByPack, err := r.loadPackInheritance(ctx, packIDs)
	if err != nil {
		return nil, err
	}
	for i := range packs {
		packs[i].InheritsFrom = inheritanceByPack[packs[i].ID]
	}

	if packs == nil {
		return []pack.Pack{}, nil
	}
	return packs, nil
}

// ListEnabled returns enabled packs in the workspace.
func (r *PackRepo) ListEnabled(ctx context.Context) ([]pack.Pack, error) {
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, workspace_key, key, name, description, enabled, metadata,
		       created_at, updated_at, created_by, updated_by, tier_key, trial_until
		FROM packs
		WHERE workspace_key = $1 AND enabled = TRUE
		ORDER BY updated_at DESC
	`, wsKey(ctx))
	if err != nil {
		return nil, fmt.Errorf("list enabled packs: %w", err)
	}
	defer rows.Close()

	packs := make([]pack.Pack, 0)
	packIDs := make([]string, 0)
	for rows.Next() {
		item, err := scanPack(rows)
		if err != nil {
			return nil, fmt.Errorf("scan enabled pack: %w", err)
		}
		packs = append(packs, *item)
		packIDs = append(packIDs, item.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate enabled packs: %w", err)
	}

	featuresByPack, err := r.loadPackFeatureKeys(ctx, packIDs)
	if err != nil {
		return nil, err
	}
	for i := range packs {
		packs[i].FeatureKeys = featuresByPack[packs[i].ID]
	}

	inheritanceByPack, err := r.loadPackInheritance(ctx, packIDs)
	if err != nil {
		return nil, err
	}
	for i := range packs {
		packs[i].InheritsFrom = inheritanceByPack[packs[i].ID]
	}

	if packs == nil {
		return []pack.Pack{}, nil
	}
	return packs, nil
}

func (r *PackRepo) replacePackFeatures(ctx context.Context, packID string, workspaceKey string, featureKeys []string) error {
	if _, err := r.client.db(ctx).Exec(ctx, `DELETE FROM pack_features WHERE pack_id = $1`, packID); err != nil {
		return fmt.Errorf("delete existing pack features: %w", err)
	}
	if len(featureKeys) == 0 {
		return nil
	}

	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, key
		FROM features
		WHERE workspace_key = $1 AND key = ANY($2)
	`, workspaceKey, featureKeys)
	if err != nil {
		return fmt.Errorf("load features for pack: %w", err)
	}
	defer rows.Close()

	featureIDs := make(map[string]uuid.UUID, len(featureKeys))
	for rows.Next() {
		var id uuid.UUID
		var key string
		if err := rows.Scan(&id, &key); err != nil {
			return fmt.Errorf("scan feature for pack: %w", err)
		}
		featureIDs[key] = id
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate features for pack: %w", err)
	}

	for idx, key := range featureKeys {
		id, ok := featureIDs[key]
		if !ok {
			return apierror.NewBadRequest(
				fmt.Sprintf("feature %q does not exist", key),
				"error.featureNotFound",
			)
		}

		if _, err := r.client.db(ctx).Exec(ctx, `
			INSERT INTO pack_features (pack_id, feature_id, position)
			VALUES ($1, $2, $3)
		`, packID, id, idx); err != nil {
			return fmt.Errorf("insert pack feature: %w", err)
		}
	}

	return nil
}

func (r *PackRepo) loadPackFeatureKeys(ctx context.Context, packIDs []string) (map[string][]string, error) {
	result := make(map[string][]string, len(packIDs))
	if len(packIDs) == 0 {
		return result, nil
	}

	ids, err := parseUUIDStrings(packIDs)
	if err != nil {
		return nil, fmt.Errorf("parse pack ids for features: %w", err)
	}

	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT p.id, f.key
		FROM packs p
		LEFT JOIN pack_features pf ON pf.pack_id = p.id
		LEFT JOIN features f ON f.id = pf.feature_id
		WHERE p.id = ANY($1)
		ORDER BY pf.position ASC, f.key ASC
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("load pack features: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var packID string
		var featureKey *string
		if err := rows.Scan(&packID, &featureKey); err != nil {
			return nil, fmt.Errorf("scan pack feature: %w", err)
		}
		if _, ok := result[packID]; !ok {
			result[packID] = []string{}
		}
		if featureKey != nil {
			result[packID] = append(result[packID], *featureKey)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pack features: %w", err)
	}

	for _, id := range packIDs {
		if _, ok := result[id]; !ok {
			result[id] = []string{}
		}
	}
	return result, nil
}

func (r *PackRepo) replacePackInheritance(ctx context.Context, packID, workspaceKey string, parentKeys []string) error {
	if _, err := r.client.db(ctx).Exec(ctx, `DELETE FROM pack_inheritance WHERE pack_id = $1`, packID); err != nil {
		return fmt.Errorf("delete pack inheritance: %w", err)
	}
	if len(parentKeys) == 0 {
		return nil
	}

	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, key
		FROM packs
		WHERE workspace_key = $1 AND key = ANY($2)
	`, workspaceKey, parentKeys)
	if err != nil {
		return fmt.Errorf("load parent packs for inheritance: %w", err)
	}
	defer rows.Close()

	parentIDs := make(map[string]string, len(parentKeys))
	for rows.Next() {
		var id, key string
		if err := rows.Scan(&id, &key); err != nil {
			return fmt.Errorf("scan parent pack for inheritance: %w", err)
		}
		parentIDs[key] = id
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate parent packs for inheritance: %w", err)
	}

	for idx, key := range parentKeys {
		id, ok := parentIDs[key]
		if !ok {
			return apierror.NewBadRequest(
				fmt.Sprintf("parent pack %q does not exist", key),
				"error.packNotFound",
			)
		}

		if _, err := r.client.db(ctx).Exec(ctx, `
			INSERT INTO pack_inheritance (pack_id, parent_pack_id, position)
			VALUES ($1, $2, $3)
		`, packID, id, idx); err != nil {
			return fmt.Errorf("insert pack inheritance: %w", err)
		}
	}

	return nil
}

func (r *PackRepo) loadPackInheritance(ctx context.Context, packIDs []string) (map[string][]string, error) {
	result := make(map[string][]string, len(packIDs))
	if len(packIDs) == 0 {
		return result, nil
	}

	ids, err := parseUUIDStrings(packIDs)
	if err != nil {
		return nil, fmt.Errorf("parse pack ids for inheritance: %w", err)
	}

	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT pi.pack_id, pp.key
		FROM pack_inheritance pi
		JOIN packs pp ON pp.id = pi.parent_pack_id
		WHERE pi.pack_id = ANY($1)
		ORDER BY pi.position ASC
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("load pack inheritance: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var packID, parentKey string
		if err := rows.Scan(&packID, &parentKey); err != nil {
			return nil, fmt.Errorf("scan pack inheritance: %w", err)
		}
		result[packID] = append(result[packID], parentKey)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pack inheritance: %w", err)
	}

	for _, id := range packIDs {
		if _, ok := result[id]; !ok {
			result[id] = []string{}
		}
	}
	return result, nil
}

// ListAllInheritance returns packKey -> parentKeys for all packs in the workspace.
func (r *PackRepo) ListAllInheritance(ctx context.Context) (map[string][]string, error) {
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT p.key, pp.key
		FROM pack_inheritance pi
		JOIN packs p ON p.id = pi.pack_id
		JOIN packs pp ON pp.id = pi.parent_pack_id
		WHERE p.workspace_key = $1
		ORDER BY pi.position ASC
	`, wsKey(ctx))
	if err != nil {
		return nil, fmt.Errorf("list all pack inheritance: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var packKey, parentKey string
		if err := rows.Scan(&packKey, &parentKey); err != nil {
			return nil, fmt.Errorf("scan all pack inheritance: %w", err)
		}
		result[packKey] = append(result[packKey], parentKey)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate all pack inheritance: %w", err)
	}

	return result, nil
}

type packScanner interface {
	Scan(dest ...any) error
}

func scanPack(scanner packScanner) (*pack.Pack, error) {
	var p pack.Pack
	var metadataJSON []byte
	if err := scanner.Scan(
		&p.ID,
		&p.WorkspaceKey,
		&p.Key,
		&p.Name,
		&p.Description,
		&p.Enabled,
		&metadataJSON,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.CreatedBy,
		&p.UpdatedBy,
		&p.TierKey,
		&p.TrialUntil,
	); err != nil {
		return nil, err
	}
	if err := decodeJSON(metadataJSON, &p.Metadata); err != nil {
		return nil, err
	}

	return &p, nil
}

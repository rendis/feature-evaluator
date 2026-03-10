package postgres

import (
	"context"
	"fmt"

	"github.com/rendis/feature-evaluator/internal/domain/pack"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// PackActivationRepo implements pack.ActivationRepository using PostgreSQL.
type PackActivationRepo struct {
	client *Client
}

// NewPackActivationRepo creates a new PackActivationRepo.
func NewPackActivationRepo(client *Client) *PackActivationRepo {
	return &PackActivationRepo{client: client}
}

// Create inserts a pack activation row.
func (r *PackActivationRepo) Create(ctx context.Context, a *pack.Activation) error {
	if a.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		a.ID = id
	}
	a.WorkspaceKey = wsKey(ctx)

	metadataJSON, err := jsonBytes(a.Metadata, "{}")
	if err != nil {
		return fmt.Errorf("marshal pack activation metadata: %w", err)
	}

	var packID string
	if err := r.client.db(ctx).QueryRow(ctx, `
		SELECT id
		FROM packs
		WHERE workspace_key = $1 AND key = $2
	`, a.WorkspaceKey, a.PackKey).Scan(&packID); err != nil {
		if isNoRows(err) {
			return apierror.NewNotFound("pack not found", "error.packNotFound")
		}
		return fmt.Errorf("find pack for activation: %w", err)
	}

	_, err = r.client.db(ctx).Exec(ctx, `
		INSERT INTO pack_activations (
			id, workspace_key, pack_id, target_type, target_id, activated_at, activated_by, expires_at, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb
		)
	`,
		a.ID,
		a.WorkspaceKey,
		packID,
		a.TargetType,
		a.TargetID,
		a.ActivatedAt,
		a.ActivatedBy,
		a.ExpiresAt,
		metadataJSON,
	)
	if isUniqueViolation(err) {
		return apierror.NewConflict(
			fmt.Sprintf("pack %q is already activated for %s %q", a.PackKey, a.TargetType, a.TargetID),
			"error.packActivationExists",
		)
	}
	if err != nil {
		return fmt.Errorf("insert pack activation: %w", err)
	}

	return nil
}

// Delete removes a pack activation by pack key and target.
func (r *PackActivationRepo) Delete(ctx context.Context, packKey string, targetType pack.TargetType, targetID string) error {
	tag, err := r.client.db(ctx).Exec(ctx, `
		DELETE FROM pack_activations pa
		USING packs p
		WHERE pa.pack_id = p.id
		  AND pa.workspace_key = $1
		  AND p.workspace_key = $1
		  AND p.key = $2
		  AND pa.target_type = $3
		  AND pa.target_id = $4
	`, wsKey(ctx), packKey, targetType, targetID)
	if err != nil {
		return fmt.Errorf("delete pack activation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound("pack activation not found", "error.packActivationNotFound")
	}

	return nil
}

// ListByPack returns activations for a pack key.
func (r *PackActivationRepo) ListByPack(ctx context.Context, packKey string) ([]pack.Activation, error) {
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT pa.id, pa.workspace_key, p.key, pa.target_type, pa.target_id,
		       pa.activated_at, pa.activated_by, pa.expires_at, pa.metadata
		FROM pack_activations pa
		JOIN packs p ON p.id = pa.pack_id
		WHERE pa.workspace_key = $1 AND p.key = $2
		ORDER BY pa.activated_at DESC
	`, wsKey(ctx), packKey)
	if err != nil {
		return nil, fmt.Errorf("list pack activations: %w", err)
	}
	defer rows.Close()

	return scanActivations(rows)
}

// FindByTarget returns activations for a target.
func (r *PackActivationRepo) FindByTarget(ctx context.Context, targetType pack.TargetType, targetID string) ([]pack.Activation, error) {
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT pa.id, pa.workspace_key, p.key, pa.target_type, pa.target_id,
		       pa.activated_at, pa.activated_by, pa.expires_at, pa.metadata
		FROM pack_activations pa
		JOIN packs p ON p.id = pa.pack_id
		WHERE pa.workspace_key = $1 AND pa.target_type = $2 AND pa.target_id = $3
		ORDER BY pa.activated_at DESC
	`, wsKey(ctx), targetType, targetID)
	if err != nil {
		return nil, fmt.Errorf("find pack activations by target: %w", err)
	}
	defer rows.Close()

	return scanActivations(rows)
}

// FindActiveFeatureKeys returns feature keys granted by active pack activations.
func (r *PackActivationRepo) FindActiveFeatureKeys(ctx context.Context, tenantID, campusID, programID string) ([]string, error) {
	targetTypes := make([]string, 0, 3)
	targetIDs := make([]string, 0, 3)
	if tenantID != "" {
		targetTypes = append(targetTypes, string(pack.TargetTenant))
		targetIDs = append(targetIDs, tenantID)
	}
	if campusID != "" {
		targetTypes = append(targetTypes, string(pack.TargetCampus))
		targetIDs = append(targetIDs, campusID)
	}
	if programID != "" {
		targetTypes = append(targetTypes, string(pack.TargetProgram))
		targetIDs = append(targetIDs, programID)
	}
	if len(targetTypes) == 0 {
		return []string{}, nil
	}

	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT DISTINCT f.key
		FROM pack_activations pa
		JOIN packs p ON p.id = pa.pack_id
		JOIN pack_features pf ON pf.pack_id = p.id
		JOIN features f ON f.id = pf.feature_id
		WHERE pa.workspace_key = $1
		  AND p.workspace_key = $1
		  AND f.workspace_key = $1
		  AND p.enabled = TRUE
		  AND (
			(pa.target_type = 'tenant' AND pa.target_id = $2) OR
			(pa.target_type = 'campus' AND pa.target_id = $3) OR
			(pa.target_type = 'program' AND pa.target_id = $4)
		  )
		  AND (pa.expires_at IS NULL OR pa.expires_at > NOW())
		ORDER BY f.key ASC
	`, wsKey(ctx), tenantID, campusID, programID)
	if err != nil {
		return nil, fmt.Errorf("find active feature keys: %w", err)
	}
	defer rows.Close()

	keys := make([]string, 0)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan active feature key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active feature keys: %w", err)
	}

	if keys == nil {
		return []string{}, nil
	}
	return keys, nil
}

type activationScanner interface {
	Scan(dest ...any) error
}

func scanActivations(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]pack.Activation, error) {
	activations := make([]pack.Activation, 0)
	for rows.Next() {
		var activation pack.Activation
		var metadataJSON []byte
		if err := rows.Scan(
			&activation.ID,
			&activation.WorkspaceKey,
			&activation.PackKey,
			&activation.TargetType,
			&activation.TargetID,
			&activation.ActivatedAt,
			&activation.ActivatedBy,
			&activation.ExpiresAt,
			&metadataJSON,
		); err != nil {
			return nil, fmt.Errorf("scan pack activation: %w", err)
		}
		if err := decodeJSON(metadataJSON, &activation.Metadata); err != nil {
			return nil, fmt.Errorf("decode pack activation metadata: %w", err)
		}
		activations = append(activations, activation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pack activations: %w", err)
	}

	if activations == nil {
		return []pack.Activation{}, nil
	}
	return activations, nil
}

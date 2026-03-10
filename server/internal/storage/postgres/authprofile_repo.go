package postgres

import (
	"context"
	"fmt"

	"github.com/rendis/feature-evaluator/internal/domain/authprofile"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// AuthProfileRepo implements authprofile.Repository using PostgreSQL.
type AuthProfileRepo struct {
	client *Client
}

// NewAuthProfileRepo creates a new AuthProfileRepo.
func NewAuthProfileRepo(client *Client) *AuthProfileRepo {
	return &AuthProfileRepo{client: client}
}

// Create inserts a new auth profile.
func (r *AuthProfileRepo) Create(ctx context.Context, profile *authprofile.Profile) error {
	if profile.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		profile.ID = id
	}
	profile.WorkspaceKey = wsKey(ctx)

	configJSON, err := jsonBytes(profile.Config, "{}")
	if err != nil {
		return fmt.Errorf("marshal auth profile config: %w", err)
	}

	_, err = r.client.db(ctx).Exec(ctx, `
		INSERT INTO auth_profiles (
			id, workspace_key, key, name, active, type, config, cache_ttl_seconds,
			version, secret_payload_encrypted, has_secret, created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7::jsonb, $8,
			$9, $10, $11, $12, $13, $14, $15
		)
	`,
		profile.ID,
		profile.WorkspaceKey,
		profile.Key,
		profile.Name,
		profile.Active,
		profile.Type,
		configJSON,
		profile.CacheTTLSeconds,
		profile.Version,
		profile.SecretPayloadEncrypted,
		profile.HasSecret,
		profile.CreatedAt,
		profile.UpdatedAt,
		profile.CreatedBy,
		profile.UpdatedBy,
	)
	if isUniqueViolation(err) {
		return apierror.NewConflict(
			fmt.Sprintf("auth profile with key %q already exists", profile.Key),
			"error.authProfileExists",
		)
	}
	if err != nil {
		return fmt.Errorf("insert auth profile: %w", err)
	}

	return nil
}

// GetByKey finds an auth profile by key.
func (r *AuthProfileRepo) GetByKey(ctx context.Context, key string) (*authprofile.Profile, error) {
	row := r.client.db(ctx).QueryRow(ctx, `
		SELECT id, workspace_key, key, name, active, type, config, cache_ttl_seconds,
		       version, secret_payload_encrypted, has_secret, created_at, updated_at, created_by, updated_by
		FROM auth_profiles
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), key)

	profile, err := scanAuthProfile(row)
	if err != nil {
		if isNoRows(err) {
			return nil, apierror.NewNotFound(
				fmt.Sprintf("auth profile %q not found", key),
				"error.authProfileNotFound",
			)
		}
		return nil, fmt.Errorf("find auth profile: %w", err)
	}

	return profile, nil
}

// Update updates an existing auth profile.
func (r *AuthProfileRepo) Update(ctx context.Context, currentKey string, profile *authprofile.Profile) error {
	configJSON, err := jsonBytes(profile.Config, "{}")
	if err != nil {
		return fmt.Errorf("marshal auth profile config: %w", err)
	}

	tag, err := r.client.db(ctx).Exec(ctx, `
		UPDATE auth_profiles
		SET key = $3, name = $4, active = $5, type = $6, config = $7::jsonb,
		    cache_ttl_seconds = $8, version = $9, secret_payload_encrypted = $10,
		    has_secret = $11, updated_at = $12, updated_by = $13
		WHERE workspace_key = $1 AND key = $2
	`,
		wsKey(ctx),
		currentKey,
		profile.Key,
		profile.Name,
		profile.Active,
		profile.Type,
		configJSON,
		profile.CacheTTLSeconds,
		profile.Version,
		profile.SecretPayloadEncrypted,
		profile.HasSecret,
		profile.UpdatedAt,
		profile.UpdatedBy,
	)
	if isUniqueViolation(err) {
		return apierror.NewConflict(
			fmt.Sprintf("auth profile with key %q already exists", profile.Key),
			"error.authProfileExists",
		)
	}
	if err != nil {
		return fmt.Errorf("update auth profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound("auth profile not found", "error.authProfileNotFound")
	}

	return nil
}

// Delete removes an auth profile by key.
func (r *AuthProfileRepo) Delete(ctx context.Context, key string) error {
	tag, err := r.client.db(ctx).Exec(ctx, `
		DELETE FROM auth_profiles
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), key)
	if err != nil {
		return fmt.Errorf("delete auth profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound("auth profile not found", "error.authProfileNotFound")
	}

	return nil
}

// List returns all auth profiles for the workspace.
func (r *AuthProfileRepo) List(ctx context.Context) ([]authprofile.Profile, error) {
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, workspace_key, key, name, active, type, config, cache_ttl_seconds,
		       version, secret_payload_encrypted, has_secret, created_at, updated_at, created_by, updated_by
		FROM auth_profiles
		WHERE workspace_key = $1
		ORDER BY updated_at DESC
	`, wsKey(ctx))
	if err != nil {
		return nil, fmt.Errorf("list auth profiles: %w", err)
	}
	defer rows.Close()

	profiles := make([]authprofile.Profile, 0)
	for rows.Next() {
		profile, err := scanAuthProfile(rows)
		if err != nil {
			return nil, fmt.Errorf("scan auth profile: %w", err)
		}
		profiles = append(profiles, *profile)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate auth profiles: %w", err)
	}

	if profiles == nil {
		return []authprofile.Profile{}, nil
	}
	return profiles, nil
}

// CountFeatureUsages counts features referencing an auth profile key.
func (r *AuthProfileRepo) CountFeatureUsages(ctx context.Context, key string) (int64, error) {
	var count int64
	if err := r.client.db(ctx).QueryRow(ctx, `
		SELECT COUNT(*)
		FROM features f
		JOIN auth_profiles ap ON ap.id = f.auth_profile_id
		WHERE ap.workspace_key = $1 AND ap.key = $2
	`, wsKey(ctx), key).Scan(&count); err != nil {
		return 0, fmt.Errorf("count auth profile feature usages: %w", err)
	}

	return count, nil
}

type authProfileScanner interface {
	Scan(dest ...any) error
}

func scanAuthProfile(scanner authProfileScanner) (*authprofile.Profile, error) {
	var profile authprofile.Profile
	var configJSON []byte
	if err := scanner.Scan(
		&profile.ID,
		&profile.WorkspaceKey,
		&profile.Key,
		&profile.Name,
		&profile.Active,
		&profile.Type,
		&configJSON,
		&profile.CacheTTLSeconds,
		&profile.Version,
		&profile.SecretPayloadEncrypted,
		&profile.HasSecret,
		&profile.CreatedAt,
		&profile.UpdatedAt,
		&profile.CreatedBy,
		&profile.UpdatedBy,
	); err != nil {
		return nil, err
	}
	if err := decodeJSON(configJSON, &profile.Config); err != nil {
		return nil, err
	}

	return &profile, nil
}

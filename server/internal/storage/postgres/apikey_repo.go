package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/apikey"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// APIKeyRepo implements apikey.Repository using PostgreSQL.
type APIKeyRepo struct {
	client *Client
}

// NewAPIKeyRepo creates a new APIKeyRepo.
func NewAPIKeyRepo(client *Client) *APIKeyRepo {
	return &APIKeyRepo{client: client}
}

// Create inserts a new API key row.
func (r *APIKeyRepo) Create(ctx context.Context, key *apikey.APIKey) error {
	if key.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		key.ID = id
	}
	key.WorkspaceKey = wsKey(ctx)

	permissionsJSON, err := jsonBytes(key.Permissions, "[]")
	if err != nil {
		return fmt.Errorf("marshal api key permissions: %w", err)
	}
	createdByPermissionsJSON, err := jsonBytes(key.CreatedByPermissions, "[]")
	if err != nil {
		return fmt.Errorf("marshal api key creator permissions: %w", err)
	}

	_, err = r.client.db(ctx).Exec(ctx, `
		INSERT INTO api_keys (
			id, workspace_key, name, hash, prefix, type, description, permissions,
			created_by, created_by_permissions, created_at, expires_at, last_used_at, revoked
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8::jsonb,
			$9, $10::jsonb, $11, $12, $13, $14
		)
	`,
		key.ID,
		key.WorkspaceKey,
		key.Name,
		key.Hash,
		key.Prefix,
		key.Type,
		key.Description,
		permissionsJSON,
		key.CreatedBy,
		createdByPermissionsJSON,
		key.CreatedAt,
		key.ExpiresAt,
		key.LastUsedAt,
		key.Revoked,
	)
	if err != nil {
		return fmt.Errorf("insert api key: %w", err)
	}

	return nil
}

// FindByHash finds an API key by hash without workspace scoping.
func (r *APIKeyRepo) FindByHash(ctx context.Context, hash string) (*apikey.APIKey, error) {
	row := r.client.db(ctx).QueryRow(ctx, `
		SELECT id, workspace_key, name, hash, prefix, type, description, permissions,
		       created_by, created_by_permissions, created_at, expires_at, last_used_at, revoked
		FROM api_keys
		WHERE hash = $1
	`, hash)

	key, err := scanAPIKey(row)
	if err != nil {
		if isNoRows(err) {
			return nil, apierror.NewUnauthorized("invalid api key", "error.invalidApiKey")
		}
		return nil, fmt.Errorf("find api key by hash: %w", err)
	}

	return key, nil
}

// List returns all API keys in the current workspace.
func (r *APIKeyRepo) List(ctx context.Context) ([]apikey.APIKey, error) {
	return r.listByPredicate(ctx, `workspace_key = $1`, wsKey(ctx))
}

// ListByType returns API keys filtered by type in the current workspace.
func (r *APIKeyRepo) ListByType(ctx context.Context, keyType apikey.KeyType) ([]apikey.APIKey, error) {
	return r.listByPredicate(ctx, `workspace_key = $1 AND type = $2`, wsKey(ctx), keyType)
}

func (r *APIKeyRepo) listByPredicate(ctx context.Context, predicate string, args ...any) ([]apikey.APIKey, error) {
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, workspace_key, name, hash, prefix, type, description, permissions,
		       created_by, created_by_permissions, created_at, expires_at, last_used_at, revoked
		FROM api_keys
		WHERE `+predicate+`
		ORDER BY created_at DESC
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	keys := make([]apikey.APIKey, 0)
	for rows.Next() {
		key, err := scanAPIKey(rows)
		if err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, *key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate api keys: %w", err)
	}

	if keys == nil {
		return []apikey.APIKey{}, nil
	}
	return keys, nil
}

// Revoke marks an API key as revoked.
func (r *APIKeyRepo) Revoke(ctx context.Context, id string) error {
	parsed, err := parseUUID(id)
	if err != nil {
		return apierror.NewBadRequest("invalid api key id", "error.invalidId")
	}

	tag, err := r.client.db(ctx).Exec(ctx, `
		UPDATE api_keys
		SET revoked = TRUE
		WHERE workspace_key = $1 AND id = $2
	`, wsKey(ctx), parsed)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound("api key not found", "error.apiKeyNotFound")
	}

	return nil
}

// UpdateLastUsed updates last_used_at without workspace scoping.
func (r *APIKeyRepo) UpdateLastUsed(ctx context.Context, id string, t time.Time) error {
	parsed, err := parseUUID(id)
	if err != nil {
		return fmt.Errorf("invalid api key id: %w", err)
	}

	if _, err := r.client.db(ctx).Exec(ctx, `
		UPDATE api_keys
		SET last_used_at = $2
		WHERE id = $1
	`, parsed, t); err != nil {
		return fmt.Errorf("update api key last_used_at: %w", err)
	}

	return nil
}

// UpdateHash rotates the stored hash and prefix for a key.
func (r *APIKeyRepo) UpdateHash(ctx context.Context, id string, newHash string, newPrefix string) error {
	parsed, err := parseUUID(id)
	if err != nil {
		return apierror.NewBadRequest("invalid api key id", "error.invalidId")
	}

	tag, err := r.client.db(ctx).Exec(ctx, `
		UPDATE api_keys
		SET hash = $3, prefix = $4
		WHERE workspace_key = $1 AND id = $2 AND revoked = FALSE
	`, wsKey(ctx), parsed, newHash, newPrefix)
	if err != nil {
		return fmt.Errorf("update api key hash: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound("api key not found or revoked", "error.apiKeyNotFound")
	}

	return nil
}

type apiKeyScanner interface {
	Scan(dest ...any) error
}

func scanAPIKey(scanner apiKeyScanner) (*apikey.APIKey, error) {
	var key apikey.APIKey
	var permissionsJSON []byte
	var creatorPermissionsJSON []byte
	if err := scanner.Scan(
		&key.ID,
		&key.WorkspaceKey,
		&key.Name,
		&key.Hash,
		&key.Prefix,
		&key.Type,
		&key.Description,
		&permissionsJSON,
		&key.CreatedBy,
		&creatorPermissionsJSON,
		&key.CreatedAt,
		&key.ExpiresAt,
		&key.LastUsedAt,
		&key.Revoked,
	); err != nil {
		return nil, err
	}
	if err := decodeJSON(permissionsJSON, &key.Permissions); err != nil {
		return nil, err
	}
	if err := decodeJSON(creatorPermissionsJSON, &key.CreatedByPermissions); err != nil {
		return nil, err
	}

	return &key, nil
}

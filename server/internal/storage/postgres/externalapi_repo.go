package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rendis/feature-evaluator/internal/domain/externalapi"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// ExternalAPIRepo implements externalapi.Repository using PostgreSQL.
type ExternalAPIRepo struct {
	client *Client
}

// NewExternalAPIRepo creates a new ExternalAPIRepo.
func NewExternalAPIRepo(client *Client) *ExternalAPIRepo {
	return &ExternalAPIRepo{client: client}
}

// Create inserts a reusable external API.
func (r *ExternalAPIRepo) Create(ctx context.Context, api *externalapi.ExternalAPI) error {
	if api.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		api.ID = id
	}
	api.WorkspaceKey = wsKey(ctx)

	requestJSON, err := jsonBytes(api.Request, "{}")
	if err != nil {
		return fmt.Errorf("marshal external api request: %w", err)
	}
	paramsJSON, err := jsonBytes(api.Params, "[]")
	if err != nil {
		return fmt.Errorf("marshal external api params: %w", err)
	}
	responseValidationJSON, err := jsonBytes(api.ResponseValidation, "{}")
	if err != nil {
		return fmt.Errorf("marshal external api response validation: %w", err)
	}

	_, err = r.client.db(ctx).Exec(ctx, `
		INSERT INTO external_apis (
			id, workspace_key, key, name, active, request, params, response_validation,
			secret_payload_encrypted, has_secrets, version, created_at, updated_at, created_by, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6::jsonb, $7::jsonb, $8::jsonb,
			$9, $10, $11, $12, $13, $14, $15
		)
	`,
		api.ID,
		api.WorkspaceKey,
		api.Key,
		api.Name,
		api.Active,
		requestJSON,
		paramsJSON,
		responseValidationJSON,
		api.SecretPayloadEncrypted,
		api.HasSecrets,
		api.Version,
		api.CreatedAt,
		api.UpdatedAt,
		api.CreatedBy,
		api.UpdatedBy,
	)
	if isUniqueViolation(err) {
		return apierror.NewConflict(
			fmt.Sprintf("external api with key %q already exists", api.Key),
			"error.externalAPIExists",
		)
	}
	if err != nil {
		return fmt.Errorf("insert external api: %w", err)
	}

	return nil
}

// GetByKey finds a reusable external API by key.
func (r *ExternalAPIRepo) GetByKey(ctx context.Context, key string) (*externalapi.ExternalAPI, error) {
	row := r.client.db(ctx).QueryRow(ctx, `
		SELECT id, workspace_key, key, name, active, request, params, response_validation,
		       secret_payload_encrypted, has_secrets, version, created_at, updated_at, created_by, updated_by
		FROM external_apis
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), key)

	api, err := scanExternalAPI(row)
	if err != nil {
		if isNoRows(err) {
			return nil, apierror.NewNotFound(
				fmt.Sprintf("external api %q not found", key),
				"error.externalAPINotFound",
			)
		}
		return nil, fmt.Errorf("find external api: %w", err)
	}

	return api, nil
}

// Update updates a reusable external API, allowing key renames.
func (r *ExternalAPIRepo) Update(ctx context.Context, currentKey string, api *externalapi.ExternalAPI) error {
	requestJSON, err := jsonBytes(api.Request, "{}")
	if err != nil {
		return fmt.Errorf("marshal external api request: %w", err)
	}
	paramsJSON, err := jsonBytes(api.Params, "[]")
	if err != nil {
		return fmt.Errorf("marshal external api params: %w", err)
	}
	responseValidationJSON, err := jsonBytes(api.ResponseValidation, "{}")
	if err != nil {
		return fmt.Errorf("marshal external api response validation: %w", err)
	}

	tag, err := r.client.db(ctx).Exec(ctx, `
		UPDATE external_apis
		SET key = $3, name = $4, active = $5, request = $6::jsonb, params = $7::jsonb,
		    response_validation = $8::jsonb, secret_payload_encrypted = $9,
		    has_secrets = $10, version = $11, updated_at = $12, updated_by = $13
		WHERE workspace_key = $1 AND key = $2
	`,
		wsKey(ctx),
		currentKey,
		api.Key,
		api.Name,
		api.Active,
		requestJSON,
		paramsJSON,
		responseValidationJSON,
		api.SecretPayloadEncrypted,
		api.HasSecrets,
		api.Version,
		api.UpdatedAt,
		api.UpdatedBy,
	)
	if isUniqueViolation(err) {
		return apierror.NewConflict(
			fmt.Sprintf("external api with key %q already exists", api.Key),
			"error.externalAPIExists",
		)
	}
	if err != nil {
		return fmt.Errorf("update external api: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound("external api not found", "error.externalAPINotFound")
	}

	return nil
}

// Delete removes a reusable external API by key.
func (r *ExternalAPIRepo) Delete(ctx context.Context, key string) error {
	tag, err := r.client.db(ctx).Exec(ctx, `
		DELETE FROM external_apis
		WHERE workspace_key = $1 AND key = $2
	`, wsKey(ctx), key)
	if err != nil {
		return fmt.Errorf("delete external api: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apierror.NewNotFound("external api not found", "error.externalAPINotFound")
	}

	return nil
}

// List returns all external APIs in the workspace.
func (r *ExternalAPIRepo) List(ctx context.Context) ([]externalapi.ExternalAPI, error) {
	rows, err := r.client.db(ctx).Query(ctx, `
		SELECT id, workspace_key, key, name, active, request, params, response_validation,
		       secret_payload_encrypted, has_secrets, version, created_at, updated_at, created_by, updated_by
		FROM external_apis
		WHERE workspace_key = $1
		ORDER BY updated_at DESC
	`, wsKey(ctx))
	if err != nil {
		return nil, fmt.Errorf("list external apis: %w", err)
	}
	defer rows.Close()

	apis := make([]externalapi.ExternalAPI, 0)
	for rows.Next() {
		api, err := scanExternalAPI(rows)
		if err != nil {
			return nil, fmt.Errorf("scan external api: %w", err)
		}
		apis = append(apis, *api)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate external apis: %w", err)
	}

	if apis == nil {
		return []externalapi.ExternalAPI{}, nil
	}
	return apis, nil
}

// CountRuleUsages returns how many feature rules reference the given external API key.
func (r *ExternalAPIRepo) CountRuleUsages(ctx context.Context, key string) (int64, error) {
	searchDoc, _ := json.Marshal([]map[string]string{{"externalApiKey": key}})
	var count int64
	if err := r.client.db(ctx).QueryRow(ctx, `
		SELECT COUNT(*)
		FROM feature_rules fr
		JOIN features f ON f.id = fr.feature_id
		WHERE f.workspace_key = $1
		  AND fr.external_validation @> $2::jsonb
	`, wsKey(ctx), searchDoc).Scan(&count); err != nil {
		return 0, fmt.Errorf("count external api rule usages: %w", err)
	}
	return count, nil
}

type externalAPIScanner interface {
	Scan(dest ...any) error
}

func scanExternalAPI(scanner externalAPIScanner) (*externalapi.ExternalAPI, error) {
	var api externalapi.ExternalAPI
	var requestJSON []byte
	var paramsJSON []byte
	var responseValidationJSON []byte
	if err := scanner.Scan(
		&api.ID,
		&api.WorkspaceKey,
		&api.Key,
		&api.Name,
		&api.Active,
		&requestJSON,
		&paramsJSON,
		&responseValidationJSON,
		&api.SecretPayloadEncrypted,
		&api.HasSecrets,
		&api.Version,
		&api.CreatedAt,
		&api.UpdatedAt,
		&api.CreatedBy,
		&api.UpdatedBy,
	); err != nil {
		return nil, err
	}
	if err := decodeJSON(requestJSON, &api.Request); err != nil {
		return nil, err
	}
	if err := decodeJSON(paramsJSON, &api.Params); err != nil {
		return nil, err
	}
	if err := decodeJSON(responseValidationJSON, &api.ResponseValidation); err != nil {
		return nil, err
	}

	return &api, nil
}

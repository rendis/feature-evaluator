package externalapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/resourcekey"
	"github.com/rendis/feature-evaluator/internal/domain/workspace"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// Service handles reusable external API business logic.
type Service struct {
	repo   Repository
	cipher SecretCipher
}

// NewService creates a new external API service.
func NewService(repo Repository, cipher SecretCipher) *Service {
	return &Service{repo: repo, cipher: cipher}
}

// Create validates and stores a new reusable external API.
func (s *Service) Create(ctx context.Context, api *ExternalAPI, secretPayload map[string]string) error {
	api.Key = resourcekey.Normalize(api.Key)
	if !resourcekey.IsValid(api.Key) {
		return apierror.NewBadRequest("invalid external api key format", "error.invalidExternalAPIKey")
	}
	if err := Validate(api); err != nil {
		return err
	}

	mergedSecrets, err := s.resolveDraftSecrets(ctx, "", api, secretPayload, true)
	if err != nil {
		return err
	}

	api.WorkspaceKey = workspace.KeyFromContext(ctx)
	api.Version = 1
	now := time.Now().UTC()
	api.CreatedAt = now
	api.UpdatedAt = now
	if len(mergedSecrets) > 0 {
		ciphertext, encErr := s.cipher.EncryptMap(mergedSecrets, secretAAD(api.WorkspaceKey, api.Key))
		if encErr != nil {
			return fmt.Errorf("encrypting external api secret payload: %w", encErr)
		}
		api.SecretPayloadEncrypted = ciphertext
		api.HasSecrets = true
	}

	return s.repo.Create(ctx, api)
}

// GetByKey retrieves one reusable external API without exposing secrets.
func (s *Service) GetByKey(ctx context.Context, key string) (*ExternalAPI, error) {
	return s.repo.GetByKey(ctx, resourcekey.Normalize(key))
}

// List returns all reusable external APIs in the current workspace.
func (s *Service) List(ctx context.Context) ([]ExternalAPI, error) {
	return s.repo.List(ctx)
}

// Update persists changes, including key renames and secret rotation/merge.
func (s *Service) Update(
	ctx context.Context,
	currentKey string,
	api *ExternalAPI,
	secretPayload map[string]string,
	replaceSecret bool,
) error {
	currentKey = resourcekey.Normalize(currentKey)
	existing, err := s.repo.GetByKey(ctx, currentKey)
	if err != nil {
		return err
	}

	api.Key = resourcekey.Normalize(api.Key)
	if !resourcekey.IsValid(api.Key) {
		return apierror.NewBadRequest("invalid external api key format", "error.invalidExternalAPIKey")
	}
	if err := Validate(api); err != nil {
		return err
	}

	mergedSecrets, err := s.resolveDraftSecrets(ctx, currentKey, api, secretPayload, replaceSecret)
	if err != nil {
		return err
	}

	existing.Key = api.Key
	existing.Name = strings.TrimSpace(api.Name)
	existing.Active = api.Active
	existing.Request = api.Request
	existing.Params = api.Params
	existing.ResponseValidation = api.ResponseValidation
	existing.UpdatedAt = time.Now().UTC()
	existing.UpdatedBy = api.UpdatedBy
	existing.Version++

	if len(mergedSecrets) == 0 {
		existing.SecretPayloadEncrypted = ""
		existing.HasSecrets = false
	} else {
		ciphertext, encErr := s.cipher.EncryptMap(mergedSecrets, secretAAD(existing.WorkspaceKey, existing.Key))
		if encErr != nil {
			return fmt.Errorf("encrypting external api secret payload: %w", encErr)
		}
		existing.SecretPayloadEncrypted = ciphertext
		existing.HasSecrets = true
	}

	return s.repo.Update(ctx, currentKey, existing)
}

// DecryptSecrets decrypts and returns the secret payload for the given API.
func (s *Service) DecryptSecrets(_ context.Context, api *ExternalAPI) (map[string]string, error) {
	if api.SecretPayloadEncrypted == "" {
		return nil, nil
	}
	return s.cipher.DecryptMap(api.SecretPayloadEncrypted, secretAAD(api.WorkspaceKey, api.Key))
}

// Delete removes a reusable external API when no rules reference it.
func (s *Service) Delete(ctx context.Context, key string) error {
	key = resourcekey.Normalize(key)
	count, err := s.repo.CountRuleUsages(ctx, key)
	if err != nil {
		return fmt.Errorf("counting rule usages for external api %q: %w", key, err)
	}
	if count > 0 {
		return apierror.NewConflict(
			fmt.Sprintf("external api %q is used by %d rule(s)", key, count),
			"error.externalApiInUse",
		)
	}
	return s.repo.Delete(ctx, key)
}

// ResolveDraftSecrets validates a draft and returns the merged plaintext secrets needed for tests.
func (s *Service) ResolveDraftSecrets(
	ctx context.Context,
	currentKey string,
	api *ExternalAPI,
	secretPayload map[string]string,
	replaceSecret bool,
) (map[string]string, error) {
	currentKey = resourcekey.Normalize(currentKey)
	if currentKey == "" {
		api.Key = resourcekey.Normalize(api.Key)
		if !resourcekey.IsValid(api.Key) {
			return nil, apierror.NewBadRequest("invalid external api key format", "error.invalidExternalAPIKey")
		}
	} else {
		api.Key = resourcekey.Normalize(api.Key)
		if !resourcekey.IsValid(api.Key) {
			return nil, apierror.NewBadRequest("invalid external api key format", "error.invalidExternalAPIKey")
		}
	}
	if err := Validate(api); err != nil {
		return nil, err
	}
	return s.resolveDraftSecrets(ctx, currentKey, api, secretPayload, replaceSecret)
}

func (s *Service) resolveDraftSecrets(
	ctx context.Context,
	currentKey string,
	api *ExternalAPI,
	secretPayload map[string]string,
	replaceSecret bool,
) (map[string]string, error) {
	merged := map[string]string{}
	if currentKey != "" {
		existing, err := s.repo.GetByKey(ctx, currentKey)
		if err != nil {
			return nil, err
		}
		if existing.SecretPayloadEncrypted != "" && !replaceSecret {
			existingSecrets, decErr := s.cipher.DecryptMap(
				existing.SecretPayloadEncrypted,
				secretAAD(existing.WorkspaceKey, existing.Key),
			)
			if decErr != nil {
				return nil, fmt.Errorf("decrypting external api secret payload: %w", decErr)
			}
			for key, value := range existingSecrets {
				merged[key] = value
			}
		}
	}
	if replaceSecret {
		merged = map[string]string{}
	}
	for key, value := range secretPayload {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		merged[trimmedKey] = value
	}
	if err := validateRequiredSecrets(api, merged); err != nil {
		return nil, err
	}
	return merged, nil
}

func validateRequiredSecrets(api *ExternalAPI, secrets map[string]string) error {
	_, secretRefs := CollectTemplateReferences(api.Request)
	for key := range secretRefs {
		if strings.TrimSpace(secrets[key]) == "" {
			return apierror.NewBadRequest(
				fmt.Sprintf("missing secret payload value for %q", key),
				"error.invalidExternalAPISecret",
			)
		}
	}
	return nil
}

func secretAAD(workspaceKey, apiKey string) string {
	return workspaceKey + ":" + apiKey + ":externalApi"
}

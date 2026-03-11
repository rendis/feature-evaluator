package authprofile

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/resourcekey"
	"github.com/rendis/feature-evaluator/internal/domain/workspace"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// Service handles auth profile business logic.
type Service struct {
	repo   Repository
	cipher SecretCipher
}

// NewService creates a new auth profile service.
func NewService(repo Repository, cipher SecretCipher) *Service {
	return &Service{repo: repo, cipher: cipher}
}

// Create validates and stores a new auth profile.
func (s *Service) Create(ctx context.Context, profile *Profile, secretPayload map[string]string) error {
	profile.Key = resourcekey.Normalize(profile.Key)
	if !resourcekey.IsValid(profile.Key) {
		return apierror.NewBadRequest(
			fmt.Sprintf("invalid auth profile key format: %s", profile.Key),
			"error.invalidAuthProfileKey",
		)
	}
	if strings.TrimSpace(profile.Name) == "" {
		return apierror.NewBadRequest("auth profile name is required", "error.authProfileNameRequired")
	}
	if !profile.Type.Valid() {
		return apierror.NewBadRequest(
			fmt.Sprintf("invalid auth profile type: %s", profile.Type),
			"error.invalidAuthProfileType",
		)
	}
	if profile.Config == nil {
		profile.Config = map[string]any{}
	}
	profile.Normalize()
	if err := ValidateProfile(profile, secretPayload, true, false); err != nil {
		return err
	}

	workspaceKey := workspace.KeyFromContext(ctx)
	profile.WorkspaceKey = workspaceKey

	if len(secretPayload) > 0 {
		ciphertext, err := s.cipher.EncryptMap(secretPayload, secretAAD(workspaceKey, profile.Key))
		if err != nil {
			return fmt.Errorf("encrypting auth profile secret payload: %w", err)
		}
		profile.SecretPayloadEncrypted = ciphertext
		profile.HasSecret = true
	}

	profile.Version = 1
	now := time.Now().UTC()
	profile.CreatedAt = now
	profile.UpdatedAt = now

	return s.repo.Create(ctx, profile)
}

// GetByKey retrieves an auth profile without exposing secrets.
func (s *Service) GetByKey(ctx context.Context, key string) (*Profile, error) {
	return s.repo.GetByKey(ctx, key)
}

// List returns all auth profiles for the current workspace.
func (s *Service) List(ctx context.Context) ([]Profile, error) {
	return s.repo.List(ctx)
}

// Update persists profile changes, supports key renames and optionally rotates its secret payload.
func (s *Service) Update( //nolint:cyclop,funlen // update validates many optional fields
	ctx context.Context,
	currentKey string,
	profile *Profile,
	secretPayload map[string]string,
	replaceSecret bool,
) error {
	currentKey = resourcekey.Normalize(currentKey)
	existing, err := s.repo.GetByKey(ctx, currentKey)
	if err != nil {
		return err
	}

	profile.Key = resourcekey.Normalize(profile.Key)
	if !resourcekey.IsValid(profile.Key) {
		return apierror.NewBadRequest(
			fmt.Sprintf("invalid auth profile key format: %s", profile.Key),
			"error.invalidAuthProfileKey",
		)
	}
	if strings.TrimSpace(profile.Name) == "" {
		return apierror.NewBadRequest("auth profile name is required", "error.authProfileNameRequired")
	}
	if !profile.Type.Valid() {
		return apierror.NewBadRequest(
			fmt.Sprintf("invalid auth profile type: %s", profile.Type),
			"error.invalidAuthProfileType",
		)
	}
	if existing.Type != profile.Type {
		return apierror.NewBadRequest(
			"auth profile type cannot be changed once created",
			"error.authProfileTypeImmutable",
		)
	}
	if profile.Config == nil {
		profile.Config = map[string]any{}
	}
	profile.Normalize()
	if err := ValidateProfile(profile, secretPayload, false, replaceSecret || !existing.HasSecret); err != nil {
		return err
	}

	secretPayloadToStore := secretPayload
	keyChanged := existing.Key != profile.Key
	if keyChanged && existing.HasSecret && !replaceSecret {
		secretPayloadToStore, err = s.cipher.DecryptMap(
			existing.SecretPayloadEncrypted,
			secretAAD(existing.WorkspaceKey, existing.Key),
		)
		if err != nil {
			return fmt.Errorf("decrypting auth profile secret payload for rename: %w", err)
		}
		replaceSecret = true
	}

	existing.Key = profile.Key
	existing.Name = profile.Name
	existing.Active = profile.Active
	existing.Type = profile.Type
	existing.Config = profile.Config
	existing.CacheTTLSeconds = profile.CacheTTLSeconds
	existing.UpdatedAt = time.Now().UTC()
	existing.UpdatedBy = profile.UpdatedBy

	//nolint:nestif // Secret rotation keeps the replace/clear/encrypt branches explicit.
	if replaceSecret {
		if len(secretPayloadToStore) == 0 {
			existing.SecretPayloadEncrypted = ""
			existing.HasSecret = false
		} else {
			ciphertext, encErr := s.cipher.EncryptMap(
				secretPayloadToStore,
				secretAAD(existing.WorkspaceKey, existing.Key),
			)
			if encErr != nil {
				return fmt.Errorf("encrypting auth profile secret payload: %w", encErr)
			}
			existing.SecretPayloadEncrypted = ciphertext
			existing.HasSecret = true
		}
	}

	existing.Version++
	return s.repo.Update(ctx, currentKey, existing)
}

// Delete removes an auth profile when no feature references it.
func (s *Service) Delete(ctx context.Context, key string) error {
	usageCount, err := s.repo.CountFeatureUsages(ctx, key)
	if err != nil {
		return fmt.Errorf("counting auth profile usages: %w", err)
	}
	if usageCount > 0 {
		return apierror.NewConflict(
			fmt.Sprintf("auth profile %q is used by %d feature(s)", key, usageCount),
			"error.authProfileInUse",
		)
	}
	return s.repo.Delete(ctx, key)
}

// Resolve returns the profile and its decrypted secret payload.
func (s *Service) Resolve(ctx context.Context, key string) (*Profile, map[string]string, error) {
	profile, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, nil, err
	}
	secrets := map[string]string{}
	if profile.SecretPayloadEncrypted != "" {
		secrets, err = s.cipher.DecryptMap(profile.SecretPayloadEncrypted, secretAAD(profile.WorkspaceKey, profile.Key))
		if err != nil {
			return nil, nil, fmt.Errorf("decrypting auth profile secret payload: %w", err)
		}
	}
	return profile, secrets, nil
}

func secretAAD(workspaceKey, profileKey string) string {
	return workspaceKey + ":" + profileKey
}

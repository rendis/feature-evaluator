package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/member"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// Repository defines the persistence interface for API keys.
type Repository interface {
	Create(ctx context.Context, key *APIKey) error
	FindByHash(ctx context.Context, hash string) (*APIKey, error)
	List(ctx context.Context) ([]APIKey, error)
	ListByType(ctx context.Context, keyType KeyType) ([]APIKey, error)
	Revoke(ctx context.Context, id string) error
	UpdateLastUsed(ctx context.Context, id string, t time.Time) error
	UpdateHash(ctx context.Context, id string, newHash string, newPrefix string) error
}

// Service handles API key business logic.
type Service struct {
	repo Repository
}

// NewService creates a new API key service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GenerateKey creates a new API key and stores its hash.
// Returns the plaintext key (shown once) and the stored record.
func (s *Service) GenerateKey(
	ctx context.Context,
	name string,
	keyType KeyType,
	permissions []string,
	description string,
	createdBy string,
	createdByPermissions []string,
	expiresAt *time.Time,
) (string, *APIKey, error) {

	// Validate key type.
	if keyType != KeyTypeAdmin {
		return "", nil, apierror.NewBadRequest("invalid key type, must be 'admin'", "error.invalidKeyType")
	}

	if len(permissions) == 0 {
		return "", nil, apierror.NewBadRequest("admin keys require at least one permission", "error.missingPermissions")
	}
	if err := validatePermissions(permissions, createdByPermissions); err != nil {
		return "", nil, err
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generating random key: %w", err)
	}

	prefix := PrefixAdmin
	plaintext := prefix + hex.EncodeToString(raw)
	hash := HashKey(plaintext)
	storedPrefix := plaintext[:len(prefix)+6]

	key := &APIKey{
		Name:                 name,
		Hash:                 hash,
		Prefix:               storedPrefix,
		Type:                 keyType,
		Description:          description,
		Permissions:          permissions,
		CreatedBy:            createdBy,
		CreatedByPermissions: createdByPermissions,
		CreatedAt:            time.Now().UTC(),
		ExpiresAt:            expiresAt,
	}

	if err := s.repo.Create(ctx, key); err != nil {
		return "", nil, fmt.Errorf("storing api key: %w", err)
	}

	return plaintext, key, nil
}

// validatePermissions checks that all requested permissions are allowed for API keys
// and are a subset of the creator's permissions.
func validatePermissions(permissions []string, creatorPermissions []string) error {
	creatorSet := make(map[string]bool, len(creatorPermissions))
	for _, p := range creatorPermissions {
		creatorSet[p] = true
	}

	for _, perm := range permissions {
		if !member.IsAllowedAPIKeyPermission(perm) {
			return apierror.NewBadRequest(
				fmt.Sprintf("permission %q is not allowed for API keys", perm),
				"error.forbiddenPermission",
			)
		}
		if !creatorSet[perm] {
			return apierror.NewForbidden(
				fmt.Sprintf("cannot grant permission %q that you do not have", perm),
				"error.permissionEscalation",
			)
		}
	}
	return nil
}

// Validate checks a plaintext API key against stored hashes.
func (s *Service) Validate(ctx context.Context, plaintext string) (*APIKey, error) {
	hash := HashKey(plaintext)
	key, err := s.repo.FindByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if key.Revoked {
		return nil, apierror.NewUnauthorized("api key revoked", "error.apiKeyRevoked")
	}
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now().UTC()) {
		return nil, apierror.NewUnauthorized("api key expired", "error.apiKeyExpired")
	}
	return key, nil
}

// ValidateAdmin validates an API key and ensures it is an admin key with the required permission.
func (s *Service) ValidateAdmin(ctx context.Context, plaintext string, requiredPerm string) (*APIKey, error) {
	key, err := s.Validate(ctx, plaintext)
	if err != nil {
		return nil, err
	}
	if key.Type != KeyTypeAdmin {
		return nil, apierror.NewForbidden("api key cannot access admin endpoints", "error.evalKeyForbidden")
	}
	if requiredPerm != "" && !key.HasPermission(requiredPerm) {
		return nil, apierror.NewForbidden("api key lacks required permission", "error.insufficientKeyPermission")
	}
	return key, nil
}

// Rotate replaces the hash of an existing key with a new one.
// Returns the new plaintext (shown once).
func (s *Service) Rotate(ctx context.Context, id string) (string, *APIKey, error) {
	// First list to find the key and verify it exists.
	keys, err := s.repo.List(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("listing keys for rotate: %w", err)
	}

	var existing *APIKey
	for i := range keys {
		if keys[i].ID == id {
			existing = &keys[i]
			break
		}
	}
	if existing == nil {
		return "", nil, apierror.NewNotFound("api key not found", "error.apiKeyNotFound")
	}
	if existing.Revoked {
		return "", nil, apierror.NewBadRequest("cannot rotate a revoked key", "error.apiKeyRevoked")
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generating random key: %w", err)
	}

	prefix := PrefixAdmin
	plaintext := prefix + hex.EncodeToString(raw)
	newHash := HashKey(plaintext)
	newPrefix := plaintext[:len(prefix)+6]

	if err := s.repo.UpdateHash(ctx, id, newHash, newPrefix); err != nil {
		return "", nil, fmt.Errorf("rotating api key: %w", err)
	}

	existing.Hash = newHash
	existing.Prefix = newPrefix

	return plaintext, existing, nil
}

// UpdateLastUsed updates the lastUsedAt timestamp for a key.
func (s *Service) UpdateLastUsed(ctx context.Context, id string) {
	_ = s.repo.UpdateLastUsed(ctx, id, time.Now().UTC())
}

// List returns all API keys.
func (s *Service) List(ctx context.Context) ([]APIKey, error) {
	return s.repo.List(ctx)
}

// ListByType returns API keys filtered by type.
func (s *Service) ListByType(ctx context.Context, keyType KeyType) ([]APIKey, error) {
	return s.repo.ListByType(ctx, keyType)
}

// Revoke marks an API key as revoked.
func (s *Service) Revoke(ctx context.Context, id string) error {
	return s.repo.Revoke(ctx, id)
}

// HashKey returns the SHA-256 hex digest of a plaintext key.
func HashKey(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(h[:])
}

package apikey

import "time"

// KeyType distinguishes API key usage scopes.
type KeyType string

const (
	KeyTypeAdmin KeyType = "admin"
)

// Key prefix constants.
const (
	PrefixAdmin = "fev_admin_"
)

// APIKey represents a stored API key with a hashed value.
type APIKey struct {
	ID                   string     `json:"id"`
	WorkspaceKey         string     `json:"workspaceKey"`
	Name                 string     `json:"name"`
	Hash                 string     `json:"-"`
	Prefix               string     `json:"prefix"`
	Type                 KeyType    `json:"type"`
	Description          string     `json:"description"`
	Permissions          []string   `json:"permissions"`
	CreatedBy            string     `json:"createdBy"`
	CreatedByPermissions []string   `json:"createdByPermissions"`
	CreatedAt            time.Time  `json:"createdAt"`
	ExpiresAt            *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt           *time.Time `json:"lastUsedAt,omitempty"`
	Revoked              bool       `json:"revoked"`
}

// HasPermission checks whether this key has a specific permission.
func (k *APIKey) HasPermission(perm string) bool {
	for _, p := range k.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}

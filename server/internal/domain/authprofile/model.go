package authprofile

import "time"

// Type identifies the inbound authentication strategy for eval requests.
type Type string

// Supported inbound authentication mechanisms.
const (
	TypeAPIKey       Type = "api_key"
	TypeOIDCStandard Type = "oidc_standard"
	TypeCustom       Type = "custom"
)

// Valid returns true if the type is supported.
func (t Type) Valid() bool {
	switch t {
	case TypeAPIKey, TypeOIDCStandard, TypeCustom:
		return true
	default:
		return false
	}
}

// Profile stores reusable inbound authentication configuration for eval requests.
type Profile struct {
	ID                     string         `json:"id"`
	WorkspaceKey           string         `json:"workspaceKey"`
	Key                    string         `json:"key"`
	Name                   string         `json:"name"`
	Active                 bool           `json:"active"`
	Type                   Type           `json:"type"`
	Config                 map[string]any `json:"config,omitempty"`
	CacheTTLSeconds        int            `json:"cacheTTLSeconds,omitempty"`
	Version                int            `json:"version"`
	SecretPayloadEncrypted string         `json:"-"`
	HasSecret              bool           `json:"hasSecret"`
	CreatedAt              time.Time      `json:"createdAt"`
	UpdatedAt              time.Time      `json:"updatedAt"`
	CreatedBy              string         `json:"createdBy"`
	UpdatedBy              string         `json:"updatedBy"`
}

// Normalize applies defaults and bounds to runtime cache settings.
func (p *Profile) Normalize() {
	if p.Type == TypeAPIKey || p.Type == TypeOIDCStandard {
		p.CacheTTLSeconds = 0
		return
	}
	if p.CacheTTLSeconds == 0 {
		return
	}
	if p.CacheTTLSeconds < 30 {
		p.CacheTTLSeconds = 30
	}
	if p.CacheTTLSeconds > 3600 {
		p.CacheTTLSeconds = 3600
	}
}

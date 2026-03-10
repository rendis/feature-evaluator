package pack

import "time"

// TargetType represents the scope of a pack activation.
type TargetType string

// Supported target types.
const (
	TargetTenant  TargetType = "tenant"
	TargetCampus  TargetType = "campus"
	TargetProgram TargetType = "program"
)

// ValidTargetType returns true if the target type is one of the supported types.
func ValidTargetType(t string) bool {
	switch TargetType(t) {
	case TargetTenant, TargetCampus, TargetProgram:
		return true
	default:
		return false
	}
}

// MaxFeaturesPerPack is the maximum number of feature keys allowed in a pack.
const MaxFeaturesPerPack = 50

// Pack represents a collection of feature flags bundled together.
type Pack struct {
	ID           string         `json:"id"`
	WorkspaceKey string         `json:"workspaceKey"`
	Key          string         `json:"key"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	FeatureKeys  []string       `json:"featureKeys"`
	Enabled      bool           `json:"enabled"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	CreatedBy    string         `json:"createdBy"`
	UpdatedBy    string         `json:"updatedBy"`
}

// Activation represents the activation of a pack for a specific target.
type Activation struct {
	ID           string         `json:"id"`
	WorkspaceKey string         `json:"workspaceKey"`
	PackKey      string         `json:"packKey"`
	TargetType   TargetType     `json:"targetType"`
	TargetID     string         `json:"targetId"`
	ActivatedAt  time.Time      `json:"activatedAt"`
	ActivatedBy  string         `json:"activatedBy"`
	ExpiresAt    *time.Time     `json:"expiresAt,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

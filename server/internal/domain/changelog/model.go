package changelog

import "time"

// EntityType identifies the kind of entity being changed.
type EntityType string

// Supported entity types.
const (
	EntityFeature    EntityType = "feature"
	EntityRule       EntityType = "rule"
	EntitySegment    EntityType = "segment"
	EntityPack       EntityType = "pack"
	EntityExperiment EntityType = "experiment"
)

// Action identifies what kind of change was made.
type Action string

// Supported actions.
const (
	ActionCreate  Action = "create"
	ActionUpdate  Action = "update"
	ActionDelete  Action = "delete"
	ActionToggle  Action = "toggle"
	ActionReorder Action = "reorder"
)

// ActorType identifies how the actor authenticated.
type ActorType string

// Supported actor types.
const (
	ActorUser   ActorType = "user"
	ActorAPIKey ActorType = "apikey"
	ActorSystem ActorType = "system"
)

// FieldChange represents a single field diff.
type FieldChange struct {
	Field    string `json:"field"`
	OldValue any    `json:"oldValue"`
	NewValue any    `json:"newValue"`
}

// ChangeEntry represents a single change record in the audit trail.
type ChangeEntry struct {
	ID           string         `json:"id"`
	WorkspaceKey string         `json:"workspaceKey"`
	EntityType   EntityType     `json:"entityType"`
	EntityKey    string         `json:"entityKey"`
	ParentKey    string         `json:"parentKey,omitempty"` // e.g. featureKey for rules
	Action       Action         `json:"action"`
	Actor        string         `json:"actor"`
	ActorType    ActorType      `json:"actorType"`
	FieldChanges []FieldChange  `json:"fieldChanges,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
}

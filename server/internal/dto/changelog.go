package dto

import (
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/changelog"
)

// ChangeEntryResponse is the response DTO for a changelog entry.
type ChangeEntryResponse struct {
	ID           string                `json:"id"`
	EntityType   string                `json:"entityType"`
	EntityKey    string                `json:"entityKey"`
	ParentKey    string                `json:"parentKey,omitempty"`
	Action       string                `json:"action"`
	Actor        string                `json:"actor"`
	ActorType    string                `json:"actorType"`
	FieldChanges []FieldChangeResponse `json:"fieldChanges,omitempty"`
	Metadata     map[string]any        `json:"metadata,omitempty"`
	CreatedAt    string                `json:"createdAt"`
}

// FieldChangeResponse is the response DTO for a single field diff.
type FieldChangeResponse struct {
	Field    string `json:"field"`
	OldValue any    `json:"oldValue"`
	NewValue any    `json:"newValue"`
}

// ToChangeEntryResponse maps a domain changelog entry to its response DTO.
func ToChangeEntryResponse(e *changelog.ChangeEntry) ChangeEntryResponse {
	fieldChanges := make([]FieldChangeResponse, 0, len(e.FieldChanges))
	for _, fc := range e.FieldChanges {
		fieldChanges = append(fieldChanges, FieldChangeResponse{
			Field:    fc.Field,
			OldValue: fc.OldValue,
			NewValue: fc.NewValue,
		})
	}

	return ChangeEntryResponse{
		ID:           e.ID,
		EntityType:   string(e.EntityType),
		EntityKey:    e.EntityKey,
		ParentKey:    e.ParentKey,
		Action:       string(e.Action),
		Actor:        e.Actor,
		ActorType:    string(e.ActorType),
		FieldChanges: fieldChanges,
		Metadata:     e.Metadata,
		CreatedAt:    e.CreatedAt.UTC().Format(time.RFC3339),
	}
}

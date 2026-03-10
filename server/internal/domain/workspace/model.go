package workspace

import "time"

// DefaultKey is kept for backwards compatibility with older data and tests.
const DefaultKey = "default"

// Workspace represents a logical tenant workspace.
type Workspace struct {
	ID          string         `json:"id"`
	Key         string         `json:"key"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	CreatedBy   string         `json:"createdBy"`
	ArchivedAt  *time.Time     `json:"archivedAt,omitempty"`
	ArchivedBy  string         `json:"archivedBy,omitempty"`
}

// IsArchived returns whether the workspace has been archived.
func (w *Workspace) IsArchived() bool {
	return w != nil && w.ArchivedAt != nil
}

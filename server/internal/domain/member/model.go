package member

import "time"

// Role represents a team member's role in the workspace.
type Role string

// Supported roles.
const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

// Valid returns true if the role is one of the supported roles.
func (r Role) Valid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleEditor, RoleViewer:
		return true
	default:
		return false
	}
}

// Member represents a team member with access to the workspace.
type Member struct {
	ID           string    `json:"id"`
	WorkspaceKey string    `json:"workspaceKey"`
	Email        string    `json:"email"`
	Role         Role      `json:"role"`
	DisplayName  string    `json:"displayName"`
	AddedBy      string    `json:"addedBy"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

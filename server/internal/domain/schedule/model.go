package schedule

import "time"

// Status represents the lifecycle state of a scheduled change.
type Status string

// Supported statuses.
const (
	StatusPending   Status = "pending"
	StatusExecuting Status = "executing"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// ChangeType represents what kind of change will be applied.
type ChangeType string

// Supported change types.
const (
	ChangeToggle      ChangeType = "toggle"
	ChangeUpdate      ChangeType = "update"
	ChangeDefaultVal  ChangeType = "default_value"
	ChangeEnvironment ChangeType = "environment"
)

// ScheduledChange represents a future change to a feature.
type ScheduledChange struct {
	ID           string         `json:"id"`
	WorkspaceKey string         `json:"workspaceKey"`
	FeatureKey   string         `json:"featureKey"`
	ChangeType   ChangeType     `json:"changeType"`
	Payload      map[string]any `json:"payload"`
	ScheduledAt  time.Time      `json:"scheduledAt"`
	Status       Status         `json:"status"`
	Error        string         `json:"error,omitempty"`
	ExecutedAt   *time.Time     `json:"executedAt,omitempty"`
	CreatedBy    string         `json:"createdBy"`
	CreatedAt    time.Time      `json:"createdAt"`
}

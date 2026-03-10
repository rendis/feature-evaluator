package audit

import "time"

// EvalError represents a sanitized evaluation error record.
type EvalError struct {
	ID           string    `json:"id"`
	WorkspaceKey string    `json:"workspaceKey"`
	FeatureKey   string    `json:"featureKey"`
	RuleID       string    `json:"ruleId,omitempty"`
	ErrorType    string    `json:"errorType"`
	Message      string    `json:"message"`
	TenantID     string    `json:"tenantId,omitempty"`
	CampusID     string    `json:"campusId,omitempty"`
	ProgramID    string    `json:"programId,omitempty"`
	RequestID    string    `json:"requestId,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

// ErrorType constants for categorization.
const (
	ErrorTypeExpression  = "expression_error"
	ErrorTypeExternal    = "external_call_error"
	ErrorTypeTimeout     = "timeout"
	ErrorTypeInternal    = "internal_error"
	ErrorTypeCircuitOpen = "circuit_open"
)

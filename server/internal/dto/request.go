package dto

import (
	"github.com/rendis/feature-evaluator/internal/domain/authprofile"
	"github.com/rendis/feature-evaluator/internal/domain/externalapi"
)

// CreateMemberRequest is the request body for creating a member.
type CreateMemberRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Role        string `json:"role" binding:"required"`
	DisplayName string `json:"displayName"`
}

// UpdateRoleRequest is the request body for changing a member's role.
type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// TransferOwnershipRequest is the request body for ownership transfer.
type TransferOwnershipRequest struct {
	ToMemberID string `json:"toMemberId" binding:"required"`
}

// CreateFeatureRequest is the request body for creating a feature.
type CreateFeatureRequest struct {
	Key                 string               `json:"key" binding:"required"`
	Name                string               `json:"name" binding:"required"`
	Description         string               `json:"description"`
	Enabled             bool                 `json:"enabled"`
	EvalCacheEnabled    bool                 `json:"evalCacheEnabled"`
	EvalCacheTTLSeconds int                  `json:"evalCacheTTLSeconds"`
	ValueType           string               `json:"valueType" binding:"required"`
	DefaultValue        any                  `json:"defaultValue" binding:"required"`
	ActiveFrom          *string              `json:"activeFrom"`
	ActiveUntil         *string              `json:"activeUntil"`
	Environments        []string             `json:"environments"`
	AccessPolicy        string               `json:"accessPolicy"`
	AuthProfileKey      string               `json:"authProfileKey"`
	InputContract       InputContractRequest `json:"inputContract"`
	Metadata            map[string]any       `json:"metadata"`
	Tags                []string             `json:"tags"`
	TrialUntil          *string              `json:"trialUntil"`
	TrialValue          any                  `json:"trialValue"`
}

// UpdateFeatureRequest is the request body for updating a feature.
type UpdateFeatureRequest struct {
	Name                string               `json:"name" binding:"required"`
	Description         string               `json:"description"`
	Enabled             *bool                `json:"enabled"`
	EvalCacheEnabled    *bool                `json:"evalCacheEnabled"`
	EvalCacheTTLSeconds *int                 `json:"evalCacheTTLSeconds"`
	ValueType           string               `json:"valueType"`
	DefaultValue        any                  `json:"defaultValue"`
	ActiveFrom          *string              `json:"activeFrom"`
	ActiveUntil         *string              `json:"activeUntil"`
	Environments        []string             `json:"environments"`
	AccessPolicy        string               `json:"accessPolicy"`
	AuthProfileKey      string               `json:"authProfileKey"`
	InputContract       InputContractRequest `json:"inputContract"`
	Metadata            map[string]any       `json:"metadata"`
	Tags                []string             `json:"tags"`
	TrialUntil          *string              `json:"trialUntil"`
	TrialValue          any                  `json:"trialValue"`
}

// ToggleFeatureRequest is the request body for toggling a feature.
type ToggleFeatureRequest struct {
	Enabled bool `json:"enabled"`
}

// CreateRuleRequest is the request body for creating a rule.
type CreateRuleRequest struct {
	Name                string                      `json:"name" binding:"required"`
	Priority            int                         `json:"priority"`
	Enabled             bool                        `json:"enabled"`
	Expression          string                      `json:"expression" binding:"required"`
	Value               any                         `json:"value" binding:"required"`
	RolloutPercentage   *int                        `json:"rolloutPercentage"`
	SourceBindings      SourceBindingsRequest       `json:"sourceBindings"`
	ExternalAPIBindings []ExternalAPIBindingRequest `json:"externalApiBindings"`
	Metadata            map[string]any              `json:"metadata"`
}

// UpdateRuleRequest is the request body for updating a rule.
type UpdateRuleRequest struct {
	Name                string                      `json:"name" binding:"required"`
	Priority            int                         `json:"priority"`
	Enabled             bool                        `json:"enabled"`
	Expression          string                      `json:"expression" binding:"required"`
	Value               any                         `json:"value" binding:"required"`
	RolloutPercentage   *int                        `json:"rolloutPercentage"`
	SourceBindings      SourceBindingsRequest       `json:"sourceBindings"`
	ExternalAPIBindings []ExternalAPIBindingRequest `json:"externalApiBindings"`
	Metadata            map[string]any              `json:"metadata"`
}

// ExternalAPIBindingRequest binds a workspace-level ExternalAPI to a rule with param mappings.
type ExternalAPIBindingRequest struct {
	ExternalAPIKey string                `json:"externalApiKey" binding:"required"`
	ParamMappings  []ParamMappingRequest `json:"paramMappings"`
	FailMode       string                `json:"failMode"`
	CacheEnabled   bool                  `json:"cacheEnabled"`
	CacheTTL       int                   `json:"cacheTTL"`
}

// ParamMappingRequest maps an external API parameter to an input path or a literal value.
type ParamMappingRequest struct {
	ParamName    string `json:"paramName" binding:"required"`
	Mode         string `json:"mode" binding:"required"` // "input" or "literal"
	InputPath    string `json:"inputPath"`
	LiteralValue string `json:"literalValue"`
}

// ReorderRulesRequest is the request body for reordering rules.
type ReorderRulesRequest struct {
	RuleIDs []string `json:"ruleIds" binding:"required"`
}

// CreateSegmentRequest is the request body for creating a segment.
type CreateSegmentRequest struct {
	Key                       string         `json:"key" binding:"required"`
	Name                      string         `json:"name" binding:"required"`
	Description               string         `json:"description"`
	Metadata                  map[string]any `json:"metadata"`
	MembershipCacheEnabled    bool           `json:"membershipCacheEnabled"`
	MembershipCacheTTLSeconds int            `json:"membershipCacheTTLSeconds"`
	RecordCacheEnabled        bool           `json:"recordCacheEnabled"`
	RecordCacheTTLSeconds     int            `json:"recordCacheTTLSeconds"`
}

// UpdateSegmentRequest is the request body for updating a segment.
type UpdateSegmentRequest struct {
	Name                      string         `json:"name" binding:"required"`
	Description               string         `json:"description"`
	Metadata                  map[string]any `json:"metadata"`
	MembershipCacheEnabled    bool           `json:"membershipCacheEnabled"`
	MembershipCacheTTLSeconds int            `json:"membershipCacheTTLSeconds"`
	RecordCacheEnabled        bool           `json:"recordCacheEnabled"`
	RecordCacheTTLSeconds     int            `json:"recordCacheTTLSeconds"`
}

// ImportSegmentDataRequest is the request body for importing dynamic segment records.
type ImportSegmentDataRequest struct {
	Mode          string           `json:"mode" binding:"required"`
	SourceType    string           `json:"sourceType" binding:"required"`
	RecordKeyPath string           `json:"recordKeyPath" binding:"required"`
	Schema        map[string]any   `json:"schema" binding:"required"`
	Records       []map[string]any `json:"records" binding:"required"`
}

// CreatePackRequest is the request body for creating a pack.
type CreatePackRequest struct {
	Key          string         `json:"key" binding:"required"`
	Name         string         `json:"name" binding:"required"`
	Description  string         `json:"description"`
	FeatureKeys  []string       `json:"featureKeys"`
	Enabled      bool           `json:"enabled"`
	Metadata     map[string]any `json:"metadata"`
	TierKey      *string        `json:"tierKey"`
	InheritsFrom []string       `json:"inheritsFrom"`
	TrialUntil   *string        `json:"trialUntil"`
}

// UpdatePackRequest is the request body for updating a pack.
type UpdatePackRequest struct {
	Name         string         `json:"name" binding:"required"`
	Description  string         `json:"description"`
	FeatureKeys  []string       `json:"featureKeys"`
	Enabled      bool           `json:"enabled"`
	Metadata     map[string]any `json:"metadata"`
	TierKey      *string        `json:"tierKey"`
	InheritsFrom []string       `json:"inheritsFrom"`
	TrialUntil   *string        `json:"trialUntil"`
}

// ActivatePackRequest is the request body for activating a pack on a target.
type ActivatePackRequest struct {
	TargetType string         `json:"targetType" binding:"required"`
	TargetID   string         `json:"targetId" binding:"required"`
	ExpiresAt  *string        `json:"expiresAt"`
	Metadata   map[string]any `json:"metadata"`
}

// DeactivatePackRequest is the request body for deactivating a pack from a target.
type DeactivatePackRequest struct {
	TargetType string `json:"targetType" binding:"required"`
	TargetID   string `json:"targetId" binding:"required"`
}

// ValidateExpressionRequest is the request body for validating an expression.
type ValidateExpressionRequest struct {
	Expression string `json:"expression" binding:"required"`
}

// ValidateExternalAPIExpressionRequest is the request body for validating an
// external API response expression while editing.
type ValidateExternalAPIExpressionRequest struct {
	Expression string `json:"expression"`
}

// TestExpressionRequest is the request body for testing an expression.
type TestExpressionRequest struct {
	Expression string         `json:"expression" binding:"required"`
	Context    map[string]any `json:"context" binding:"required"`
}

type InputContractRequest struct {
	Headers            []InputHeaderRequest `json:"headers"`
	RequestBodyExample map[string]any       `json:"requestBodyExample"`
}

type InputHeaderRequest struct {
	HeaderName    string `json:"headerName" binding:"required"`
	ExpressionKey string `json:"expressionKey"`
	Label         string `json:"label"`
	Type          string `json:"type"`
	Required      bool   `json:"required"`
	Description   string `json:"description"`
}

type SourceBindingsRequest struct {
	Segments []SegmentSourceBindingRequest `json:"segments"`
}

type SegmentSourceBindingRequest struct {
	SegmentKey string `json:"segmentKey" binding:"required"`
	LookupPath string `json:"lookupPath" binding:"required"`
}

type FeatureExpressionTestRequest struct {
	Expression          string                      `json:"expression" binding:"required"`
	SourceBindings      SourceBindingsRequest       `json:"sourceBindings"`
	ExternalAPIBindings []ExternalAPIBindingRequest `json:"externalApiBindings"`
	Scenario            FeatureTestScenario         `json:"scenario" binding:"required"`
}

type FeatureTestScenario struct {
	Headers     map[string]string `json:"headers"`
	RequestBody map[string]any    `json:"requestBody"`
}

// CreateAPIKeyRequest is the request body for creating an API key.
type CreateAPIKeyRequest struct {
	Name        string   `json:"name" binding:"required"`
	Type        string   `json:"type" binding:"required"`
	Permissions []string `json:"permissions"`
	Description string   `json:"description"`
	ExpiresAt   *string  `json:"expiresAt"`
}

// CreateTagRequest is the request body for creating a tag.
type CreateTagRequest struct {
	Name  string `json:"name" binding:"required"`
	Color string `json:"color" binding:"required"`
}

// UpdateTagRequest is the request body for updating a tag.
type UpdateTagRequest struct {
	Name  string `json:"name" binding:"required"`
	Color string `json:"color" binding:"required"`
}

// CreateScheduleRequest is the request body for creating a scheduled change.
type CreateScheduleRequest struct {
	ChangeType  string         `json:"changeType" binding:"required"`
	Payload     map[string]any `json:"payload" binding:"required"`
	ScheduledAt string         `json:"scheduledAt" binding:"required"`
}

// CreateWorkspaceRequest is the request body for creating a workspace.
type CreateWorkspaceRequest struct {
	Key         string `json:"key" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateWorkspaceRequest is the request body for updating a workspace.
type UpdateWorkspaceRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateSecurityPolicyRequest replaces the app-managed global security policy lists.
type UpdateSecurityPolicyRequest struct {
	CORSOrigins []string `json:"corsOrigins"`
}

// CreateAuthProfileRequest is the request body for creating an auth profile.
type CreateAuthProfileRequest struct {
	Key             string            `json:"key" binding:"required"`
	Name            string            `json:"name" binding:"required"`
	Type            authprofile.Type  `json:"type" binding:"required"`
	Active          bool              `json:"active"`
	Config          map[string]any    `json:"config"`
	CacheEnabled    bool              `json:"cacheEnabled"`
	CacheTTLSeconds int               `json:"cacheTTLSeconds"`
	SecretPayload   map[string]string `json:"secretPayload"`
}

// UpdateAuthProfileRequest is the request body for updating an auth profile.
type UpdateAuthProfileRequest struct {
	Key             string            `json:"key" binding:"required"`
	Name            string            `json:"name" binding:"required"`
	Type            authprofile.Type  `json:"type" binding:"required"`
	Active          bool              `json:"active"`
	Config          map[string]any    `json:"config"`
	CacheEnabled    bool              `json:"cacheEnabled"`
	CacheTTLSeconds int               `json:"cacheTTLSeconds"`
	SecretPayload   map[string]string `json:"secretPayload"`
	ReplaceSecret   bool              `json:"replaceSecret"`
}

// TestRequestPayload simulates an incoming eval request during auth profile testing.
type TestAuthProfileRequest struct {
	Name            string                      `json:"name"`
	Type            authprofile.Type            `json:"type" binding:"required"`
	Active          bool                        `json:"active"`
	Config          map[string]any              `json:"config"`
	CacheEnabled    bool                        `json:"cacheEnabled"`
	CacheTTLSeconds int                         `json:"cacheTTLSeconds"`
	SecretPayload   map[string]string           `json:"secretPayload"`
	TestRequest     TestAuthProfileRequestInput `json:"testRequest"`
}

// TestAuthProfileRequestInput simulates the raw request data sent to /eval.
type TestAuthProfileRequestInput struct {
	Headers map[string]string `json:"headers"`
	Query   map[string]string `json:"query"`
	Body    map[string]any    `json:"body"`
}

// CreateExternalAPIRequest creates a reusable external API definition.
type CreateExternalAPIRequest struct {
	Key                 string                           `json:"key" binding:"required"`
	Name                string                           `json:"name" binding:"required"`
	Active              bool                             `json:"active"`
	Request             externalapi.RequestConfig        `json:"request" binding:"required"`
	Params              []externalapi.Param              `json:"params"`
	ExpressionVariables []externalapi.ExpressionVariable `json:"expressionVariables"`
	ResponseValidation  externalapi.ResponseValidation   `json:"responseValidation" binding:"required"`
	SecretPayload       map[string]string                `json:"secretPayload"`
}

// UpdateExternalAPIRequest updates a reusable external API definition.
type UpdateExternalAPIRequest struct {
	Key                 string                           `json:"key" binding:"required"`
	Name                string                           `json:"name" binding:"required"`
	Active              bool                             `json:"active"`
	Request             externalapi.RequestConfig        `json:"request" binding:"required"`
	Params              []externalapi.Param              `json:"params"`
	ExpressionVariables []externalapi.ExpressionVariable `json:"expressionVariables"`
	ResponseValidation  externalapi.ResponseValidation   `json:"responseValidation" binding:"required"`
	SecretPayload       map[string]string                `json:"secretPayload"`
	ReplaceSecret       bool                             `json:"replaceSecret"`
}

// TestExternalAPIRequest executes a reusable external API draft with sample params.
type TestExternalAPIRequest struct {
	CurrentKey          string                           `json:"currentKey"`
	Key                 string                           `json:"key" binding:"required"`
	Name                string                           `json:"name" binding:"required"`
	Request             externalapi.RequestConfig        `json:"request" binding:"required"`
	Params              []externalapi.Param              `json:"params"`
	ExpressionVariables []externalapi.ExpressionVariable `json:"expressionVariables"`
	ResponseValidation  externalapi.ResponseValidation   `json:"responseValidation" binding:"required"`
	SecretPayload       map[string]string                `json:"secretPayload"`
	ReplaceSecret       bool                             `json:"replaceSecret"`
	ParamValues         map[string]any                   `json:"paramValues"`
}

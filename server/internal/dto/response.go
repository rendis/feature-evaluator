package dto

import (
	extdebug "github.com/rendis/feature-evaluator/internal/external"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// ErrorResponse is an alias for swagger documentation. Handlers return
// apierror.ErrorResponse but swag resolves types from the file's imports.
type ErrorResponse = apierror.ErrorResponse

// PaginationResponse is the standard pagination metadata in list responses.
type PaginationResponse struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

// ListResponse is a generic paginated list response.
type ListResponse[T any] struct {
	Data       []T                `json:"data"`
	Pagination PaginationResponse `json:"pagination"`
}

// MemberResponse is the response DTO for a member.
type MemberResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	DisplayName string `json:"displayName"`
	AddedBy     string `json:"addedBy"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// TagResponse is the response DTO for a tag.
type TagResponse struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// TagDetailResponse is the response DTO for a tag with all fields.
type TagDetailResponse struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	CreatedBy string `json:"createdBy"`
}

// FeatureResponse is the response DTO for a feature.
type FeatureResponse struct {
	ID             string                `json:"id"`
	Key            string                `json:"key"`
	Name           string                `json:"name"`
	Description    string                `json:"description"`
	Enabled        bool                  `json:"enabled"`
	ValueType      string                `json:"valueType"`
	DefaultValue   any                   `json:"defaultValue"`
	ActiveFrom     *string               `json:"activeFrom,omitempty"`
	ActiveUntil    *string               `json:"activeUntil,omitempty"`
	Environments   []string              `json:"environments,omitempty"`
	AccessPolicy   string                `json:"accessPolicy,omitempty"`
	AuthProfileKey string                `json:"authProfileKey,omitempty"`
	InputContract  InputContractResponse `json:"inputContract"`
	Metadata       map[string]any        `json:"metadata,omitempty"`
	Tags           []TagResponse         `json:"tags"`
	Packs          []PackRef             `json:"packs"`
	RolloutSalt    string                `json:"rolloutSalt,omitempty"`
	RuleCount      int                   `json:"ruleCount"`
	CreatedAt      string                `json:"createdAt"`
	UpdatedAt      string                `json:"updatedAt"`
	CreatedBy      string                `json:"createdBy"`
	UpdatedBy      string                `json:"updatedBy"`
	TrialUntil     *string               `json:"trialUntil,omitempty"`
	TrialValue     any                   `json:"trialValue,omitempty"`
	Tiers          []TierRef             `json:"tiers"`
}

// FeatureSummaryResponse is the lightweight DTO used by feature list views.
type FeatureSummaryResponse struct {
	ID             string        `json:"id"`
	Key            string        `json:"key"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	Enabled        bool          `json:"enabled"`
	ValueType      string        `json:"valueType"`
	Environments   []string      `json:"environments,omitempty"`
	AccessPolicy   string        `json:"accessPolicy,omitempty"`
	AuthProfileKey string        `json:"authProfileKey,omitempty"`
	Tags           []TagResponse `json:"tags"`
	PackCount      int           `json:"packCount"`
	RuleCount      int           `json:"ruleCount"`
	CreatedAt      string        `json:"createdAt"`
	UpdatedAt      string        `json:"updatedAt"`
	CreatedBy      string        `json:"createdBy"`
	UpdatedBy      string        `json:"updatedBy"`
	TrialUntil     *string       `json:"trialUntil,omitempty"`
	Tiers          []TierRef     `json:"tiers"`
}

// FeatureDetailResponse includes rules in the feature response.
type FeatureDetailResponse struct {
	FeatureResponse
	Rules []RuleResponse `json:"rules"`
}

// RuleResponse is the response DTO for a rule.
type RuleResponse struct {
	ID                  string                       `json:"id"`
	Name                string                       `json:"name"`
	Priority            int                          `json:"priority"`
	Enabled             bool                         `json:"enabled"`
	Expression          string                       `json:"expression"`
	Value               any                          `json:"value"`
	RolloutPercentage   *int                         `json:"rolloutPercentage,omitempty"`
	SourceBindings      SourceBindingsResponse       `json:"sourceBindings"`
	ExternalAPIBindings []ExternalAPIBindingResponse `json:"externalApiBindings"`
	Metadata            map[string]any               `json:"metadata,omitempty"`
	CreatedAt           string                       `json:"createdAt"`
	UpdatedAt           string                       `json:"updatedAt"`
}

// ExternalAPIBindingResponse is the response DTO for a rule's external API binding.
type ExternalAPIBindingResponse struct {
	ExternalAPIKey string                 `json:"externalApiKey"`
	ParamMappings  []ParamMappingResponse `json:"paramMappings"`
	FailMode       string                 `json:"failMode"`
	CacheTTL       int                    `json:"cacheTTL"`
}

// ParamMappingResponse is the response DTO for a parameter mapping.
type ParamMappingResponse struct {
	ParamName    string `json:"paramName"`
	Mode         string `json:"mode"`
	InputPath    string `json:"inputPath,omitempty"`
	LiteralValue string `json:"literalValue,omitempty"`
}

// SegmentResponse is the response DTO for a segment.
type SegmentResponse struct {
	ID            string         `json:"id"`
	Key           string         `json:"key"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	RecordCount   int64          `json:"recordCount"`
	RecordKeyPath string         `json:"recordKeyPath,omitempty"`
	PreviewFields []string       `json:"previewFields,omitempty"`
	SourceType    string         `json:"sourceType,omitempty"`
	LastImportAt  *string        `json:"lastImportAt,omitempty"`
	CreatedAt     string         `json:"createdAt"`
	UpdatedAt     string         `json:"updatedAt"`
	CreatedBy     string         `json:"createdBy"`
	UpdatedBy     string         `json:"updatedBy"`
}

// SegmentSchemaResponse returns the stored schema metadata for a segment.
type SegmentSchemaResponse struct {
	SegmentKey           string         `json:"segmentKey"`
	Schema               map[string]any `json:"schema"`
	RecordKeyPath        string         `json:"recordKeyPath"`
	ActiveDatasetVersion string         `json:"activeDatasetVersion,omitempty"`
	PreviewFields        []string       `json:"previewFields"`
	SourceType           string         `json:"sourceType,omitempty"`
	LastImportAt         *string        `json:"lastImportAt,omitempty"`
	RecordCount          int64          `json:"recordCount"`
}

// SegmentRecordResponse is the response DTO for a segment record.
type SegmentRecordResponse struct {
	ID         string         `json:"id"`
	RecordKey  string         `json:"recordKey"`
	Attributes map[string]any `json:"attributes"`
	CreatedAt  string         `json:"createdAt"`
}

// ImportResultResponse is the response for dynamic segment import.
type ImportResultResponse struct {
	Inserted       int64    `json:"inserted"`
	DatasetVersion string   `json:"datasetVersion,omitempty"`
	PreviewFields  []string `json:"previewFields,omitempty"`
}

// PackRef is a lightweight reference to a pack for embedding in other responses.
type PackRef struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// PackResponse is the response DTO for a pack.
type PackResponse struct {
	ID                   string         `json:"id"`
	Key                  string         `json:"key"`
	Name                 string         `json:"name"`
	Description          string         `json:"description"`
	FeatureKeys          []string       `json:"featureKeys"`
	Enabled              bool           `json:"enabled"`
	Metadata             map[string]any `json:"metadata,omitempty"`
	CreatedAt            string         `json:"createdAt"`
	UpdatedAt            string         `json:"updatedAt"`
	CreatedBy            string         `json:"createdBy"`
	UpdatedBy            string         `json:"updatedBy"`
	TierKey              *string        `json:"tierKey,omitempty"`
	Tier                 *TierRef       `json:"tier,omitempty"`
	InheritsFrom         []string       `json:"inheritsFrom"`
	TrialUntil           *string        `json:"trialUntil,omitempty"`
	ResolvedFeatureCount int            `json:"resolvedFeatureCount"`
}

// ActivationResponse is the response DTO for a pack activation.
type ActivationResponse struct {
	ID          string         `json:"id"`
	PackKey     string         `json:"packKey"`
	TargetType  string         `json:"targetType"`
	TargetID    string         `json:"targetId"`
	ActivatedAt string         `json:"activatedAt"`
	ActivatedBy string         `json:"activatedBy"`
	ExpiresAt   *string        `json:"expiresAt,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ValidateExpressionResponse is the response for expression validation.
type ValidateExpressionResponse struct {
	Valid bool    `json:"valid"`
	Error *string `json:"error,omitempty"`
}

// TestExpressionResponse is the response for expression testing.
type TestExpressionResponse struct {
	Result  any     `json:"result"`
	Matched bool    `json:"matched"`
	Error   *string `json:"error,omitempty"`
}

type InputContractResponse struct {
	Headers            []InputHeaderResponse `json:"headers"`
	RequestBodyExample map[string]any        `json:"requestBodyExample,omitempty"`
	RequestBodySchema  map[string]any        `json:"requestBodySchema,omitempty"`
}

type InputHeaderResponse struct {
	HeaderName    string `json:"headerName"`
	ExpressionKey string `json:"expressionKey"`
	Label         string `json:"label"`
	Type          string `json:"type"`
	Required      bool   `json:"required"`
	Description   string `json:"description,omitempty"`
}

type SourceBindingsResponse struct {
	Segments []SegmentSourceBindingResponse `json:"segments"`
}

type SegmentSourceBindingResponse struct {
	SegmentKey string `json:"segmentKey"`
	LookupPath string `json:"lookupPath"`
}

type FeatureExpressionFieldResponse struct {
	Path        string `json:"path"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"`
	Example     any    `json:"example,omitempty"`
	Group       string `json:"group"`
}

type FeatureExpressionSchemaResponse struct {
	Headers      []FeatureExpressionFieldResponse `json:"headers"`
	RequestBody  []FeatureExpressionFieldResponse `json:"requestBody"`
	Derived      []FeatureExpressionFieldResponse `json:"derived"`
	AdvancedMode bool                             `json:"advancedMode"`
}

type ResolvedSegmentSourceResponse struct {
	SegmentKey  string         `json:"segmentKey"`
	LookupPath  string         `json:"lookupPath"`
	LookupValue any            `json:"lookupValue,omitempty"`
	Found       bool           `json:"found"`
	Data        map[string]any `json:"data,omitempty"`
}

type FeatureExpressionTestResponse struct {
	Result          any                             `json:"result"`
	Matched         bool                            `json:"matched"`
	Derived         map[string]any                  `json:"derived,omitempty"`
	ResolvedSources []ResolvedSegmentSourceResponse `json:"resolvedSources,omitempty"`
	Explanation     string                          `json:"explanation,omitempty"`
	Error           *string                         `json:"error,omitempty"`
}

// AuditErrorResponse is the response DTO for an evaluation error.
type AuditErrorResponse struct {
	ID         string `json:"id"`
	FeatureKey string `json:"featureKey"`
	RuleID     string `json:"ruleId,omitempty"`
	ErrorType  string `json:"errorType"`
	Message    string `json:"message"`
	TenantID   string `json:"tenantId,omitempty"`
	CampusID   string `json:"campusId,omitempty"`
	ProgramID  string `json:"programId,omitempty"`
	RequestID  string `json:"requestId,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

// AuthProfileResponse is the response DTO for an auth profile.
type AuthProfileResponse struct {
	ID              string         `json:"id"`
	Key             string         `json:"key"`
	Name            string         `json:"name"`
	Active          bool           `json:"active"`
	Type            string         `json:"type"`
	Config          map[string]any `json:"config,omitempty"`
	CacheTTLSeconds int            `json:"cacheTTLSeconds,omitempty"`
	Version         int            `json:"version"`
	HasSecret       bool           `json:"hasSecret"`
	CreatedAt       string         `json:"createdAt"`
	UpdatedAt       string         `json:"updatedAt"`
	CreatedBy       string         `json:"createdBy"`
	UpdatedBy       string         `json:"updatedBy"`
}

// ExternalAPIResponse is the response DTO for a reusable external API.
type ExternalAPIResponse struct {
	ID                  string `json:"id"`
	Key                 string `json:"key"`
	Name                string `json:"name"`
	Active              bool   `json:"active"`
	Request             any    `json:"request"`
	Params              any    `json:"params"`
	ExpressionVariables any    `json:"expressionVariables,omitempty"`
	ResponseValidation  any    `json:"responseValidation"`
	HasSecrets          bool   `json:"hasSecrets"`
	Version             int    `json:"version"`
	CreatedAt           string `json:"createdAt"`
	UpdatedAt           string `json:"updatedAt"`
	CreatedBy           string `json:"createdBy"`
	UpdatedBy           string `json:"updatedBy"`
}

// AuthProfileTestResponse is the response DTO for testing an auth profile or external validation.
type AuthProfileTestResponse struct {
	OK         bool           `json:"ok"`
	Attempted  bool           `json:"attempted"`
	Cached     bool           `json:"cached,omitempty"`
	HTTPStatus int            `json:"httpStatus,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
}

// ExternalAPITestResponse is the response DTO for testing a reusable external API.
type ExternalAPITestResponse struct {
	OK         bool                            `json:"ok"`
	Attempted  bool                            `json:"attempted"`
	HTTPStatus int                             `json:"httpStatus,omitempty"`
	Details    *ExternalAPITestDetailsResponse `json:"details,omitempty"`
}

// ExternalAPITestDetailsResponse documents the debug payload returned by a reusable external API test.
type ExternalAPITestDetailsResponse struct {
	Request         *ExternalAPITestRequestResponse `json:"request,omitempty"`
	ResponseText    string                          `json:"responseText,omitempty"`
	ResponseHeaders map[string]string               `json:"responseHeaders,omitempty"`
	ResponseBody    any                             `json:"responseBody,omitempty"`
	Evaluations     *extdebug.APIEvaluationDetails  `json:"evaluations,omitempty"`
}

// ExternalAPITestRequestResponse describes the rendered outbound request used during the test.
type ExternalAPITestRequestResponse struct {
	URL     string            `json:"url,omitempty"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    any               `json:"body,omitempty"`
}

// ExternalAPIExpressionSymbolResponse describes a top-level symbol available in
// reusable external API response validation expressions.
type ExternalAPIExpressionSymbolResponse struct {
	Path        string `json:"path"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// ExternalAPIExpressionActionResponse defines a canonical snippet/action for a
// given expression type.
type ExternalAPIExpressionActionResponse struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	Detail    string   `json:"detail,omitempty"`
	Category  string   `json:"category"`
	AppliesTo []string `json:"appliesTo"`
	Template  string   `json:"template"`
	Priority  int      `json:"priority"`
}

// ExternalAPIExpressionProfileResponse is the semantic catalog consumed by the
// external API expression editor.
type ExternalAPIExpressionProfileResponse struct {
	Keywords []string                              `json:"keywords"`
	Symbols  []ExternalAPIExpressionSymbolResponse `json:"symbols"`
	Actions  []ExternalAPIExpressionActionResponse `json:"actions"`
}

// TierRef is a lightweight tier reference for embedding in other responses.
type TierRef struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// --- Swagger response wrappers ---
// These types document the gin.H shapes returned by handlers.

// DataResponse wraps a single data payload. Used for list endpoints that return {"data": [...]}.
type DataResponse[T any] struct {
	Data T `json:"data"`
}

// MessageResponse represents a simple message response.
type MessageResponse struct {
	Message string `json:"message"`
}

// ToggleMessageResponse represents a toggle response with message and state.
type ToggleMessageResponse struct {
	Message string `json:"message"`
	Enabled bool   `json:"enabled"`
}

// APIKeyCreatedResponse is the response returned when creating or rotating an API key.
type APIKeyCreatedResponse struct {
	Key         string    `json:"key"`
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Prefix      string    `json:"prefix"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	CreatedBy   string    `json:"createdBy"`
	CreatedAt   string    `json:"createdAt"`
	ExpiresAt   *string   `json:"expiresAt,omitempty"`
}

// DashboardStatsResponse is the response for workspace dashboard stats.
type DashboardStatsResponse struct {
	TotalFeatures       int64 `json:"totalFeatures"`
	ActiveFeatures      int64 `json:"activeFeatures"`
	TotalSegments       int64 `json:"totalSegments"`
	TotalSegmentMembers int64 `json:"totalSegmentMembers"`
}

// DashboardActivityItem represents a single activity entry.
type DashboardActivityItem struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	FeatureKey  string `json:"featureKey"`
	Description string `json:"description"`
	CreatedBy   string `json:"createdBy"`
	CreatedAt   string `json:"createdAt"`
}

// DashboardErrorSummaryResponse is the response for error summary.
type DashboardErrorSummaryResponse struct {
	Total  int64          `json:"total"`
	ByType map[string]int `json:"byType"`
}

// DashboardOperationsResponse is the response for operations health check.
type DashboardOperationsResponse struct {
	CheckedAt     string         `json:"checkedAt"`
	OverallStatus string         `json:"overallStatus"`
	Services      map[string]any `json:"services"`
	Metrics       map[string]any `json:"metrics"`
}

// MetricsOverviewResponse is the response for metrics overview.
type MetricsOverviewResponse struct {
	Today map[string]any   `json:"today"`
	Trend []map[string]any `json:"trend"`
}

// MetricsFeatureEntry represents a per-feature metric entry.
type MetricsFeatureEntry struct {
	FeatureKey string `json:"featureKey"`
	Count      int64  `json:"count"`
}

// MetricsTenantEntry represents a per-tenant metric entry.
type MetricsTenantEntry struct {
	TenantID string `json:"tenantId"`
	Count    int64  `json:"count"`
}

// MetricsCacheResponse is the response for cache metrics.
type MetricsCacheResponse struct {
	Hits   int64   `json:"hits"`
	Misses int64   `json:"misses"`
	Ratio  float64 `json:"ratio"`
}

// MetricsExternalResponse is the response for external API metrics.
type MetricsExternalResponse struct {
	P50Ms         float64 `json:"p50Ms"`
	P95Ms         float64 `json:"p95Ms"`
	CBOpenEvents  int64   `json:"cbOpenEvents"`
	CBCloseEvents int64   `json:"cbCloseEvents"`
}

// ExpressionSchemaResponse is the response for the global expression schema endpoint.
type ExpressionSchemaResponse struct {
	Fields    any `json:"fields"`
	Functions any `json:"functions"`
	Operators any `json:"operators"`
	Notes     any `json:"notes"`
}

// HealthResponse is the response for health check endpoints.
type HealthResponse struct {
	Status string         `json:"status"`
	Checks map[string]any `json:"checks,omitempty"`
}

// DeclareWinnerResponse is the response after declaring an experiment winner.
type DeclareWinnerResponse struct {
	Message    string `json:"message"`
	VariantKey string `json:"variantKey"`
}

// EnvironmentItem represents an environment entry.
type EnvironmentItem struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// SecurityPolicyListResponse returns managed, inherited, and effective values for one list.
type SecurityPolicyListResponse struct {
	Managed   []string `json:"managed"`
	Inherited []string `json:"inherited"`
	Effective []string `json:"effective"`
}

// SecurityPolicyResponse is the response DTO for the global security policy.
type SecurityPolicyResponse struct {
	CORSOrigins           SecurityPolicyListResponse `json:"corsOrigins"`
	ExternalAPIAllowHosts SecurityPolicyListResponse `json:"externalApiAllowHosts"`
	UpdatedAt             *string                    `json:"updatedAt,omitempty"`
	UpdatedBy             string                     `json:"updatedBy,omitempty"`
}

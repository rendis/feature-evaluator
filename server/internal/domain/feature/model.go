package feature

import "time"

// Environment represents a deployment environment.
type Environment string

// Supported environments.
const (
	EnvDev        Environment = "dev"
	EnvUAT        Environment = "uat"
	EnvProduction Environment = "production"
)

// AllEnvironments returns the list of valid environments.
func AllEnvironments() []string {
	return []string{string(EnvDev), string(EnvUAT), string(EnvProduction)}
}

// ValidEnvironment returns true if the environment is one of the allowed values.
func ValidEnvironment(env string) bool {
	switch Environment(env) {
	case EnvDev, EnvUAT, EnvProduction:
		return true
	default:
		return false
	}
}

// ValueType represents the type of a feature's value.
type ValueType string

// Supported value types.
const (
	ValueTypeBoolean ValueType = "boolean"
	ValueTypeString  ValueType = "string"
	ValueTypeNumber  ValueType = "number"
	ValueTypeJSON    ValueType = "json"
)

// Valid returns true if the value type is one of the supported types.
func (vt ValueType) Valid() bool {
	switch vt {
	case ValueTypeBoolean, ValueTypeString, ValueTypeNumber, ValueTypeJSON:
		return true
	default:
		return false
	}
}

// InputValueType represents a typed value exposed to rule expressions.
type InputValueType string

// Supported input value types.
const (
	InputValueTypeString  InputValueType = "string"
	InputValueTypeNumber  InputValueType = "number"
	InputValueTypeBoolean InputValueType = "boolean"
)

// Valid returns true if the input value type is supported.
func (ivt InputValueType) Valid() bool {
	switch ivt {
	case InputValueTypeString, InputValueTypeNumber, InputValueTypeBoolean:
		return true
	default:
		return false
	}
}

// AccessPolicy controls whether feature evaluation allows anonymous requests.
type AccessPolicy string

// Supported evaluation access policies.
const (
	AccessPolicyPublic   AccessPolicy = "public"
	AccessPolicyOptional AccessPolicy = "optional"
	AccessPolicyRequired AccessPolicy = "required"
)

// Valid returns true if the access policy is supported.
func (ap AccessPolicy) Valid() bool {
	switch ap {
	case "", AccessPolicyPublic, AccessPolicyOptional, AccessPolicyRequired:
		return true
	default:
		return false
	}
}

// Feature represents a feature flag with embedded rules.
type Feature struct {
	ID             string         `json:"id"`
	WorkspaceKey   string         `json:"workspaceKey"`
	Key            string         `json:"key"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Enabled        bool           `json:"enabled"`
	ValueType      ValueType      `json:"valueType"`
	DefaultValue   any            `json:"defaultValue"`
	ActiveFrom     *time.Time     `json:"activeFrom,omitempty"`
	ActiveUntil    *time.Time     `json:"activeUntil,omitempty"`
	Environments   []string       `json:"environments,omitempty"`
	AccessPolicy   AccessPolicy   `json:"accessPolicy,omitempty"`
	AuthProfileKey string         `json:"authProfileKey,omitempty"`
	InputContract  InputContract  `json:"inputContract,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	Tags           []string       `json:"tags,omitempty"`
	RolloutSalt    string         `json:"rolloutSalt,omitempty"`
	RuleCount      int            `json:"ruleCount,omitempty"`
	PackCount      int            `json:"packCount,omitempty"`
	TrialUntil     *time.Time     `json:"trialUntil,omitempty"`
	TrialValue     any            `json:"trialValue,omitempty"`
	Rules          []Rule         `json:"rules,omitempty"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	CreatedBy      string         `json:"createdBy"`
	UpdatedBy      string         `json:"updatedBy"`
}

// Rule represents a targeting rule within a feature.
type Rule struct {
	ID                  string               `json:"id"`
	Name                string               `json:"name"`
	Priority            int                  `json:"priority"`
	Enabled             bool                 `json:"enabled"`
	Expression          string               `json:"expression"`
	Value               any                  `json:"value"`
	RolloutPercentage   *int                 `json:"rolloutPercentage,omitempty"`
	SourceBindings      SourceBindings       `json:"sourceBindings,omitempty"`
	ExternalAPIBindings []ExternalAPIBinding `json:"externalAPIBindings,omitempty"`
	Metadata            map[string]any       `json:"metadata,omitempty"`
	CreatedAt           time.Time            `json:"createdAt"`
	UpdatedAt           time.Time            `json:"updatedAt"`
}

// InputContract defines the request inputs available to rule expressions.
type InputContract struct {
	Headers            []InputHeader  `json:"headers,omitempty"`
	RequestBodyExample map[string]any `json:"requestBodyExample,omitempty"`
	RequestBodySchema  map[string]any `json:"requestBodySchema,omitempty"`
}

// InputHeader describes a supported request header for a feature.
type InputHeader struct {
	HeaderName    string         `json:"headerName"`
	ExpressionKey string         `json:"expressionKey"`
	Label         string         `json:"label"`
	Type          InputValueType `json:"type"`
	Required      bool           `json:"required"`
	Description   string         `json:"description,omitempty"`
}

// SourceBindings defines the external sources resolved before expression evaluation.
type SourceBindings struct {
	Segments []SegmentSourceBinding `json:"segments,omitempty"`
}

// SegmentSourceBinding binds a segment namespace to a lookup field from the request inputs.
type SegmentSourceBinding struct {
	SegmentKey string `json:"segmentKey"`
	LookupPath string `json:"lookupPath"`
}

// FailMode determines behavior when an external API binding call fails.
type FailMode string

// Supported fail modes.
const (
	FailModeOpen   FailMode = "open"
	FailModeClosed FailMode = "closed"
)

// ExternalAPIBinding references a workspace-level ExternalAPI with param mappings
// for use inside rule expressions via externalApi("key").
type ExternalAPIBinding struct {
	ExternalAPIKey string         `json:"externalApiKey"`
	ParamMappings  []ParamMapping `json:"paramMappings,omitempty"`
	FailMode       FailMode       `json:"failMode"`
	CacheTTL       int            `json:"cacheTTL"`
}

// ParamMapping maps an external API parameter to an input path or a literal value.
type ParamMapping struct {
	ParamName    string `json:"paramName"`
	Mode         string `json:"mode"` // "input" or "literal"
	InputPath    string `json:"inputPath,omitempty"`
	LiteralValue string `json:"literalValue,omitempty"`
}

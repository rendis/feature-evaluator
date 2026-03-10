package externalapi

import "time"

// ParamType represents the runtime type expected for a dynamic request parameter.
type ParamType string

const (
	ParamTypeAny    ParamType = "any"
	ParamTypeString ParamType = "string"
	ParamTypeNumber ParamType = "number"
	ParamTypeBool   ParamType = "bool"
)

// Location identifies where a parameter is referenced in the request template.
type Location string

const (
	LocationURL    Location = "url"
	LocationHeader Location = "header"
	LocationBody   Location = "body"
)

// URLKind identifies which URL section contains a parameter.
type URLKind string

const (
	URLKindDomain URLKind = "domain"
	URLKindPath   URLKind = "path"
	URLKindQuery  URLKind = "query"
)

// ValidationMode determines how the HTTP response is converted into a final boolean result.
type ValidationMode string

const (
	ValidationModeHTTPCode     ValidationMode = "httpCode"
	ValidationModeResponseBody ValidationMode = "responseBody"
	ValidationModeBoth         ValidationMode = "both"
)

// HTTPValidationMode determines how status codes are matched.
type HTTPValidationMode string

const (
	HTTPValidationModeAny2xx     HTTPValidationMode = "any_2xx"
	HTTPValidationModeStatusCode HTTPValidationMode = "status_codes"
)

// HeaderTemplate describes one outbound header template row.
type HeaderTemplate struct {
	KeyTemplate   string `json:"keyTemplate"`
	ValueTemplate string `json:"valueTemplate"`
}

// RequestConfig holds the reusable outbound request definition.
type RequestConfig struct {
	Method       string           `json:"method"`
	URLTemplate  string           `json:"urlTemplate"`
	Headers      []HeaderTemplate `json:"headers,omitempty"`
	BodyTemplate any              `json:"bodyTemplate,omitempty"`
}

// Param describes one dynamic placeholder exposed to callers of the reusable API.
type Param struct {
	Name      string     `json:"name"`
	Type      ParamType  `json:"type"`
	Required  bool       `json:"required"`
	Locations []Location `json:"locations"`
	URLKind   *URLKind   `json:"urlKind,omitempty"`
}

// ExpressionVariable describes an additional input available only to response expressions.
type ExpressionVariable struct {
	Name     string    `json:"name"`
	Type     ParamType `json:"type"`
	Required bool      `json:"required"`
}

// HTTPValidation defines the HTTP status matching rule.
type HTTPValidation struct {
	Mode  HTTPValidationMode `json:"mode"`
	Codes []int              `json:"codes,omitempty"`
}

// BodyValidation defines the response-body validation rule and schema metadata.
type BodyValidation struct {
	Expression         string         `json:"expression,omitempty"`
	Schema             map[string]any `json:"schema,omitempty"`
	SampleResponseText string         `json:"sampleResponseText,omitempty"`
}

// ResponseValidation defines the full response-to-boolean mapping.
type ResponseValidation struct {
	Mode ValidationMode `json:"mode"`
	HTTP HTTPValidation `json:"http"`
	Body BodyValidation `json:"body"`
}

// ExternalAPI stores a reusable, testable outbound API definition.
type ExternalAPI struct {
	ID                     string               `json:"id"`
	WorkspaceKey           string               `json:"workspaceKey"`
	Key                    string               `json:"key"`
	Name                   string               `json:"name"`
	Active                 bool                 `json:"active"`
	Request                RequestConfig        `json:"request"`
	Params                 []Param              `json:"params,omitempty"`
	ExpressionVariables    []ExpressionVariable `json:"expressionVariables,omitempty"`
	ResponseValidation     ResponseValidation   `json:"responseValidation"`
	SecretPayloadEncrypted string               `json:"-"`
	HasSecrets             bool                 `json:"hasSecrets"`
	Version                int                  `json:"version"`
	CreatedAt              time.Time            `json:"createdAt"`
	UpdatedAt              time.Time            `json:"updatedAt"`
	CreatedBy              string               `json:"createdBy"`
	UpdatedBy              string               `json:"updatedBy"`
}

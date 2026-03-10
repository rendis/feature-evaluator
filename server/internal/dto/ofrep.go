package dto

import "github.com/rendis/feature-evaluator/internal/domain/evaluation"

// OFREPEvalRequest is the OFREP single-flag evaluation request body.
type OFREPEvalRequest struct {
	Context map[string]any `json:"context"`
}

// OFREPBulkRequest is the OFREP bulk evaluation request body.
type OFREPBulkRequest struct {
	Context map[string]any `json:"context"`
}

// OFREPEvalResponse is the OFREP single-flag evaluation success response.
type OFREPEvalResponse struct {
	Key      string         `json:"key"`
	Value    any            `json:"value"`
	Variant  string         `json:"variant,omitempty"`
	Reason   string         `json:"reason"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// OFREPErrorResponse is the OFREP per-flag evaluation error response.
type OFREPErrorResponse struct {
	Key          string         `json:"key"`
	ErrorCode    string         `json:"errorCode"`
	ErrorDetails string         `json:"errorDetails,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// OFREPBulkErrorResponse is the OFREP bulk-level error response (400/500).
// Unlike per-flag errors, bulk-level errors have no key field.
type OFREPBulkErrorResponse struct {
	ErrorCode    string `json:"errorCode"`
	ErrorDetails string `json:"errorDetails,omitempty"`
}

// OFREPBulkResponse is the OFREP bulk evaluation response.
type OFREPBulkResponse struct {
	Flags    []any          `json:"flags"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ToOFREPReason maps an internal evaluation.Reason to the OFREP reason string.
// Valid OFREP reason values: STATIC, DEFAULT, TARGETING_MATCH, SPLIT, DISABLED, ERROR, UNKNOWN.
func ToOFREPReason(r evaluation.Reason) string {
	switch r {
	case evaluation.ReasonMatchedRule:
		return "TARGETING_MATCH"
	case evaluation.ReasonDefaultValue:
		return "STATIC"
	case evaluation.ReasonExperiment:
		return "SPLIT"
	case evaluation.ReasonFeatureDisabled,
		evaluation.ReasonNotYetActive,
		evaluation.ReasonExpired,
		evaluation.ReasonEnvironmentMismatch:
		return "DISABLED"
	case evaluation.ReasonRolloutExcluded:
		return "DEFAULT"
	case evaluation.ReasonError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ToOFREPResponse maps an evaluation.Result to an OFREP success response.
func ToOFREPResponse(result evaluation.Result) OFREPEvalResponse {
	variant := ""
	if result.Experiment != nil {
		variant = result.Experiment.VariantKey
	} else if result.MatchedRule != nil {
		variant = result.MatchedRule.Name
	}

	metadata := result.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}

	return OFREPEvalResponse{
		Key:      result.FeatureKey,
		Value:    result.Value,
		Variant:  variant,
		Reason:   ToOFREPReason(result.Reason),
		Metadata: metadata,
	}
}

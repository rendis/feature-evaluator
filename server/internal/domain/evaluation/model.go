package evaluation

import (
	"fmt"
	"time"
)

// Request represents a feature evaluation request.
type Request struct {
	FeatureKey  string         `json:"featureKey"`
	Context     map[string]any `json:"context"`
	Environment string         `json:"environment,omitempty"`

	// Deprecated: Use Context with a "user" namespace instead.
	User map[string]any `json:"user,omitempty"`
}

// NormalizeContext applies backward compatibility: if Context is nil but User
// is set, it wraps User into Context under the "user" namespace.
// Returns true if the legacy format was detected and converted.
func (r *Request) NormalizeContext() bool {
	if r.Context != nil {
		return false
	}
	if r.User != nil {
		r.Context = map[string]any{"user": r.User}
		r.User = nil
		return true
	}
	// Both nil — set empty context
	r.Context = map[string]any{}
	return false
}

// GetNamespace returns the map for a given namespace, or nil if missing.
func (r *Request) GetNamespace(name string) map[string]any {
	return getNamespaceMap(r.Context, name)
}

// GetContextString returns a string value from context[namespace][key].
func (r *Request) GetContextString(namespace, key string) string {
	ns := getNamespaceMap(r.Context, namespace)
	if ns == nil {
		return ""
	}
	if v, ok := ns[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func getNamespaceMap(ctx map[string]any, name string) map[string]any {
	if ctx == nil {
		return nil
	}
	v, ok := ctx[name]
	if !ok {
		return nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return m
}

// BulkRequest represents a bulk evaluation request.
type BulkRequest struct {
	Features []Request `json:"features"`
}

// SegmentResult holds segment membership info for the response.
type SegmentResult struct {
	Key    string `json:"key"`
	Member bool   `json:"member"`
}

// MatchedRule holds the matched rule info for the response.
type MatchedRule struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Reason explains why a particular value was returned.
type Reason string

// Supported evaluation reasons.
const (
	ReasonMatchedRule         Reason = "matched_rule"
	ReasonDefaultValue        Reason = "default_value"
	ReasonFeatureDisabled     Reason = "feature_disabled"
	ReasonNotYetActive        Reason = "not_yet_active"
	ReasonExpired             Reason = "expired"
	ReasonEnvironmentMismatch Reason = "environment_mismatch"
	ReasonUnauthorized        Reason = "unauthorized"
	ReasonExperiment          Reason = "experiment"
	ReasonRolloutExcluded     Reason = "rollout_excluded"
	ReasonError               Reason = "error"
	ReasonTrialActive         Reason = "trial_active"
)

// ExperimentInfo holds experiment assignment info for the response.
type ExperimentInfo struct {
	ExperimentID string `json:"experimentId"`
	VariantKey   string `json:"variantKey"`
}

// Result represents a single feature evaluation result.
type Result struct {
	FeatureKey  string          `json:"featureKey"`
	Value       any             `json:"value"`
	ValueType   string          `json:"valueType"`
	Environment string          `json:"environment,omitempty"`
	MatchedRule *MatchedRule    `json:"matchedRule"`
	PackGrant   string          `json:"packGrant,omitempty"`
	InTrial     bool            `json:"inTrial,omitempty"`
	TrialEndsAt *time.Time      `json:"trialEndsAt,omitempty"`
	TierKeys    []string        `json:"tierKeys,omitempty"`
	Experiment  *ExperimentInfo `json:"experiment,omitempty"`
	Segments    []SegmentResult `json:"segments"`
	Metadata    map[string]any  `json:"metadata,omitempty"`
	Reason      Reason          `json:"reason"`
	EvaluatedAt time.Time       `json:"evaluatedAt"`
	Error       *EvalError      `json:"error,omitempty"`
}

// EvalError represents an inline error in bulk evaluation.
type EvalError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// BulkResult represents the response for a bulk evaluation.
type BulkResult struct {
	Results []Result `json:"results"`
}

// AllRequest represents a request to evaluate all enabled features.
type AllRequest struct {
	Context     map[string]any `json:"context"`
	Environment string         `json:"environment"`
	Tags        []string       `json:"tags,omitempty"`
}

// AllResult represents the response for evaluating all enabled features.
type AllResult struct {
	Features       []Result  `json:"features"`
	TotalEvaluated int       `json:"totalEvaluated"`
	TotalActive    int       `json:"totalActive"`
	Environment    string    `json:"environment"`
	EvaluatedAt    time.Time `json:"evaluatedAt"`
}

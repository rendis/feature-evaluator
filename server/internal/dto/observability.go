package dto

import "github.com/rendis/feature-evaluator/internal/domain/observability"

// ObservabilityOverviewResponse is the response for feature observability overview.
type ObservabilityOverviewResponse struct {
	FeatureKey      string                           `json:"featureKey"`
	Count           int64                            `json:"count"`
	UsedRedisCount  int64                            `json:"usedRedisCount"`
	ErrorCount      int64                            `json:"errorCount"`
	TotalDurationMs int64                            `json:"totalDurationMs"`
	Components      []ObservabilityComponentResponse `json:"components"`
}

// ObservabilityComponentResponse is the response for a component aggregate.
type ObservabilityComponentResponse struct {
	Name               string `json:"name"`
	CacheBackend       string `json:"cacheBackend"`
	CacheEnabled       bool   `json:"cacheEnabled"`
	CacheStatus        string `json:"cacheStatus,omitempty"`
	TTLSeconds         int    `json:"ttlSeconds,omitempty"`
	Count              int64  `json:"count"`
	TotalDurationMs    int64  `json:"totalDurationMs"`
	HitCount           int64  `json:"hitCount"`
	MissCount          int64  `json:"missCount"`
	DisabledCount      int64  `json:"disabledCount"`
	ComputedCount      int64  `json:"computedCount"`
	NotApplicableCount int64  `json:"notApplicableCount"`
}

// ObservabilityRuleResponse is the response for a rule aggregate.
type ObservabilityRuleResponse struct {
	RuleID               string                              `json:"ruleId"`
	Priority             int                                 `json:"priority"`
	Count                int64                               `json:"count"`
	MatchedCount         int64                               `json:"matchedCount"`
	TotalDurationMs      int64                               `json:"totalDurationMs"`
	ExpressionDurationMs int64                               `json:"expressionDurationMs"`
	CompileCacheHitCount int64                               `json:"compileCacheHitCount"`
	ExternalCalls        []ObservabilityExternalCallResponse `json:"externalCalls"`
}

// ObservabilityExternalCallResponse is the response for an external API aggregate.
type ObservabilityExternalCallResponse struct {
	APIKey             string `json:"apiKey"` //nolint:gosec // API key identifier is not a secret value
	Count              int64  `json:"count"`
	TotalDurationMs    int64  `json:"totalDurationMs"`
	HitCount           int64  `json:"hitCount"`
	MissCount          int64  `json:"missCount"`
	DisabledCount      int64  `json:"disabledCount"`
	ComputedCount      int64  `json:"computedCount"`
	NotApplicableCount int64  `json:"notApplicableCount"`
}

// ObservabilityTraceResponse is the response for a persisted evaluation trace.
type ObservabilityTraceResponse struct {
	ID              string                                `json:"id,omitempty"`
	FeatureKey      string                                `json:"featureKey"`
	RequestID       string                                `json:"requestId,omitempty"`
	Environment     string                                `json:"environment,omitempty"`
	UsedRedis       bool                                  `json:"usedRedis"`
	CacheStatus     string                                `json:"cacheStatus,omitempty"`
	TotalDurationMs int64                                 `json:"totalDurationMs"`
	ResultReason    string                                `json:"resultReason,omitempty"`
	ErrorCode       string                                `json:"errorCode,omitempty"`
	CreatedAt       string                                `json:"createdAt"`
	Components      []ObservabilityComponentTraceResponse `json:"components"`
	Rules           []ObservabilityRuleTraceResponse      `json:"rules"`
}

// ObservabilityComponentTraceResponse mirrors a trace component entry.
type ObservabilityComponentTraceResponse struct {
	Name         string `json:"name"`
	CacheBackend string `json:"cacheBackend"`
	CacheEnabled bool   `json:"cacheEnabled"`
	CacheStatus  string `json:"cacheStatus"`
	TTLSeconds   int    `json:"ttlSeconds"`
	DurationMs   int64  `json:"durationMs"`
	Outcome      string `json:"outcome"`
}

// ObservabilityExternalCallTraceResponse mirrors a trace external call.
type ObservabilityExternalCallTraceResponse struct {
	//nolint:gosec // API key identifier is not a secret value
	APIKey      string `json:"apiKey"`
	DurationMs  int64  `json:"durationMs"`
	CacheStatus string `json:"cacheStatus"`
	Passed      bool   `json:"passed"`
	HTTPStatus  int    `json:"httpStatus"`
}

// ObservabilityRuleTraceResponse mirrors a trace rule entry.
type ObservabilityRuleTraceResponse struct {
	RuleID               string                                   `json:"ruleId"`
	Priority             int                                      `json:"priority"`
	Matched              bool                                     `json:"matched"`
	DurationMs           int64                                    `json:"durationMs"`
	ExpressionDurationMs int64                                    `json:"expressionDurationMs"`
	CompileCacheHit      bool                                     `json:"compileCacheHit"`
	ExternalCalls        []ObservabilityExternalCallTraceResponse `json:"externalCalls"`
}

// ToObservabilityOverviewResponse maps a domain overview aggregate to DTO.
func ToObservabilityOverviewResponse(o *observability.Overview) ObservabilityOverviewResponse {
	components := make([]ObservabilityComponentResponse, 0, len(o.Components))
	for i := range o.Components {
		components = append(components, ObservabilityComponentResponse{
			Name:               o.Components[i].Name,
			CacheBackend:       string(o.Components[i].CacheBackend),
			CacheEnabled:       o.Components[i].CacheEnabled,
			CacheStatus:        string(o.Components[i].CacheStatus),
			TTLSeconds:         o.Components[i].TTLSeconds,
			Count:              o.Components[i].Count,
			TotalDurationMs:    o.Components[i].TotalDurationMs,
			HitCount:           o.Components[i].HitCount,
			MissCount:          o.Components[i].MissCount,
			DisabledCount:      o.Components[i].DisabledCount,
			ComputedCount:      o.Components[i].ComputedCount,
			NotApplicableCount: o.Components[i].NotApplicableCount,
		})
	}
	return ObservabilityOverviewResponse{
		FeatureKey:      o.FeatureKey,
		Count:           o.Count,
		UsedRedisCount:  o.UsedRedisCount,
		ErrorCount:      o.ErrorCount,
		TotalDurationMs: o.TotalDurationMs,
		Components:      components,
	}
}

// ToObservabilityRuleResponse maps a domain rule summary to DTO.
func ToObservabilityRuleResponse(r observability.RuleSummary) ObservabilityRuleResponse {
	calls := make([]ObservabilityExternalCallResponse, 0, len(r.ExternalCalls))
	for i := range r.ExternalCalls {
		calls = append(calls, ObservabilityExternalCallResponse{
			APIKey:             r.ExternalCalls[i].APIKey,
			Count:              r.ExternalCalls[i].Count,
			TotalDurationMs:    r.ExternalCalls[i].TotalDurationMs,
			HitCount:           r.ExternalCalls[i].HitCount,
			MissCount:          r.ExternalCalls[i].MissCount,
			DisabledCount:      r.ExternalCalls[i].DisabledCount,
			ComputedCount:      r.ExternalCalls[i].ComputedCount,
			NotApplicableCount: r.ExternalCalls[i].NotApplicableCount,
		})
	}
	return ObservabilityRuleResponse{
		RuleID:               r.RuleID,
		Priority:             r.Priority,
		Count:                r.Count,
		MatchedCount:         r.MatchedCount,
		TotalDurationMs:      r.TotalDurationMs,
		ExpressionDurationMs: r.ExpressionDurationMs,
		CompileCacheHitCount: r.CompileCacheHitCount,
		ExternalCalls:        calls,
	}
}

// ToObservabilityTraceResponse maps a persisted trace to DTO.
func ToObservabilityTraceResponse(t observability.EvaluationTrace) ObservabilityTraceResponse {
	components := make([]ObservabilityComponentTraceResponse, 0, len(t.Components))
	for i := range t.Components {
		components = append(components, ObservabilityComponentTraceResponse{
			Name:         t.Components[i].Name,
			CacheBackend: string(t.Components[i].CacheBackend),
			CacheEnabled: t.Components[i].CacheEnabled,
			CacheStatus:  string(t.Components[i].CacheStatus),
			TTLSeconds:   t.Components[i].TTLSeconds,
			DurationMs:   t.Components[i].DurationMs,
			Outcome:      t.Components[i].Outcome,
		})
	}
	rules := make([]ObservabilityRuleTraceResponse, 0, len(t.Rules))
	for i := range t.Rules {
		calls := make([]ObservabilityExternalCallTraceResponse, 0, len(t.Rules[i].ExternalCalls))
		for j := range t.Rules[i].ExternalCalls {
			calls = append(calls, ObservabilityExternalCallTraceResponse{
				APIKey:      t.Rules[i].ExternalCalls[j].APIKey,
				DurationMs:  t.Rules[i].ExternalCalls[j].DurationMs,
				CacheStatus: string(t.Rules[i].ExternalCalls[j].CacheStatus),
				Passed:      t.Rules[i].ExternalCalls[j].Passed,
				HTTPStatus:  t.Rules[i].ExternalCalls[j].HTTPStatus,
			})
		}
		rules = append(rules, ObservabilityRuleTraceResponse{
			RuleID:               t.Rules[i].RuleID,
			Priority:             t.Rules[i].Priority,
			Matched:              t.Rules[i].Matched,
			DurationMs:           t.Rules[i].DurationMs,
			ExpressionDurationMs: t.Rules[i].ExpressionDurationMs,
			CompileCacheHit:      t.Rules[i].CompileCacheHit,
			ExternalCalls:        calls,
		})
	}
	return ObservabilityTraceResponse{
		ID:              t.ID,
		FeatureKey:      t.FeatureKey,
		RequestID:       t.RequestID,
		Environment:     t.Environment,
		UsedRedis:       t.UsedRedis,
		CacheStatus:     string(t.CacheStatus),
		TotalDurationMs: t.TotalDurationMs,
		ResultReason:    t.ResultReason,
		ErrorCode:       t.ErrorCode,
		CreatedAt:       t.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		Components:      components,
		Rules:           rules,
	}
}

package observability

import (
	"context"
	"time"
)

// CacheBackend identifies where a component's cache lives.
type CacheBackend string

const (
	CacheBackendNone   CacheBackend = "none"
	CacheBackendMemory CacheBackend = "memory"
	CacheBackendRedis  CacheBackend = "redis"
)

// CacheStatus describes how a component behaved with respect to caching.
type CacheStatus string

const (
	CacheStatusHit           CacheStatus = "hit"
	CacheStatusMiss          CacheStatus = "miss"
	CacheStatusDisabled      CacheStatus = "disabled"
	CacheStatusComputed      CacheStatus = "computed"
	CacheStatusNotApplicable CacheStatus = "not_applicable"
)

// EvaluationMeta contains request metadata captured at the beginning of an evaluation.
type EvaluationMeta struct {
	WorkspaceKey string
	FeatureKey   string
	RequestID    string
	Environment  string
	StartedAt    time.Time
}

// ComponentTrace captures one named evaluation step.
type ComponentTrace struct {
	Name         string       `json:"name"`
	CacheBackend CacheBackend `json:"cacheBackend"`
	CacheEnabled bool         `json:"cacheEnabled"`
	CacheStatus  CacheStatus  `json:"cacheStatus"`
	TTLSeconds   int          `json:"ttlSeconds"`
	DurationMs   int64        `json:"durationMs"`
	Outcome      string       `json:"outcome"`
}

// ExternalCallTrace captures one external API binding call.
type ExternalCallTrace struct {
	APIKey      string      `json:"apiKey"` //nolint:gosec // API key identifier is not a secret value
	DurationMs  int64       `json:"durationMs"`
	CacheStatus CacheStatus `json:"cacheStatus"`
	Passed      bool        `json:"passed"`
	HTTPStatus  int         `json:"httpStatus"`
}

// RuleTrace captures rule evaluation timings and nested external calls.
type RuleTrace struct {
	RuleID               string              `json:"ruleId"`
	Priority             int                 `json:"priority"`
	Matched              bool                `json:"matched"`
	DurationMs           int64               `json:"durationMs"`
	ExpressionDurationMs int64               `json:"expressionDurationMs"`
	CompileCacheHit      bool                `json:"compileCacheHit"`
	ExternalCalls        []ExternalCallTrace `json:"externalCalls"`
}

// EvaluationTrace captures one request evaluation end-to-end.
type EvaluationTrace struct {
	ID              string           `json:"id,omitempty"`
	WorkspaceKey    string           `json:"workspaceKey,omitempty"`
	FeatureKey      string           `json:"featureKey"`
	RequestID       string           `json:"requestId,omitempty"`
	Environment     string           `json:"environment,omitempty"`
	UsedRedis       bool             `json:"usedRedis"`
	CacheStatus     CacheStatus      `json:"cacheStatus"`
	TotalDurationMs int64            `json:"totalDurationMs"`
	ResultReason    string           `json:"resultReason,omitempty"`
	ErrorCode       string           `json:"errorCode,omitempty"`
	CreatedAt       time.Time        `json:"createdAt"`
	Components      []ComponentTrace `json:"components"`
	Rules           []RuleTrace      `json:"rules"`
}

// ComponentSummary aggregates a component's behavior across traces.
type ComponentSummary struct {
	Name               string       `json:"name"`
	CacheBackend       CacheBackend `json:"cacheBackend"`
	CacheEnabled       bool         `json:"cacheEnabled"`
	CacheStatus        CacheStatus  `json:"cacheStatus,omitempty"`
	TTLSeconds         int          `json:"ttlSeconds,omitempty"`
	Count              int64        `json:"count"`
	TotalDurationMs    int64        `json:"totalDurationMs"`
	HitCount           int64        `json:"hitCount"`
	MissCount          int64        `json:"missCount"`
	DisabledCount      int64        `json:"disabledCount"`
	ComputedCount      int64        `json:"computedCount"`
	NotApplicableCount int64        `json:"notApplicableCount"`
}

// ExternalCallSummary aggregates one external API binding across traces.
type ExternalCallSummary struct {
	APIKey             string `json:"apiKey"` //nolint:gosec // API key identifier is not a secret value
	Count              int64  `json:"count"`
	TotalDurationMs    int64  `json:"totalDurationMs"`
	HitCount           int64  `json:"hitCount"`
	MissCount          int64  `json:"missCount"`
	DisabledCount      int64  `json:"disabledCount"`
	ComputedCount      int64  `json:"computedCount"`
	NotApplicableCount int64  `json:"notApplicableCount"`
}

// RuleSummary aggregates one rule across traces.
type RuleSummary struct {
	RuleID               string                `json:"ruleId"`
	Priority             int                   `json:"priority"`
	Count                int64                 `json:"count"`
	MatchedCount         int64                 `json:"matchedCount"`
	TotalDurationMs      int64                 `json:"totalDurationMs"`
	ExpressionDurationMs int64                 `json:"expressionDurationMs"`
	CompileCacheHitCount int64                 `json:"compileCacheHitCount"`
	ExternalCalls        []ExternalCallSummary `json:"externalCalls"`
}

// Overview aggregates the top-level evaluation metrics for a feature.
type Overview struct {
	FeatureKey      string             `json:"featureKey"`
	Count           int64              `json:"count"`
	UsedRedisCount  int64              `json:"usedRedisCount"`
	ErrorCount      int64              `json:"errorCount"`
	TotalDurationMs int64              `json:"totalDurationMs"`
	Components      []ComponentSummary `json:"components"`
}

// Observer creates per-request trace recorders.
type Observer interface {
	Start(meta EvaluationMeta) TraceRecorder
}

// TraceRecorder records an evaluation trace incrementally.
type TraceRecorder interface {
	MarkUsedRedis()
	RecordComponent(component ComponentTrace)
	StartRule(ruleID string, priority int) RuleRecorder
	Finalize(resultReason, errorCode string, totalDuration time.Duration)
	Trace() EvaluationTrace
}

// RuleRecorder records a single rule evaluation.
type RuleRecorder interface {
	RecordExpression(duration time.Duration, compileCacheHit bool)
	RecordExternalCall(call ExternalCallTrace)
	Finalize(matched bool, duration time.Duration)
}

// NoopObserver is an observer that discards all traces.
type NoopObserver struct{}

// Start implements Observer.
func (NoopObserver) Start(meta EvaluationMeta) TraceRecorder {
	return NoopTraceRecorder{trace: EvaluationTrace{
		WorkspaceKey: meta.WorkspaceKey,
		FeatureKey:   meta.FeatureKey,
		RequestID:    meta.RequestID,
		Environment:  meta.Environment,
		CreatedAt:    meta.StartedAt,
	}}
}

// NoopTraceRecorder is a recorder that discards all calls.
type NoopTraceRecorder struct {
	trace EvaluationTrace
}

// MarkUsedRedis implements TraceRecorder.
func (n NoopTraceRecorder) MarkUsedRedis() {}

// RecordComponent implements TraceRecorder.
func (n NoopTraceRecorder) RecordComponent(ComponentTrace) {}

// StartRule implements TraceRecorder.
func (n NoopTraceRecorder) StartRule(ruleID string, priority int) RuleRecorder {
	return NoopRuleRecorder{}
}

// Finalize implements TraceRecorder.
func (n NoopTraceRecorder) Finalize(resultReason, errorCode string, totalDuration time.Duration) {}

// Trace implements TraceRecorder.
func (n NoopTraceRecorder) Trace() EvaluationTrace { return n.trace }

// NoopRuleRecorder is a no-op rule recorder.
type NoopRuleRecorder struct{}

// RecordExpression implements RuleRecorder.
func (NoopRuleRecorder) RecordExpression(duration time.Duration, compileCacheHit bool) {}

// RecordExternalCall implements RuleRecorder.
func (NoopRuleRecorder) RecordExternalCall(call ExternalCallTrace) {}

// Finalize implements RuleRecorder.
func (NoopRuleRecorder) Finalize(matched bool, duration time.Duration) {}

type traceRecorderKey struct{}
type ruleRecorderKey struct{}

// WithTraceRecorder stores a trace recorder in the context.
func WithTraceRecorder(ctx context.Context, recorder TraceRecorder) context.Context {
	return context.WithValue(ctx, traceRecorderKey{}, recorder)
}

// TraceRecorderFromContext retrieves a trace recorder from context.
func TraceRecorderFromContext(ctx context.Context) (TraceRecorder, bool) {
	recorder, ok := ctx.Value(traceRecorderKey{}).(TraceRecorder)
	return recorder, ok
}

// WithRuleRecorder stores a rule recorder in the context.
func WithRuleRecorder(ctx context.Context, recorder RuleRecorder) context.Context {
	return context.WithValue(ctx, ruleRecorderKey{}, recorder)
}

// RuleRecorderFromContext retrieves a rule recorder from context.
func RuleRecorderFromContext(ctx context.Context) (RuleRecorder, bool) {
	recorder, ok := ctx.Value(ruleRecorderKey{}).(RuleRecorder)
	return recorder, ok
}

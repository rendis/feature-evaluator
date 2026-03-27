package evalmetrics

import (
	"context"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rendis/feature-evaluator/internal/domain/audit"
	"github.com/rendis/feature-evaluator/internal/domain/observability"
	"github.com/rendis/feature-evaluator/internal/domain/workspace"
	redisclient "github.com/rendis/feature-evaluator/internal/storage/redis"
)

const observabilityKeyPrefix = "fe:obs:"

// ObservabilityService records evaluation traces and exposes aggregated summaries.
type ObservabilityService struct {
	rdb       *redisclient.Client
	traceRepo audit.TraceRepository
}

// NewObservabilityService creates a new observability service.
func NewObservabilityService(rdb *redisclient.Client, traceRepo audit.TraceRepository) *ObservabilityService {
	return &ObservabilityService{rdb: rdb, traceRepo: traceRepo}
}

// Start implements observability.Observer.
func (s *ObservabilityService) Start(meta observability.EvaluationMeta) observability.TraceRecorder {
	return &observabilityRecorder{
		service: s,
		trace: observability.EvaluationTrace{
			WorkspaceKey: meta.WorkspaceKey,
			FeatureKey:   meta.FeatureKey,
			RequestID:    meta.RequestID,
			Environment:  meta.Environment,
			CreatedAt:    meta.StartedAt,
		},
	}
}

// Overview returns top-level aggregates for a feature.
func (s *ObservabilityService) Overview(ctx context.Context, featureKey string) (*observability.Overview, error) {
	overview := &observability.Overview{FeatureKey: featureKey}
	if s.rdb == nil || !s.rdb.Available() {
		return overview, nil
	}
	wsKey := workspace.KeyFromContext(ctx)
	featureKey = strings.TrimSpace(featureKey)
	featureHash := s.hashKey(wsKey, "feature", featureKey)
	fields, err := s.rdb.Underlying().HGetAll(ctx, featureHash).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("read feature observability hash: %w", err)
	}
	overview.Count = parseInt64(fields["count"])
	overview.UsedRedisCount = parseInt64(fields["used_redis_count"])
	overview.ErrorCount = parseInt64(fields["error_count"])
	overview.TotalDurationMs = parseInt64(fields["duration_sum_ms"])

	componentKeys, err := s.scanKeys(ctx, s.hashPattern(wsKey, "feature", featureKey, "component", "*"))
	if err != nil {
		return nil, err
	}
	components := make([]observability.ComponentSummary, 0, len(componentKeys))
	for _, key := range componentKeys {
		compFields, err := s.rdb.Underlying().HGetAll(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("read component observability hash: %w", err)
		}
		components = append(components, componentSummaryFromHash(key, compFields))
	}
	sort.Slice(components, func(i, j int) bool { return components[i].Name < components[j].Name })
	overview.Components = components
	return overview, nil
}

// Rules returns aggregates for every rule in a feature.
func (s *ObservabilityService) Rules(ctx context.Context, featureKey string) ([]observability.RuleSummary, error) {
	if s.rdb == nil {
		return nil, nil
	}
	wsKey := workspace.KeyFromContext(ctx)
	keys, err := s.scanKeys(ctx, s.hashPattern(wsKey, "feature", featureKey, "rule", "*"))
	if err != nil {
		return nil, err
	}
	summaries := make([]observability.RuleSummary, 0, len(keys))
	for _, key := range keys {
		fields, err := s.rdb.Underlying().HGetAll(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("read rule observability hash: %w", err)
		}
		summaries = append(summaries, ruleSummaryFromHash(key, fields))
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].RuleID < summaries[j].RuleID })
	for i := range summaries {
		summaries[i].ExternalCalls = s.ruleExternalCalls(ctx, featureKey, summaries[i].RuleID)
	}
	return summaries, nil
}

// Rule returns aggregates for a single rule.
func (s *ObservabilityService) Rule(ctx context.Context, featureKey, ruleID string) (*observability.RuleSummary, error) {
	if s.rdb == nil {
		summary := observability.RuleSummary{RuleID: ruleID}
		return &summary, nil
	}
	wsKey := workspace.KeyFromContext(ctx)
	key := s.hashKey(wsKey, "feature", featureKey, "rule", ruleID)
	fields, err := s.rdb.Underlying().HGetAll(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("read rule observability hash: %w", err)
	}
	summary := ruleSummaryFromHash(key, fields)
	summary.ExternalCalls = s.ruleExternalCalls(ctx, featureKey, ruleID)
	return &summary, nil
}

// Traces proxies trace listing to the persistence layer.
func (s *ObservabilityService) Traces(ctx context.Context, params audit.TraceListParams) (*audit.TraceListResult, error) {
	if s.traceRepo == nil {
		return &audit.TraceListResult{Data: []audit.EvalTrace{}, Page: params.Page, PageSize: params.PageSize}, nil
	}
	return s.traceRepo.List(ctx, params)
}

func (s *ObservabilityService) ruleExternalCalls(ctx context.Context, featureKey, ruleID string) []observability.ExternalCallSummary {
	if s.rdb == nil {
		return nil
	}
	wsKey := workspace.KeyFromContext(ctx)
	keys, err := s.scanKeys(ctx, s.hashPattern(wsKey, "feature", featureKey, "rule", ruleID, "api", "*"))
	if err != nil || len(keys) == 0 {
		return nil
	}
	summaries := make([]observability.ExternalCallSummary, 0, len(keys))
	for _, key := range keys {
		fields, err := s.rdb.Underlying().HGetAll(ctx, key).Result()
		if err != nil {
			continue
		}
		summaries = append(summaries, externalCallSummaryFromHash(key, fields))
	}
	sort.Slice(summaries, func(i, j int) bool { return summaries[i].APIKey < summaries[j].APIKey })
	return summaries
}

func (s *ObservabilityService) recordTrace(ctx context.Context, trace observability.EvaluationTrace) {
	if s.rdb != nil && s.rdb.Available() {
		s.recordTraceInRedis(ctx, trace)
	}

	if shouldPersistTrace(trace) {
		s.persistTrace(trace)
	}
}

func (s *ObservabilityService) recordTraceInRedis(ctx context.Context, trace observability.EvaluationTrace) {
	pipe := s.rdb.Underlying().Pipeline()
	s.recordFeatureTrace(ctx, pipe, trace)
	s.recordComponentTraces(ctx, pipe, trace)
	s.recordRuleTraces(ctx, pipe, trace)
	_, _ = pipe.Exec(ctx)
}

func (s *ObservabilityService) recordFeatureTrace(ctx context.Context, pipe redis.Pipeliner, trace observability.EvaluationTrace) {
	traceKey := s.hashKey(trace.WorkspaceKey, "feature", trace.FeatureKey)
	s.incrementHash(pipe, traceKey, "count", 1)
	s.incrementHash(pipe, traceKey, "duration_sum_ms", trace.TotalDurationMs)
	if trace.UsedRedis {
		s.incrementHash(pipe, traceKey, "used_redis_count", 1)
	}
	if trace.ErrorCode != "" {
		s.incrementHash(pipe, traceKey, "error_count", 1)
	}
	_ = s.rdb.Underlying().Expire(ctx, traceKey, 30*24*time.Hour)
}

func (s *ObservabilityService) recordComponentTraces(ctx context.Context, pipe redis.Pipeliner, trace observability.EvaluationTrace) {
	for _, component := range trace.Components {
		s.recordComponentTrace(ctx, pipe, trace, component)
	}
}

func (s *ObservabilityService) recordComponentTrace(ctx context.Context, pipe redis.Pipeliner, trace observability.EvaluationTrace, component observability.ComponentTrace) {
	key := s.hashKey(trace.WorkspaceKey, "feature", trace.FeatureKey, "component", component.Name)
	pipe.HSet(ctx, key, "cache_backend", string(component.CacheBackend))
	pipe.HSet(ctx, key, "cache_enabled", component.CacheEnabled)
	pipe.HSet(ctx, key, "ttl_seconds", component.TTLSeconds)
	s.incrementHash(pipe, key, "count", 1)
	s.incrementHash(pipe, key, "duration_sum_ms", component.DurationMs)
	s.incrementHash(pipe, key, string(component.CacheStatus)+"_count", 1)
	_ = s.rdb.Underlying().Expire(ctx, key, 30*24*time.Hour)
}

func (s *ObservabilityService) recordRuleTraces(ctx context.Context, pipe redis.Pipeliner, trace observability.EvaluationTrace) {
	for _, rule := range trace.Rules {
		s.recordRuleTrace(ctx, pipe, trace, rule)
	}
}

func (s *ObservabilityService) recordRuleTrace(ctx context.Context, pipe redis.Pipeliner, trace observability.EvaluationTrace, rule observability.RuleTrace) {
	ruleKey := s.hashKey(trace.WorkspaceKey, "feature", trace.FeatureKey, "rule", rule.RuleID)
	pipe.HSet(ctx, ruleKey, "priority", rule.Priority)
	s.incrementHash(pipe, ruleKey, "count", 1)
	s.incrementHash(pipe, ruleKey, "duration_sum_ms", rule.DurationMs)
	s.incrementHash(pipe, ruleKey, "expression_duration_sum_ms", rule.ExpressionDurationMs)
	if rule.CompileCacheHit {
		s.incrementHash(pipe, ruleKey, "compile_cache_hit_count", 1)
	}
	if rule.Matched {
		s.incrementHash(pipe, ruleKey, "matched_count", 1)
	}
	_ = s.rdb.Underlying().Expire(ctx, ruleKey, 30*24*time.Hour)
	s.recordRuleExternalCalls(ctx, pipe, trace, rule)
}

func (s *ObservabilityService) recordRuleExternalCalls(ctx context.Context, pipe redis.Pipeliner, trace observability.EvaluationTrace, rule observability.RuleTrace) {
	for _, call := range rule.ExternalCalls {
		callKey := s.hashKey(trace.WorkspaceKey, "feature", trace.FeatureKey, "rule", rule.RuleID, "api", call.APIKey)
		s.incrementHash(pipe, callKey, "count", 1)
		s.incrementHash(pipe, callKey, "duration_sum_ms", call.DurationMs)
		s.incrementHash(pipe, callKey, string(call.CacheStatus)+"_count", 1)
		_ = s.rdb.Underlying().Expire(ctx, callKey, 30*24*time.Hour)
	}
}

func (s *ObservabilityService) persistTrace(trace observability.EvaluationTrace) {
	if s.traceRepo == nil {
		return
	}
	bgCtx := context.Background()
	if trace.WorkspaceKey != "" {
		bgCtx = workspace.WithKey(bgCtx, trace.WorkspaceKey)
	}
	_ = s.traceRepo.Create(bgCtx, &trace)
}

type observabilityRecorder struct {
	service *ObservabilityService
	trace   observability.EvaluationTrace
}

func (r *observabilityRecorder) MarkUsedRedis() {
	r.trace.UsedRedis = true
}

func (r *observabilityRecorder) RecordComponent(component observability.ComponentTrace) {
	r.trace.Components = append(r.trace.Components, component)
	if component.CacheStatus == observability.CacheStatusHit {
		r.trace.UsedRedis = true
	}
}

func (r *observabilityRecorder) StartRule(ruleID string, priority int) observability.RuleRecorder {
	return &observabilityRuleRecorder{parent: r, rule: observability.RuleTrace{RuleID: ruleID, Priority: priority}}
}

func (r *observabilityRecorder) Finalize(resultReason, errorCode string, totalDuration time.Duration) {
	r.trace.ResultReason = resultReason
	r.trace.ErrorCode = errorCode
	r.trace.TotalDurationMs = totalDuration.Milliseconds()
	r.trace.CacheStatus = deriveTraceCacheStatus(r.trace)
	if r.service != nil {
		r.service.recordTrace(context.Background(), r.trace)
	}
}

func (r *observabilityRecorder) Trace() observability.EvaluationTrace { return r.trace }

type observabilityRuleRecorder struct {
	parent *observabilityRecorder
	rule   observability.RuleTrace
}

func (r *observabilityRuleRecorder) RecordExpression(duration time.Duration, compileCacheHit bool) {
	r.rule.ExpressionDurationMs = duration.Milliseconds()
	r.rule.CompileCacheHit = compileCacheHit
}

func (r *observabilityRuleRecorder) RecordExternalCall(call observability.ExternalCallTrace) {
	r.rule.ExternalCalls = append(r.rule.ExternalCalls, call)
	if call.CacheStatus == observability.CacheStatusHit {
		r.parent.trace.UsedRedis = true
	}
}

func (r *observabilityRuleRecorder) Finalize(matched bool, duration time.Duration) {
	r.rule.Matched = matched
	r.rule.DurationMs = duration.Milliseconds()
	r.parent.trace.Rules = append(r.parent.trace.Rules, r.rule)
}

func (s *ObservabilityService) hashKey(workspaceKey string, parts ...string) string {
	parts = append([]string{workspaceKey}, parts...)
	return observabilityKeyPrefix + strings.Join(parts, ":")
}

func (s *ObservabilityService) hashPattern(workspaceKey string, parts ...string) string {
	return s.hashKey(workspaceKey, parts...)
}

func (s *ObservabilityService) incrementHash(pipe redis.Pipeliner, key, field string, value int64) {
	pipe.HIncrBy(context.Background(), key, field, value)
}

func (s *ObservabilityService) scanKeys(ctx context.Context, pattern string) ([]string, error) {
	iter := s.rdb.Underlying().Scan(ctx, 0, pattern, 100).Iterator()
	keys := make([]string, 0)
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

func componentSummaryFromHash(key string, fields map[string]string) observability.ComponentSummary {
	parts := strings.Split(key, ":")
	name := ""
	if len(parts) >= 7 {
		name = parts[6]
	}
	sum := observability.ComponentSummary{
		Name:               name,
		CacheBackend:       observability.CacheBackend(fields["cache_backend"]),
		CacheEnabled:       strings.EqualFold(fields["cache_enabled"], "true"),
		TTLSeconds:         int(parseInt64(fields["ttl_seconds"])),
		Count:              parseInt64(fields["count"]),
		TotalDurationMs:    parseInt64(fields["duration_sum_ms"]),
		HitCount:           parseInt64(fields["hit_count"]),
		MissCount:          parseInt64(fields["miss_count"]),
		DisabledCount:      parseInt64(fields["disabled_count"]),
		ComputedCount:      parseInt64(fields["computed_count"]),
		NotApplicableCount: parseInt64(fields["not_applicable_count"]),
	}
	switch {
	case sum.HitCount > 0:
		sum.CacheStatus = observability.CacheStatusHit
	case sum.MissCount > 0:
		sum.CacheStatus = observability.CacheStatusMiss
	case sum.DisabledCount > 0:
		sum.CacheStatus = observability.CacheStatusDisabled
	case sum.ComputedCount > 0:
		sum.CacheStatus = observability.CacheStatusComputed
	default:
		sum.CacheStatus = observability.CacheStatusNotApplicable
	}
	return sum
}

func ruleSummaryFromHash(key string, fields map[string]string) observability.RuleSummary {
	parts := strings.Split(key, ":")
	ruleID := ""
	if len(parts) >= 7 {
		ruleID = parts[6]
	}
	return observability.RuleSummary{
		RuleID:               ruleID,
		Priority:             int(parseInt64(fields["priority"])),
		Count:                parseInt64(fields["count"]),
		MatchedCount:         parseInt64(fields["matched_count"]),
		TotalDurationMs:      parseInt64(fields["duration_sum_ms"]),
		ExpressionDurationMs: parseInt64(fields["expression_duration_sum_ms"]),
		CompileCacheHitCount: parseInt64(fields["compile_cache_hit_count"]),
	}
}

func externalCallSummaryFromHash(key string, fields map[string]string) observability.ExternalCallSummary {
	parts := strings.Split(key, ":")
	apiKey := ""
	if len(parts) >= 9 {
		apiKey = parts[8]
	}
	return observability.ExternalCallSummary{
		APIKey:             apiKey,
		Count:              parseInt64(fields["count"]),
		TotalDurationMs:    parseInt64(fields["duration_sum_ms"]),
		HitCount:           parseInt64(fields["hit_count"]),
		MissCount:          parseInt64(fields["miss_count"]),
		DisabledCount:      parseInt64(fields["disabled_count"]),
		ComputedCount:      parseInt64(fields["computed_count"]),
		NotApplicableCount: parseInt64(fields["not_applicable_count"]),
	}
}

func shouldPersistTrace(trace observability.EvaluationTrace) bool {
	if trace.ErrorCode != "" {
		return true
	}
	if trace.TotalDurationMs >= 500 {
		return true
	}
	return shouldSampleTrace(trace.RequestID, trace.FeatureKey)
}

func shouldSampleTrace(requestID, featureKey string) bool {
	key := strings.TrimSpace(requestID)
	if key == "" {
		key = strings.TrimSpace(featureKey)
	}
	if key == "" {
		return false
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return h.Sum32()%10 == 0
}

func deriveTraceCacheStatus(trace observability.EvaluationTrace) observability.CacheStatus {
	seen := map[observability.CacheStatus]bool{}
	for _, component := range trace.Components {
		seen[component.CacheStatus] = true
	}
	for _, rule := range trace.Rules {
		for _, call := range rule.ExternalCalls {
			seen[call.CacheStatus] = true
		}
	}
	switch {
	case seen[observability.CacheStatusHit]:
		return observability.CacheStatusHit
	case seen[observability.CacheStatusMiss]:
		return observability.CacheStatusMiss
	case seen[observability.CacheStatusDisabled]:
		return observability.CacheStatusDisabled
	case seen[observability.CacheStatusComputed]:
		return observability.CacheStatusComputed
	default:
		return observability.CacheStatusNotApplicable
	}
}

func parseInt64(raw string) int64 {
	if raw == "" {
		return 0
	}
	var parsed int64
	_, _ = fmt.Sscan(raw, &parsed)
	return parsed
}

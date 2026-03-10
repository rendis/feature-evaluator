package evaluation

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/audit"
	"github.com/rendis/feature-evaluator/internal/domain/experiment"
	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/domain/pack"
	"github.com/rendis/feature-evaluator/internal/domain/segment"
	"github.com/rendis/feature-evaluator/internal/domain/workspace"
	"github.com/rendis/feature-evaluator/internal/engine"
)

const maxBulkConcurrency = 10
const maxAllConcurrency = 20

// SegmentLookupFunc is called for each segment key resolved during evaluation.
type SegmentLookupFunc func(segmentKey string)

// AuthValidator validates incoming eval requests using feature-bound auth profiles.
type AuthValidator interface {
	Validate(ctx context.Context, key string, input map[string]any) (*AuthValidationResult, error)
}

// ExternalApiResolver resolves external API binding calls during expression evaluation.
// It takes the rule's bindings plus a flat env map (for resolving param input paths),
// and returns a map of externalApiKey → passed (bool).
type ExternalApiResolver interface {
	Resolve(ctx context.Context, bindings []feature.ExternalApiBinding, env map[string]any) map[string]bool
}

// AuthValidationResult captures whether the incoming request authenticated for a feature.
type AuthValidationResult struct {
	Authenticated bool
	Attempted     bool
	Cached        bool
	HTTPStatus    int
}

// Service handles feature evaluation logic.
type Service struct {
	featureRepo   feature.Repository
	segmentSvc    *segment.Service
	auditSvc      *audit.Service
	packSvc       *pack.Service
	experimentSvc *experiment.Service
	authValidator       AuthValidator
	externalApiResolver ExternalApiResolver
	engine              *engine.Engine
	onSegmentLookup     SegmentLookupFunc
}

// NewService creates a new evaluation service.
func NewService(
	featureRepo feature.Repository,
	segmentSvc *segment.Service,
	auditSvc *audit.Service,
	eng *engine.Engine,
) *Service {
	return &Service{
		featureRepo: featureRepo,
		segmentSvc:  segmentSvc,
		auditSvc:    auditSvc,
		engine:      eng,
	}
}

// SetPackService sets the pack service for pack-based evaluation.
func (s *Service) SetPackService(packSvc *pack.Service) {
	s.packSvc = packSvc
}

// SetExperimentService sets the experiment service for A/B testing evaluation.
func (s *Service) SetExperimentService(experimentSvc *experiment.Service) {
	s.experimentSvc = experimentSvc
}

// SetAuthValidator sets the feature-bound inbound auth validator.
func (s *Service) SetAuthValidator(authValidator AuthValidator) {
	s.authValidator = authValidator
}

// SetExternalApiResolver sets the resolver for external API binding calls.
func (s *Service) SetExternalApiResolver(resolver ExternalApiResolver) {
	s.externalApiResolver = resolver
}

// SetOnSegmentLookup sets the callback for segment lookup tracking.
func (s *Service) SetOnSegmentLookup(fn SegmentLookupFunc) {
	s.onSegmentLookup = fn
}

// EvalContext holds the context for a single evaluation.
type EvalContext struct {
	Context     map[string]any
	Input       map[string]any
	RequestID   string
	Environment string
}

// Evaluate evaluates a single feature for a given context.
//
//nolint:funlen,gocognit,cyclop // Evaluation is the main orchestration path and is easier to audit as one linear flow.
func (s *Service) Evaluate(ctx context.Context, req Request, evalCtx EvalContext) Result {
	now := time.Now().UTC()

	f, err := s.featureRepo.GetByKey(ctx, req.FeatureKey)
	if err != nil {
		return s.errorResult(req.FeatureKey, now, "FEATURE_NOT_FOUND", err.Error())
	}

	authState, authErr := s.resolveAuthState(ctx, f, evalCtx)
	if authErr != nil {
		return s.errorResult(req.FeatureKey, now, "AUTH_VALIDATION_FAILED", authErr.Error())
	}

	if unauthorizedResult, blocked := s.enforceAccessPolicy(f, authState, evalCtx, now); blocked {
		return unauthorizedResult
	}

	// Feature disabled
	if !f.Enabled {
		return Result{
			FeatureKey:  f.Key,
			Value:       f.DefaultValue,
			ValueType:   string(f.ValueType),
			Environment: evalCtx.Environment,
			Reason:      ReasonFeatureDisabled,
			EvaluatedAt: now,
			Metadata:    f.Metadata,
		}
	}

	// Scheduling: not yet active
	if f.ActiveFrom != nil && now.Before(*f.ActiveFrom) {
		return Result{
			FeatureKey:  f.Key,
			Value:       f.DefaultValue,
			ValueType:   string(f.ValueType),
			Environment: evalCtx.Environment,
			Reason:      ReasonNotYetActive,
			EvaluatedAt: now,
			Metadata:    f.Metadata,
		}
	}

	// Scheduling: expired
	if f.ActiveUntil != nil && now.After(*f.ActiveUntil) {
		return Result{
			FeatureKey:  f.Key,
			Value:       f.DefaultValue,
			ValueType:   string(f.ValueType),
			Environment: evalCtx.Environment,
			Reason:      ReasonExpired,
			EvaluatedAt: now,
			Metadata:    f.Metadata,
		}
	}

	// Environment mismatch
	if len(f.Environments) > 0 && !slices.Contains(f.Environments, evalCtx.Environment) {
		return Result{
			FeatureKey:  f.Key,
			Value:       f.DefaultValue,
			ValueType:   string(f.ValueType),
			Environment: evalCtx.Environment,
			Reason:      ReasonEnvironmentMismatch,
			EvaluatedAt: now,
			Metadata:    f.Metadata,
		}
	}

	// Check for active experiment — overrides rules when running
	userID := extractNamespaceID(evalCtx.Context, "user")
	if result, ok := s.tryExperiment(ctx, f, userID, evalCtx, now); ok {
		return result
	}

	// Pre-scan for inSegment() calls and batch resolve
	segmentKeys := s.collectSegmentKeys(f.Rules)
	tenantID := extractNamespaceID(evalCtx.Context, "tenant")
	segmentResults := s.resolveSegments(ctx, segmentKeys, userID, tenantID)

	// Resolve pack activations
	campusID := extractNamespaceID(evalCtx.Context, "campus")
	programID := extractNamespaceID(evalCtx.Context, "program")
	packKey := s.resolvePackActivation(ctx, req.FeatureKey, tenantID, campusID, programID)

	// Create segment checker function
	segmentChecker := func(segKey string) bool {
		if result, ok := segmentResults[segKey]; ok {
			return result
		}
		return false
	}

	// Iterate rules by priority (ascending)
	sortedRules := sortRulesByPriority(f.Rules)
	for _, rule := range sortedRules {
		preparedInput := PrepareExpressionInput(f.InputContract, evalCtx.Input, authState, evalCtx.Context)
		if _, err := ResolveSegmentSources(ctx, s.segmentSvc, rule.SourceBindings, &preparedInput); err != nil {
			s.logEvalError(ctx, f.Key, rule.ID, err, evalCtx)
			continue
		}
		// Resolve authenticated: true if auth profile validated OR bearer token present on public feature
		authenticated := authState.Authenticated
		if !authenticated && f.AuthProfileKey == "" {
			if tokenPresent, ok := preparedInput.Derived["bearerTokenPresent"].(bool); ok && tokenPresent {
				authenticated = true
			}
		}
		// Build env with a noop externalApi checker first, then resolve with the full env.
		noopExtApi := engine.ExternalApiChecker(func(string) bool { return false })
		env := engine.BuildEnv(
			evalCtx.Context,
			engine.ExpressionInputData{
				Headers:     preparedInput.Headers,
				RequestBody: preparedInput.RequestBody,
				Derived:     preparedInput.Derived,
				Sources:     preparedInput.Sources,
			},
			authenticated,
			segmentChecker,
			noopExtApi,
		)
		// Resolve externalApi bindings with the full env so param input paths work.
		if len(rule.ExternalApiBindings) > 0 && s.externalApiResolver != nil {
			extResults := s.externalApiResolver.Resolve(ctx, rule.ExternalApiBindings, env)
			env["externalApi"] = engine.ExternalApiChecker(func(apiKey string) bool {
				return extResults[apiKey]
			})
		}
		matched, err := s.evaluateRule(&rule, env, authState, evalCtx, f.Key, f.RolloutSalt)
		if err != nil {
			s.logEvalError(ctx, f.Key, rule.ID, err, evalCtx)
			continue
		}
		if matched {
			return Result{
				FeatureKey:  f.Key,
				Value:       rule.Value,
				ValueType:   string(f.ValueType),
				Environment: evalCtx.Environment,
				MatchedRule: &MatchedRule{ID: rule.ID, Name: rule.Name},
				PackGrant:   packKey,
				Segments:    buildSegmentResults(segmentResults),
				Metadata:    mergeMetadata(f.Metadata, rule.Metadata),
				Reason:      ReasonMatchedRule,
				EvaluatedAt: now,
			}
		}
	}

	// No match, return default
	return Result{
		FeatureKey:  f.Key,
		Value:       f.DefaultValue,
		ValueType:   string(f.ValueType),
		Environment: evalCtx.Environment,
		PackGrant:   packKey,
		Segments:    buildSegmentResults(segmentResults),
		Metadata:    f.Metadata,
		Reason:      ReasonDefaultValue,
		EvaluatedAt: now,
	}
}

// BulkEvaluate evaluates multiple features concurrently.
func (s *Service) BulkEvaluate(ctx context.Context, req BulkRequest, evalCtx EvalContext) BulkResult {
	results := make([]Result, len(req.Features))
	sem := make(chan struct{}, maxBulkConcurrency)
	var wg sync.WaitGroup

	for i, featureReq := range req.Features {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, r Request) {
			defer wg.Done()
			defer func() { <-sem }()
			itemCtx := evalCtx
			itemCtx.Context = r.Context
			results[idx] = s.Evaluate(ctx, r, itemCtx)
		}(i, featureReq)
	}

	wg.Wait()
	return BulkResult{Results: results}
}

// EvaluateAll evaluates all enabled features and returns only active ones.
func (s *Service) EvaluateAll(ctx context.Context, req AllRequest, evalCtx EvalContext) AllResult {
	now := time.Now().UTC()

	features, err := s.featureRepo.ListEnabled(ctx)
	if err != nil {
		slog.Error("listing enabled features for eval/active", "error", err)
		return AllResult{
			Features:    []Result{},
			Environment: evalCtx.Environment,
			EvaluatedAt: now,
		}
	}

	// Filter by tags: keep only features that have ALL requested tags
	if len(req.Tags) > 0 {
		filtered := features[:0]
		for _, f := range features {
			if hasAllTags(f.Tags, req.Tags) {
				filtered = append(filtered, f)
			}
		}
		features = filtered
	}

	results := make([]Result, len(features))
	sem := make(chan struct{}, maxAllConcurrency)
	var wg sync.WaitGroup

	for i, f := range features {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, feat feature.Feature) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = s.evaluateFeature(ctx, &feat, evalCtx)
		}(i, f)
	}
	wg.Wait()

	// Collect only active results
	active := make([]Result, 0, len(results))
	for _, r := range results {
		if isActiveResult(r) {
			active = append(active, r)
		}
	}

	return AllResult{
		Features:       active,
		TotalEvaluated: len(features),
		TotalActive:    len(active),
		Environment:    evalCtx.Environment,
		EvaluatedAt:    now,
	}
}

// evaluateFeature evaluates a single pre-loaded feature (skips repo lookup).
//
//nolint:funlen,gocognit // Feature-level evaluation keeps all rule, rollout, and experiment decisions in one place.
func (s *Service) evaluateFeature(ctx context.Context, f *feature.Feature, evalCtx EvalContext) Result {
	now := time.Now().UTC()

	authState, authErr := s.resolveAuthState(ctx, f, evalCtx)
	if authErr != nil {
		return s.errorResult(f.Key, now, "AUTH_VALIDATION_FAILED", authErr.Error())
	}

	if unauthorizedResult, blocked := s.enforceAccessPolicy(f, authState, evalCtx, now); blocked {
		return unauthorizedResult
	}

	// Scheduling: not yet active
	if f.ActiveFrom != nil && now.Before(*f.ActiveFrom) {
		return Result{
			FeatureKey:  f.Key,
			Value:       f.DefaultValue,
			ValueType:   string(f.ValueType),
			Environment: evalCtx.Environment,
			Reason:      ReasonNotYetActive,
			EvaluatedAt: now,
			Metadata:    f.Metadata,
		}
	}

	// Scheduling: expired
	if f.ActiveUntil != nil && now.After(*f.ActiveUntil) {
		return Result{
			FeatureKey:  f.Key,
			Value:       f.DefaultValue,
			ValueType:   string(f.ValueType),
			Environment: evalCtx.Environment,
			Reason:      ReasonExpired,
			EvaluatedAt: now,
			Metadata:    f.Metadata,
		}
	}

	// Environment mismatch
	if len(f.Environments) > 0 && !slices.Contains(f.Environments, evalCtx.Environment) {
		return Result{
			FeatureKey:  f.Key,
			Value:       f.DefaultValue,
			ValueType:   string(f.ValueType),
			Environment: evalCtx.Environment,
			Reason:      ReasonEnvironmentMismatch,
			EvaluatedAt: now,
			Metadata:    f.Metadata,
		}
	}

	// Check for active experiment — overrides rules when running
	userID := extractNamespaceID(evalCtx.Context, "user")
	if result, ok := s.tryExperiment(ctx, f, userID, evalCtx, now); ok {
		return result
	}

	// Pre-scan for inSegment() calls and batch resolve
	segmentKeys := s.collectSegmentKeys(f.Rules)
	tenantID := extractNamespaceID(evalCtx.Context, "tenant")
	segmentResults := s.resolveSegments(ctx, segmentKeys, userID, tenantID)

	// Resolve pack activations
	campusID := extractNamespaceID(evalCtx.Context, "campus")
	programID := extractNamespaceID(evalCtx.Context, "program")
	packKey := s.resolvePackActivation(ctx, f.Key, tenantID, campusID, programID)

	// Create segment checker function
	segmentChecker := func(segKey string) bool {
		if result, ok := segmentResults[segKey]; ok {
			return result
		}
		return false
	}

	// Iterate rules by priority (ascending)
	sortedRules := sortRulesByPriority(f.Rules)
	for _, rule := range sortedRules {
		preparedInput := PrepareExpressionInput(f.InputContract, evalCtx.Input, authState, evalCtx.Context)
		if _, err := ResolveSegmentSources(ctx, s.segmentSvc, rule.SourceBindings, &preparedInput); err != nil {
			s.logEvalError(ctx, f.Key, rule.ID, err, evalCtx)
			continue
		}
		// Resolve authenticated: true if auth profile validated OR bearer token present on public feature
		authenticated := authState.Authenticated
		if !authenticated && f.AuthProfileKey == "" {
			if tokenPresent, ok := preparedInput.Derived["bearerTokenPresent"].(bool); ok && tokenPresent {
				authenticated = true
			}
		}
		noopExtApi := engine.ExternalApiChecker(func(string) bool { return false })
		env := engine.BuildEnv(
			evalCtx.Context,
			engine.ExpressionInputData{
				Headers:     preparedInput.Headers,
				RequestBody: preparedInput.RequestBody,
				Derived:     preparedInput.Derived,
				Sources:     preparedInput.Sources,
			},
			authenticated,
			segmentChecker,
			noopExtApi,
		)
		if len(rule.ExternalApiBindings) > 0 && s.externalApiResolver != nil {
			extResults := s.externalApiResolver.Resolve(ctx, rule.ExternalApiBindings, env)
			env["externalApi"] = engine.ExternalApiChecker(func(apiKey string) bool {
				return extResults[apiKey]
			})
		}
		matched, err := s.evaluateRule(&rule, env, authState, evalCtx, f.Key, f.RolloutSalt)
		if err != nil {
			s.logEvalError(ctx, f.Key, rule.ID, err, evalCtx)
			continue
		}
		if matched {
			return Result{
				FeatureKey:  f.Key,
				Value:       rule.Value,
				ValueType:   string(f.ValueType),
				Environment: evalCtx.Environment,
				MatchedRule: &MatchedRule{ID: rule.ID, Name: rule.Name},
				PackGrant:   packKey,
				Segments:    buildSegmentResults(segmentResults),
				Metadata:    mergeMetadata(f.Metadata, rule.Metadata),
				Reason:      ReasonMatchedRule,
				EvaluatedAt: now,
			}
		}
	}

	// No match, return default
	return Result{
		FeatureKey:  f.Key,
		Value:       f.DefaultValue,
		ValueType:   string(f.ValueType),
		Environment: evalCtx.Environment,
		PackGrant:   packKey,
		Segments:    buildSegmentResults(segmentResults),
		Metadata:    f.Metadata,
		Reason:      ReasonDefaultValue,
		EvaluatedAt: now,
	}
}

// isActiveResult returns true if the result represents an active feature.
func isActiveResult(r Result) bool {
	return r.Reason == ReasonMatchedRule || r.Reason == ReasonDefaultValue || r.Reason == ReasonExperiment
}

func (s *Service) enforceAccessPolicy(
	f *feature.Feature,
	authState AuthValidationResult,
	evalCtx EvalContext,
	now time.Time,
) (Result, bool) {
	policy := f.AccessPolicy
	if policy == "" {
		policy = feature.AccessPolicyRequired
	}
	if policy == feature.AccessPolicyPublic {
		return Result{}, false
	}
	if authState.Attempted && !authState.Authenticated {
		return unauthorizedResult(f, evalCtx.Environment, now, "feature evaluation provided invalid credentials"), true
	}
	if policy != feature.AccessPolicyRequired || authState.Authenticated {
		return Result{}, false
	}
	return unauthorizedResult(f, evalCtx.Environment, now, "feature evaluation requires authentication"), true
}

func unauthorizedResult(f *feature.Feature, environment string, now time.Time, message string) Result {
	return Result{
		FeatureKey:  f.Key,
		Value:       f.DefaultValue,
		ValueType:   string(f.ValueType),
		Environment: environment,
		Reason:      ReasonUnauthorized,
		EvaluatedAt: now,
		Metadata:    f.Metadata,
		Error: &EvalError{
			Code:    "UNAUTHORIZED",
			Message: message,
		},
	}
}

func (s *Service) resolveAuthState(
	ctx context.Context,
	f *feature.Feature,
	evalCtx EvalContext,
) (AuthValidationResult, error) {
	if f.AuthProfileKey == "" {
		return AuthValidationResult{}, nil
	}
	if s.authValidator == nil {
		return AuthValidationResult{}, fmt.Errorf("auth validation requested but validator is not configured")
	}
	result, err := s.authValidator.Validate(ctx, f.AuthProfileKey, evalCtx.Input)
	if err != nil {
		return AuthValidationResult{}, err
	}
	if result == nil {
		return AuthValidationResult{}, nil
	}
	return *result, nil
}

// hasAllTags returns true if featureTags contains every tag in requiredTags.
func hasAllTags(featureTags, requiredTags []string) bool {
	tagSet := make(map[string]struct{}, len(featureTags))
	for _, t := range featureTags {
		tagSet[t] = struct{}{}
	}
	for _, t := range requiredTags {
		if _, ok := tagSet[t]; !ok {
			return false
		}
	}
	return true
}

func (s *Service) evaluateRule(
	rule *feature.Rule,
	env map[string]any,
	authState AuthValidationResult,
	evalCtx EvalContext,
	featureKey,
	rolloutSalt string,
) (bool, error) {
	// Skip disabled rules
	if !rule.Enabled {
		return false, nil
	}

	// Evaluate expression
	result, err := s.engine.CompileAndRun(rule.Expression, env)
	if err != nil {
		return false, fmt.Errorf("evaluating rule %s expression: %w", rule.ID, err)
	}

	matched, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("rule %s expression returned non-boolean: %T", rule.ID, result)
	}

	// Rollout percentage check: nil means 100% (apply to all).
	// Empty userID skips the rollout check (include all anonymous users).
	if matched && rule.RolloutPercentage != nil {
		userID := extractNamespaceID(evalCtx.Context, "user")
		if userID != "" && !isInRollout(featureKey, rolloutSalt, userID, *rule.RolloutPercentage) {
			return false, nil
		}
	}

	return matched, nil
}

// isInRollout determines if a user is included in the rollout using FNV-1a hash.
// Returns true if hash % 100 < percentage. This is deterministic and monotonic:
// increasing the percentage always includes all previously included users.
func isInRollout(featureKey, salt, userID string, percentage int) bool {
	if percentage >= 100 {
		return true
	}
	if percentage <= 0 {
		return false
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(featureKey + ":" + salt + ":" + userID))
	return h.Sum32()%100 < uint32(percentage)
}

func (s *Service) collectSegmentKeys(rules []feature.Rule) []string {
	keySet := make(map[string]bool)
	for _, rule := range rules {
		for _, key := range engine.ExtractInSegmentKeys(rule.Expression) {
			keySet[key] = true
		}
	}
	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	return keys
}

func (s *Service) resolveSegments(ctx context.Context, segmentKeys []string, userID, tenantID string) map[string]bool {
	results := make(map[string]bool, len(segmentKeys))
	if userID == "" {
		// No user in context, skip segment resolution
		for _, key := range segmentKeys {
			results[key] = false
		}
		return results
	}
	for _, key := range segmentKeys {
		if s.onSegmentLookup != nil {
			s.onSegmentLookup(key)
		}
		isMember, err := s.segmentSvc.IsMember(ctx, key, userID, tenantID)
		if err != nil {
			slog.Warn("checking segment membership", "segment", key, "error", err)
			results[key] = false
			continue
		}
		results[key] = isMember
	}
	return results
}

func buildSegmentResults(memberships map[string]bool) []SegmentResult {
	results := make([]SegmentResult, 0, len(memberships))
	for key, member := range memberships {
		results = append(results, SegmentResult{Key: key, Member: member})
	}
	return results
}

// extractNamespaceID extracts a string ID from context[namespace].id.
func extractNamespaceID(ctx map[string]any, namespace string) string {
	if ctx == nil {
		return ""
	}
	ns, ok := ctx[namespace]
	if !ok {
		return ""
	}
	m, ok := ns.(map[string]any)
	if !ok {
		return ""
	}
	v, ok := m["id"]
	if !ok {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func mergeMetadata(featureMeta, ruleMeta map[string]any) map[string]any {
	if featureMeta == nil && ruleMeta == nil {
		return nil
	}
	merged := make(map[string]any)
	for k, v := range featureMeta {
		merged[k] = v
	}
	for k, v := range ruleMeta {
		merged[k] = v
	}
	return merged
}

func sortRulesByPriority(rules []feature.Rule) []feature.Rule {
	sorted := make([]feature.Rule, len(rules))
	copy(sorted, rules)
	slices.SortFunc(sorted, func(a, b feature.Rule) int {
		return a.Priority - b.Priority
	})
	return sorted
}

func (s *Service) errorResult(featureKey string, evalTime time.Time, code, message string) Result {
	return Result{
		FeatureKey:  featureKey,
		Value:       nil,
		Reason:      ReasonError,
		EvaluatedAt: evalTime,
		Error: &EvalError{
			Code:    code,
			Message: message,
		},
	}
}

// resolvePackActivation returns the pack key that grants the feature, if any.
func (s *Service) resolvePackActivation(ctx context.Context, featureKey, tenantID, campusID, programID string) string {
	if s.packSvc == nil {
		return ""
	}

	activeKeys, err := s.packSvc.FindActiveFeatureKeys(ctx, tenantID, campusID, programID)
	if err != nil {
		slog.Warn("resolving pack activations", "featureKey", featureKey, "error", err)
		return ""
	}

	if slices.Contains(activeKeys, featureKey) {
		// Find which pack grants this feature to provide the pack key
		packs, err := s.packSvc.FindByFeatureKey(ctx, featureKey)
		if err != nil {
			slog.Warn("finding pack by feature key", "featureKey", featureKey, "error", err)
			return ""
		}
		for _, p := range packs {
			if p.Enabled {
				return p.Key
			}
		}
		return ""
	}

	return ""
}

// tryExperiment checks if the feature has a running experiment and, if so,
// assigns the user to a variant. Returns (result, true) when experiment applies.
// The experiment overrides normal rules evaluation while running.
func (s *Service) tryExperiment(ctx context.Context, f *feature.Feature, userID string, evalCtx EvalContext, now time.Time) (Result, bool) {
	if s.experimentSvc == nil || userID == "" {
		return Result{}, false
	}

	exp, err := s.experimentSvc.FindRunningByFeatureKey(ctx, f.Key)
	if err != nil {
		slog.Warn("checking running experiment", "featureKey", f.Key, "error", err)
		return Result{}, false
	}
	if exp == nil {
		return Result{}, false
	}

	// Assign user to variant deterministically
	variantKey := experiment.AssignVariant(exp.ID, userID, exp.Variants)

	// Find the variant value
	var value any
	for _, v := range exp.Variants {
		if v.Key == variantKey {
			value = v.Value
			break
		}
	}

	// Record exposure asynchronously (fire-and-forget)
	bgCtx := workspace.WithKey(context.Background(), workspace.KeyFromContext(ctx))
	go func() {
		exposure := &experiment.Exposure{
			ExperimentID: exp.ID,
			FeatureKey:   f.Key,
			UserID:       userID,
			VariantKey:   variantKey,
		}
		if expErr := s.experimentSvc.RecordExposure(bgCtx, exposure); expErr != nil {
			slog.Error("recording experiment exposure", "experimentId", exp.ID, "error", expErr)
		}
	}()

	return Result{
		FeatureKey:  f.Key,
		Value:       value,
		ValueType:   string(f.ValueType),
		Environment: evalCtx.Environment,
		Experiment: &ExperimentInfo{
			ExperimentID: exp.ID,
			VariantKey:   variantKey,
		},
		Metadata:    f.Metadata,
		Reason:      ReasonExperiment,
		EvaluatedAt: now,
	}, true
}

func (s *Service) logEvalError(ctx context.Context, featureKey, ruleID string, err error, evalCtx EvalContext) {
	tenantID := extractNamespaceID(evalCtx.Context, "tenant")
	campusID := extractNamespaceID(evalCtx.Context, "campus")
	programID := extractNamespaceID(evalCtx.Context, "program")
	bgCtx := workspace.WithKey(context.Background(), workspace.KeyFromContext(ctx))

	go func() {
		evalErr := &audit.EvalError{
			FeatureKey: featureKey,
			RuleID:     ruleID,
			ErrorType:  audit.ErrorTypeExpression,
			Message:    err.Error(),
			TenantID:   tenantID,
			CampusID:   campusID,
			ProgramID:  programID,
			RequestID:  evalCtx.RequestID,
		}
		if logErr := s.auditSvc.LogError(bgCtx, evalErr); logErr != nil {
			slog.Error("logging eval error", "error", logErr)
		}
	}()
}

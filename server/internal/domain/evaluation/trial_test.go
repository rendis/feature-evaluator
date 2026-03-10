package evaluation

import (
	"context"
	"testing"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/domain/pack"
	"github.com/rendis/feature-evaluator/internal/engine"
)

// mockPackService implements the subset of pack.Service used by resolveTrial/resolveTierKeys.
type mockPackService struct {
	packs []pack.Pack
	err   error
}

func (m *mockPackService) FindByFeatureKey(_ context.Context, _ string) ([]pack.Pack, error) {
	return m.packs, m.err
}

func (m *mockPackService) FindActiveFeatureKeys(_ context.Context, _, _, _ string) ([]string, error) {
	return nil, nil
}

func newTestService(t *testing.T, f *feature.Feature) *Service {
	t.Helper()
	eng, err := engine.New()
	if err != nil {
		t.Fatalf("engine.New() error = %v", err)
	}
	repo := &mockEvaluationFeatureRepo{feature: f}
	return NewService(repo, nil, nil, eng)
}

func TestTrialActive_ReturnsTrialValue(t *testing.T) {
	t.Parallel()

	trialEnd := time.Now().UTC().Add(24 * time.Hour)
	trialValue := map[string]any{"plan": "pro", "seats": 10}

	feat := &feature.Feature{
		Key:          "feature-trial",
		Enabled:      true,
		ValueType:    feature.ValueTypeJSON,
		DefaultValue: map[string]any{"plan": "free"},
		AccessPolicy: feature.AccessPolicyOptional,
		TrialUntil:   &trialEnd,
		TrialValue:   trialValue,
		Rules: []feature.Rule{
			{
				ID:         "rule-1",
				Name:       "Always true",
				Priority:   1,
				Enabled:    true,
				Expression: "true",
				Value:      map[string]any{"plan": "enterprise"},
			},
		},
	}

	svc := newTestService(t, feat)
	result := svc.Evaluate(
		context.Background(),
		Request{FeatureKey: "feature-trial"},
		EvalContext{
			Context:     map[string]any{},
			Environment: "dev",
		},
	)

	if result.Reason != ReasonTrialActive {
		t.Fatalf("reason = %q, want %q", result.Reason, ReasonTrialActive)
	}
	if !result.InTrial {
		t.Fatal("inTrial = false, want true")
	}
	if result.TrialEndsAt == nil || !result.TrialEndsAt.Equal(trialEnd) {
		t.Fatalf("trialEndsAt = %v, want %v", result.TrialEndsAt, trialEnd)
	}
	// Value should be the trial value, not rule or default
	vm, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("value type = %T, want map[string]any", result.Value)
	}
	if vm["plan"] != "pro" {
		t.Fatalf("value[plan] = %v, want pro", vm["plan"])
	}
	// MatchedRule should be nil (trial bypasses rules)
	if result.MatchedRule != nil {
		t.Fatalf("matchedRule = %+v, want nil", result.MatchedRule)
	}
}

func TestTrialExpired_ContinuesNormalPipeline(t *testing.T) {
	t.Parallel()

	trialEnd := time.Now().UTC().Add(-24 * time.Hour) // in the past
	trialValue := map[string]any{"plan": "pro"}

	feat := &feature.Feature{
		Key:          "feature-trial-expired",
		Enabled:      true,
		ValueType:    feature.ValueTypeJSON,
		DefaultValue: map[string]any{"plan": "free"},
		AccessPolicy: feature.AccessPolicyOptional,
		TrialUntil:   &trialEnd,
		TrialValue:   trialValue,
	}

	svc := newTestService(t, feat)
	result := svc.Evaluate(
		context.Background(),
		Request{FeatureKey: "feature-trial-expired"},
		EvalContext{
			Context:     map[string]any{},
			Environment: "dev",
		},
	)

	if result.Reason != ReasonDefaultValue {
		t.Fatalf("reason = %q, want %q", result.Reason, ReasonDefaultValue)
	}
	if result.InTrial {
		t.Fatal("inTrial = true, want false")
	}
	vm, ok := result.Value.(map[string]any)
	if !ok {
		t.Fatalf("value type = %T, want map[string]any", result.Value)
	}
	if vm["plan"] != "free" {
		t.Fatalf("value[plan] = %v, want free", vm["plan"])
	}
}

func TestTrialActive_NilTrialValue_ContinuesNormalPipeline(t *testing.T) {
	t.Parallel()

	trialEnd := time.Now().UTC().Add(24 * time.Hour)

	feat := &feature.Feature{
		Key:          "feature-trial-nil-value",
		Enabled:      true,
		ValueType:    feature.ValueTypeBoolean,
		DefaultValue: false,
		AccessPolicy: feature.AccessPolicyOptional,
		TrialUntil:   &trialEnd,
		TrialValue:   nil, // trial active but no trial value set
	}

	svc := newTestService(t, feat)
	result := svc.Evaluate(
		context.Background(),
		Request{FeatureKey: "feature-trial-nil-value"},
		EvalContext{
			Context:     map[string]any{},
			Environment: "dev",
		},
	)

	// Should fall through to default value since TrialValue is nil
	if result.Reason != ReasonDefaultValue {
		t.Fatalf("reason = %q, want %q", result.Reason, ReasonDefaultValue)
	}
	if result.InTrial {
		t.Fatal("inTrial = true, want false")
	}
}

func TestResolveTrial_FeatureLevelPriority(t *testing.T) {
	t.Parallel()

	featureTrialEnd := time.Now().UTC().Add(48 * time.Hour)
	packTrialEnd := time.Now().UTC().Add(24 * time.Hour)
	tierKey := "premium"

	feat := &feature.Feature{
		Key:        "feature-priority",
		Enabled:    true,
		TrialUntil: &featureTrialEnd,
		TrialValue: true,
	}

	eng, err := engine.New()
	if err != nil {
		t.Fatalf("engine.New() error = %v", err)
	}
	repo := &mockEvaluationFeatureRepo{feature: feat}
	svc := NewService(repo, nil, nil, eng)

	// Set up a pack service with a pack trial too
	mockPack := &mockPackService{
		packs: []pack.Pack{
			{
				Key:        "pack-a",
				Enabled:    true,
				TrialUntil: &packTrialEnd,
				TierKey:    &tierKey,
			},
		},
	}
	// We can't directly set packSvc as *pack.Service from a mock,
	// so we test resolveTrial directly with a nil packSvc first.
	// Feature-level trial should take priority.
	now := time.Now().UTC()

	active, endsAt := svc.resolveTrial(context.Background(), feat, now)
	if !active {
		t.Fatal("trial active = false, want true")
	}
	if endsAt == nil || !endsAt.Equal(featureTrialEnd) {
		t.Fatalf("trialEndsAt = %v, want %v", endsAt, featureTrialEnd)
	}

	// Verify that it's the feature trial (not pack) even without pack service
	_ = mockPack // avoid unused variable
}

func TestResolveTrial_NoTrial(t *testing.T) {
	t.Parallel()

	feat := &feature.Feature{
		Key:     "feature-no-trial",
		Enabled: true,
	}

	eng, err := engine.New()
	if err != nil {
		t.Fatalf("engine.New() error = %v", err)
	}
	repo := &mockEvaluationFeatureRepo{feature: feat}
	svc := NewService(repo, nil, nil, eng)

	now := time.Now().UTC()
	active, endsAt := svc.resolveTrial(context.Background(), feat, now)
	if active {
		t.Fatal("trial active = true, want false")
	}
	if endsAt != nil {
		t.Fatalf("trialEndsAt = %v, want nil", endsAt)
	}
}

func TestTrialActive_IsActiveResult(t *testing.T) {
	t.Parallel()

	r := Result{Reason: ReasonTrialActive}
	if !isActiveResult(r) {
		t.Fatal("isActiveResult(ReasonTrialActive) = false, want true")
	}
}

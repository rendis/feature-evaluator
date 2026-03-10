package evaluation

import (
	"context"
	"testing"

	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/engine"
)

type mockEvaluationFeatureRepo struct {
	feature *feature.Feature
}

func (m *mockEvaluationFeatureRepo) Create(_ context.Context, _ *feature.Feature) error { return nil }
func (m *mockEvaluationFeatureRepo) GetByKey(_ context.Context, _ string) (*feature.Feature, error) {
	return m.feature, nil
}
func (m *mockEvaluationFeatureRepo) Update(_ context.Context, _ *feature.Feature) error { return nil }
func (m *mockEvaluationFeatureRepo) Delete(_ context.Context, _ string) error           { return nil }
func (m *mockEvaluationFeatureRepo) List(_ context.Context, _ feature.ListParams) (*feature.ListResult, error) {
	return nil, nil
}
func (m *mockEvaluationFeatureRepo) ListEnabled(_ context.Context) ([]feature.Feature, error) {
	return nil, nil
}
func (m *mockEvaluationFeatureRepo) Toggle(_ context.Context, _ string, _ bool, _ string) error {
	return nil
}
func (m *mockEvaluationFeatureRepo) AddRule(_ context.Context, _ string, _ *feature.Rule) error {
	return nil
}
func (m *mockEvaluationFeatureRepo) UpdateRule(_ context.Context, _ string, _ *feature.Rule) error {
	return nil
}
func (m *mockEvaluationFeatureRepo) DeleteRule(_ context.Context, _ string, _ string) error {
	return nil
}
func (m *mockEvaluationFeatureRepo) ReorderRules(_ context.Context, _ string, _ []string) error {
	return nil
}

func TestEvaluate_MatchesRuleByExpressionOnly(t *testing.T) {
	t.Parallel()

	eng, err := engine.New()
	if err != nil {
		t.Fatalf("engine.New() error = %v", err)
	}

	repo := &mockEvaluationFeatureRepo{
		feature: &feature.Feature{
			Key:          "feature-a",
			Enabled:      true,
			ValueType:    feature.ValueTypeBoolean,
			DefaultValue: false,
			AccessPolicy: feature.AccessPolicyOptional,
			RolloutSalt:  "salt-123",
			Rules: []feature.Rule{
				{
					ID:         "rule-1",
					Name:       "Rule A",
					Priority:   1,
					Enabled:    true,
					Expression: "true",
					Value:      true,
				},
			},
		},
	}

	svc := NewService(repo, nil, nil, eng)
	result := svc.Evaluate(
		context.Background(),
		Request{FeatureKey: "feature-a"},
		EvalContext{
			Context: map[string]any{
				"tenant":  map[string]any{"id": "cl"},
				"campus":  map[string]any{"id": "north"},
				"program": map[string]any{"id": "stem"},
			},
			Environment: "dev",
		},
	)

	if result.Reason != ReasonMatchedRule {
		t.Fatalf("reason = %q, want %q", result.Reason, ReasonMatchedRule)
	}
	if result.MatchedRule == nil || result.MatchedRule.ID != "rule-1" {
		t.Fatalf("matched rule = %+v, want rule-1", result.MatchedRule)
	}
	if got, ok := result.Value.(bool); !ok || !got {
		t.Fatalf("value = %#v, want true", result.Value)
	}
}

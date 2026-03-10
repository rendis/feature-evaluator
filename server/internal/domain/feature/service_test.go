package feature

import (
	"context"
	"errors"
	"testing"

	"github.com/rendis/feature-evaluator/pkg/apierror"
)

type mockFeatureRepo struct {
	createCalls int
}

func (m *mockFeatureRepo) Create(_ context.Context, _ *Feature) error {
	m.createCalls++
	return nil
}

func (m *mockFeatureRepo) GetByKey(_ context.Context, _ string) (*Feature, error) {
	return nil, errors.New("not found")
}

func (m *mockFeatureRepo) Update(_ context.Context, _ *Feature) error {
	return nil
}

func (m *mockFeatureRepo) Delete(_ context.Context, _ string) error {
	return nil
}

func (m *mockFeatureRepo) List(_ context.Context, _ ListParams) (*ListResult, error) {
	return nil, nil
}

func (m *mockFeatureRepo) ListEnabled(_ context.Context) ([]Feature, error) {
	return nil, nil
}

func (m *mockFeatureRepo) Toggle(_ context.Context, _ string, _ bool, _ string) error {
	return nil
}

func (m *mockFeatureRepo) AddRule(_ context.Context, _ string, _ *Rule) error {
	return nil
}

func (m *mockFeatureRepo) UpdateRule(_ context.Context, _ string, _ *Rule) error {
	return nil
}

func (m *mockFeatureRepo) DeleteRule(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockFeatureRepo) ReorderRules(_ context.Context, _ string, _ []string) error {
	return nil
}

func TestCreate_Success(t *testing.T) {
	t.Parallel()

	repo := &mockFeatureRepo{}
	svc := NewService(repo)

	f := &Feature{
		Key:          "my-feature",
		Name:         "My Feature",
		ValueType:    ValueTypeBoolean,
		DefaultValue: false,
		AccessPolicy: AccessPolicyPublic,
	}

	if err := svc.Create(context.Background(), f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.createCalls != 1 {
		t.Fatalf("repo.Create called %d times, want 1", repo.createCalls)
	}
	if f.Key != "my_feature" {
		t.Fatalf("Feature key = %q, want %q", f.Key, "my_feature")
	}
}

func TestCreate_InvalidKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
	}{
		{name: "empty", key: ""},
		{name: "punctuation_only", key: "!!!"},
		{name: "single_letter", key: "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := &mockFeatureRepo{}
			svc := NewService(repo)

			err := svc.Create(context.Background(), &Feature{
				Key:          tt.key,
				Name:         "My Feature",
				ValueType:    ValueTypeBoolean,
				DefaultValue: false,
				AccessPolicy: AccessPolicyPublic,
			})

			if err == nil {
				t.Fatal("expected invalid key error")
			}

			var apiErr *apierror.APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("expected apierror.APIError, got %T", err)
			}
			if apiErr.MessageKey != "error.invalidFeatureKey" {
				t.Fatalf("MessageKey = %q, want %q", apiErr.MessageKey, "error.invalidFeatureKey")
			}
		})
	}
}

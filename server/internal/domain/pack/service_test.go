package pack

import (
	"context"
	"errors"
	"testing"

	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

type mockPackRepo struct {
	createCalls    int
	packs          map[string]*Pack
	allInheritance map[string][]string
}

func newMockPackRepo() *mockPackRepo {
	return &mockPackRepo{
		packs:          make(map[string]*Pack),
		allInheritance: make(map[string][]string),
	}
}

func (m *mockPackRepo) Create(_ context.Context, p *Pack) error {
	m.createCalls++
	m.packs[p.Key] = p
	return nil
}

func (m *mockPackRepo) GetByKey(_ context.Context, key string) (*Pack, error) {
	if p, ok := m.packs[key]; ok {
		return p, nil
	}
	return nil, errors.New("not found")
}

func (m *mockPackRepo) Update(_ context.Context, _ *Pack) error {
	return nil
}

func (m *mockPackRepo) Delete(_ context.Context, _ string) error {
	return nil
}

func (m *mockPackRepo) List(_ context.Context) ([]Pack, error) {
	return nil, nil
}

func (m *mockPackRepo) Toggle(_ context.Context, _ string, _ bool, _ string) error {
	return nil
}

func (m *mockPackRepo) FindByFeatureKey(_ context.Context, _ string) ([]Pack, error) {
	return nil, nil
}

func (m *mockPackRepo) ListEnabled(_ context.Context) ([]Pack, error) {
	return nil, nil
}

func (m *mockPackRepo) ListAllInheritance(_ context.Context) (map[string][]string, error) {
	return m.allInheritance, nil
}

type mockActivationRepo struct{}

func (m *mockActivationRepo) Create(_ context.Context, _ *Activation) error { return nil }
func (m *mockActivationRepo) Delete(_ context.Context, _ string, _ TargetType, _ string) error {
	return nil
}
func (m *mockActivationRepo) ListByPack(_ context.Context, _ string) ([]Activation, error) {
	return nil, nil
}
func (m *mockActivationRepo) FindByTarget(_ context.Context, _ TargetType, _ string) ([]Activation, error) {
	return nil, nil
}
func (m *mockActivationRepo) FindActiveFeatureKeys(_ context.Context, _, _, _ string) ([]string, error) {
	return nil, nil
}

type mockFeatureRepo struct{}

func (m *mockFeatureRepo) Create(_ context.Context, _ *feature.Feature) error { return nil }
func (m *mockFeatureRepo) GetByKey(_ context.Context, _ string) (*feature.Feature, error) {
	return &feature.Feature{Key: "existing_feature"}, nil
}
func (m *mockFeatureRepo) Update(_ context.Context, _ *feature.Feature) error { return nil }
func (m *mockFeatureRepo) Delete(_ context.Context, _ string) error           { return nil }
func (m *mockFeatureRepo) List(_ context.Context, _ feature.ListParams) (*feature.ListResult, error) {
	return nil, nil
}
func (m *mockFeatureRepo) ListEnabled(_ context.Context) ([]feature.Feature, error)      { return nil, nil }
func (m *mockFeatureRepo) Toggle(_ context.Context, _ string, _ bool, _ string) error    { return nil }
func (m *mockFeatureRepo) AddRule(_ context.Context, _ string, _ *feature.Rule) error    { return nil }
func (m *mockFeatureRepo) UpdateRule(_ context.Context, _ string, _ *feature.Rule) error { return nil }
func (m *mockFeatureRepo) DeleteRule(_ context.Context, _, _ string) error               { return nil }
func (m *mockFeatureRepo) ReorderRules(_ context.Context, _ string, _ []string) error    { return nil }

type noopCache struct{}

func (noopCache) GetActiveFeatureKeys(_ context.Context, _, _, _ string) ([]string, bool) {
	return nil, false
}
func (noopCache) SetActiveFeatureKeys(_ context.Context, _, _, _ string, _ []string) {}
func (noopCache) InvalidateAll(_ context.Context)                                    {}

func TestCreate_NormalizesKey(t *testing.T) {
	t.Parallel()

	repo := newMockPackRepo()
	svc := NewService(repo, &mockActivationRepo{}, &mockFeatureRepo{}, noopCache{})

	p := &Pack{Key: "My-Pack.2026", Name: "My Pack"}
	if err := svc.Create(context.Background(), p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.createCalls != 1 {
		t.Fatalf("repo.Create called %d times, want 1", repo.createCalls)
	}
	if p.Key != "my_pack_2026" {
		t.Fatalf("Pack key = %q, want %q", p.Key, "my_pack_2026")
	}
}

func TestCreate_InvalidKey(t *testing.T) {
	t.Parallel()

	repo := newMockPackRepo()
	svc := NewService(repo, &mockActivationRepo{}, &mockFeatureRepo{}, noopCache{})

	err := svc.Create(context.Background(), &Pack{Key: "!", Name: "My Pack"})
	if err == nil {
		t.Fatal("expected invalid key error")
	}

	var apiErr *apierror.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected apierror.APIError, got %T", err)
	}
	if apiErr.MessageKey != "error.invalidPackKey" {
		t.Fatalf("MessageKey = %q, want %q", apiErr.MessageKey, "error.invalidPackKey")
	}
}

func TestCreate_WithInheritance_CycleDetected(t *testing.T) {
	t.Parallel()

	repo := newMockPackRepo()
	repo.packs["parent_pack"] = &Pack{Key: "parent_pack", Name: "Parent", Enabled: true, InheritsFrom: []string{"child_pack"}}
	repo.allInheritance = map[string][]string{
		"parent_pack": {"child_pack"},
	}
	svc := NewService(repo, &mockActivationRepo{}, &mockFeatureRepo{}, noopCache{})

	err := svc.Create(context.Background(), &Pack{Key: "child_pack", Name: "Child", InheritsFrom: []string{"parent_pack"}})
	if err == nil {
		t.Fatal("expected cycle detection error")
	}

	var apiErr *apierror.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected apierror.APIError, got %T", err)
	}
	if apiErr.MessageKey != "error.packInheritanceCycle" {
		t.Fatalf("MessageKey = %q, want %q", apiErr.MessageKey, "error.packInheritanceCycle")
	}
}

func TestCreate_WithInheritance_ParentNotFound(t *testing.T) {
	t.Parallel()

	repo := newMockPackRepo()
	svc := NewService(repo, &mockActivationRepo{}, &mockFeatureRepo{}, noopCache{})

	err := svc.Create(context.Background(), &Pack{Key: "child_pack", Name: "Child", InheritsFrom: []string{"nonexistent"}})
	if err == nil {
		t.Fatal("expected parent not found error")
	}
}

func TestCreate_WithInheritance_NoCycle(t *testing.T) {
	t.Parallel()

	repo := newMockPackRepo()
	repo.packs["parent_pack"] = &Pack{Key: "parent_pack", Name: "Parent", Enabled: true}
	repo.allInheritance = map[string][]string{}
	svc := NewService(repo, &mockActivationRepo{}, &mockFeatureRepo{}, noopCache{})

	err := svc.Create(context.Background(), &Pack{Key: "child_pack", Name: "Child", InheritsFrom: []string{"parent_pack"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeduplicate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"empty", []string{}, []string{}},
		{"no duplicates", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"with duplicates", []string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},
		{"all same", []string{"x", "x", "x"}, []string{"x"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := deduplicate(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("deduplicate(%v) = %v, want %v", tc.in, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("deduplicate(%v)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
				}
			}
		})
	}
}

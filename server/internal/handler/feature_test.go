package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	featuredomain "github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
)

type mockFeatureHandlerRepo struct {
	stored     *featuredomain.Feature
	updated    *featuredomain.Feature
	listResult *featuredomain.ListResult
	listParams featuredomain.ListParams
}

func (m *mockFeatureHandlerRepo) Create(_ context.Context, _ *featuredomain.Feature) error {
	return nil
}

func (m *mockFeatureHandlerRepo) GetByKey(_ context.Context, _ string) (*featuredomain.Feature, error) {
	return cloneFeature(m.stored), nil
}

func (m *mockFeatureHandlerRepo) Update(_ context.Context, f *featuredomain.Feature) error {
	m.updated = cloneFeature(f)
	return nil
}

func (m *mockFeatureHandlerRepo) Delete(_ context.Context, _ string) error {
	return nil
}

func (m *mockFeatureHandlerRepo) List(_ context.Context, params featuredomain.ListParams) (*featuredomain.ListResult, error) {
	m.listParams = params
	return m.listResult, nil
}

func (m *mockFeatureHandlerRepo) ListEnabled(_ context.Context) ([]featuredomain.Feature, error) {
	return nil, nil
}

func (m *mockFeatureHandlerRepo) Toggle(_ context.Context, _ string, _ bool, _ string) error {
	return nil
}

func (m *mockFeatureHandlerRepo) AddRule(_ context.Context, _ string, _ *featuredomain.Rule) error {
	return nil
}

func (m *mockFeatureHandlerRepo) UpdateRule(_ context.Context, _ string, _ *featuredomain.Rule) error {
	return nil
}

func (m *mockFeatureHandlerRepo) DeleteRule(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockFeatureHandlerRepo) ReorderRules(_ context.Context, _ string, _ []string) error {
	return nil
}

func cloneFeature(f *featuredomain.Feature) *featuredomain.Feature {
	if f == nil {
		return nil
	}

	cloned := *f

	if f.ActiveFrom != nil {
		activeFrom := *f.ActiveFrom
		cloned.ActiveFrom = &activeFrom
	}
	if f.ActiveUntil != nil {
		activeUntil := *f.ActiveUntil
		cloned.ActiveUntil = &activeUntil
	}
	if f.Environments != nil {
		cloned.Environments = append([]string(nil), f.Environments...)
	}
	if f.Tags != nil {
		cloned.Tags = append([]string(nil), f.Tags...)
	}
	if f.Metadata != nil {
		cloned.Metadata = make(map[string]any, len(f.Metadata))
		for key, value := range f.Metadata {
			cloned.Metadata[key] = value
		}
	}

	return &cloned
}

//nolint:funlen // The test keeps the full request/response fixture in one place for readability.
func TestFeatureHandlerUpdate_PreservesOmittedFields(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	activeFrom := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)
	activeUntil := time.Date(2026, time.March, 2, 12, 0, 0, 0, time.UTC)
	repo := &mockFeatureHandlerRepo{
		stored: &featuredomain.Feature{
			Key:          "external-integration-conf",
			Name:         "Old Name",
			Description:  "Old description",
			Enabled:      true,
			ValueType:    featuredomain.ValueTypeBoolean,
			DefaultValue: false,
			AccessPolicy: featuredomain.AccessPolicyPublic,
			ActiveFrom:   &activeFrom,
			ActiveUntil:  &activeUntil,
			Environments: []string{"dev"},
			Metadata: map[string]any{
				"source": "seed",
			},
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			CreatedBy: "seed@local.dev",
			UpdatedBy: "seed@local.dev",
		},
	}

	handler := NewFeatureHandler(featuredomain.NewService(repo), nil, nil, nil)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("requestId", "req-123")
		c.Set(middleware.CtxUserEmail, "dev@local.dev")
		c.Next()
	})
	router.PUT("/features/:key", handler.Update)

	req := httptest.NewRequest(
		http.MethodPut,
		"/features/external-integration-conf",
		strings.NewReader(`{
			"name":"External integration Conf",
			"description":"Enables the integrations configuration view",
			"defaultValue":true,
			"activeFrom":null,
			"activeUntil":null,
			"environments":["dev","uat","production"]
		}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if repo.updated == nil {
		t.Fatal("expected repo.Update to be called")
	}
	if repo.updated.Enabled != true {
		t.Fatalf("enabled = %v, want true", repo.updated.Enabled)
	}
	if repo.updated.ValueType != featuredomain.ValueTypeBoolean {
		t.Fatalf("valueType = %q, want %q", repo.updated.ValueType, featuredomain.ValueTypeBoolean)
	}
	if repo.updated.Metadata["source"] != "seed" {
		t.Fatalf("metadata[source] = %v, want seed", repo.updated.Metadata["source"])
	}
	if repo.updated.DefaultValue != true {
		t.Fatalf("defaultValue = %v, want true", repo.updated.DefaultValue)
	}
	if repo.updated.ActiveFrom != nil {
		t.Fatalf("activeFrom = %v, want nil", repo.updated.ActiveFrom)
	}
	if repo.updated.ActiveUntil != nil {
		t.Fatalf("activeUntil = %v, want nil", repo.updated.ActiveUntil)
	}
	if got, want := len(repo.updated.Environments), 3; got != want {
		t.Fatalf("environments length = %d, want %d", got, want)
	}
	if repo.updated.UpdatedBy != "dev@local.dev" {
		t.Fatalf("updatedBy = %q, want %q", repo.updated.UpdatedBy, "dev@local.dev")
	}
}

func TestFeatureHandlerListSummary_ReturnsLightweightPayload(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	now := time.Date(2026, time.March, 8, 12, 0, 0, 0, time.UTC)
	repo := &mockFeatureHandlerRepo{
		listResult: &featuredomain.ListResult{
			Data: []featuredomain.Feature{
				{
					ID:           "feat-1",
					Key:          "checkout_v2",
					Name:         "Checkout V2",
					Description:  "new checkout",
					Enabled:      true,
					ValueType:    featuredomain.ValueTypeBoolean,
					AccessPolicy: featuredomain.AccessPolicyPublic,
					Tags:         []string{},
					RuleCount:    3,
					PackCount:    2,
					CreatedAt:    now,
					UpdatedAt:    now,
					CreatedBy:    "tester",
					UpdatedBy:    "tester",
				},
			},
			Page:       1,
			PageSize:   20,
			Total:      1,
			TotalPages: 1,
		},
	}

	handler := NewFeatureHandler(featuredomain.NewService(repo), nil, nil, nil)
	router := gin.New()
	router.GET("/features", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/features?view=summary", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if repo.listParams.View != featuredomain.ListViewSummary {
		t.Fatalf("view = %q, want %q", repo.listParams.View, featuredomain.ListViewSummary)
	}

	body := rec.Body.String()
	for _, expected := range []string{`"packCount":2`, `"ruleCount":3`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response body missing %s: %s", expected, body)
		}
	}
	for _, unexpected := range []string{`"defaultValue"`, `"inputContract"`, `"metadata"`, `"packs"`, `"rolloutSalt"`} {
		if strings.Contains(body, unexpected) {
			t.Fatalf("response body unexpectedly contains %s: %s", unexpected, body)
		}
	}
}

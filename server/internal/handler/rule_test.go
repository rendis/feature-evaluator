package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	featuredomain "github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/engine"
)

type mockRuleHandlerRepo struct {
	addedRule   *featuredomain.Rule
	updatedRule *featuredomain.Rule
}

func (m *mockRuleHandlerRepo) Create(_ context.Context, _ *featuredomain.Feature) error { return nil }
func (m *mockRuleHandlerRepo) GetByKey(_ context.Context, _ string) (*featuredomain.Feature, error) {
	return &featuredomain.Feature{
		Key: "feature-a",
		Rules: []featuredomain.Rule{
			{
				ID:   "rule-123",
				Name: "Existing",
			},
		},
	}, nil
}
func (m *mockRuleHandlerRepo) Update(_ context.Context, _ *featuredomain.Feature) error { return nil }
func (m *mockRuleHandlerRepo) Delete(_ context.Context, _ string) error                 { return nil }
func (m *mockRuleHandlerRepo) List(_ context.Context, _ featuredomain.ListParams) (*featuredomain.ListResult, error) {
	return nil, nil
}
func (m *mockRuleHandlerRepo) ListEnabled(_ context.Context) ([]featuredomain.Feature, error) {
	return nil, nil
}
func (m *mockRuleHandlerRepo) Toggle(_ context.Context, _ string, _ bool, _ string) error { return nil }

func (m *mockRuleHandlerRepo) AddRule(_ context.Context, _ string, rule *featuredomain.Rule) error {
	cloned := *rule
	m.addedRule = &cloned
	return nil
}

func (m *mockRuleHandlerRepo) UpdateRule(_ context.Context, _ string, rule *featuredomain.Rule) error {
	cloned := *rule
	m.updatedRule = &cloned
	return nil
}

func (m *mockRuleHandlerRepo) DeleteRule(_ context.Context, _, _ string) error { return nil }
func (m *mockRuleHandlerRepo) ReorderRules(_ context.Context, _ string, _ []string) error {
	return nil
}

func TestRuleHandlerCreate_WithoutScope(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	eng, err := engine.New()
	if err != nil {
		t.Fatalf("engine.New() error = %v", err)
	}

	repo := &mockRuleHandlerRepo{}
	handler := NewRuleHandler(featuredomain.NewService(repo), nil, nil, nil, eng, nil)
	router := gin.New()
	router.POST("/features/:key/rules", handler.Create)

	req := httptest.NewRequest(
		http.MethodPost,
		"/features/feature-a/rules",
		strings.NewReader(`{"name":"Rule A","priority":1,"enabled":true,"expression":"true","value":true}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if repo.addedRule == nil {
		t.Fatal("expected repo.AddRule to be called")
	}
	if repo.addedRule.Name != "Rule A" {
		t.Fatalf("added rule name = %q, want %q", repo.addedRule.Name, "Rule A")
	}
	if strings.Contains(rec.Body.String(), `"scope"`) {
		t.Fatalf("response body should not include scope: %s", rec.Body.String())
	}
}

func TestRuleHandlerUpdate_WithoutScope(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	eng, err := engine.New()
	if err != nil {
		t.Fatalf("engine.New() error = %v", err)
	}

	repo := &mockRuleHandlerRepo{}
	handler := NewRuleHandler(featuredomain.NewService(repo), nil, nil, nil, eng, nil)
	router := gin.New()
	router.PUT("/features/:key/rules/:ruleId", handler.Update)

	req := httptest.NewRequest(
		http.MethodPut,
		"/features/feature-a/rules/rule-123",
		strings.NewReader(`{"name":"Rule B","priority":2,"enabled":true,"expression":"true","value":"on"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if repo.updatedRule == nil {
		t.Fatal("expected repo.UpdateRule to be called")
	}
	if repo.updatedRule.ID != "rule-123" {
		t.Fatalf("updated rule id = %q, want %q", repo.updatedRule.ID, "rule-123")
	}
	if strings.Contains(rec.Body.String(), `"scope"`) {
		t.Fatalf("response body should not include scope: %s", rec.Body.String())
	}
}

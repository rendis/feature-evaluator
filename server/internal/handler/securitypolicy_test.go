package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/securitypolicy"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
)

func TestSecurityPolicyHandlerGetReturnsSnapshot(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	svc := securitypolicy.NewService(&securityPolicyRepositoryStub{
		managed: &securitypolicy.ManagedPolicy{
			CORSOrigins:           []string{"https://admin.example.com"},
			ExternalAPIAllowHosts: []string{"api2.example.com"},
			UpdatedBy:             "owner@example.com",
		},
	}, securitypolicy.ManagedPolicy{
		CORSOrigins:           []string{"https://console.example.com"},
		ExternalAPIAllowHosts: []string{"api.example.com"},
	})
	if err := svc.Load(t.Context()); err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	router := gin.New()
	router.GET("/security-policy", NewSecurityPolicyHandler(svc).Get)

	req := httptest.NewRequest(http.MethodGet, "/security-policy", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response dto.SecurityPolicyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(response.CORSOrigins.Inherited) != 1 || response.CORSOrigins.Inherited[0] != "https://console.example.com" {
		t.Fatalf("response.CORSOrigins.Inherited = %#v, want inherited origin", response.CORSOrigins.Inherited)
	}
	if len(response.CORSOrigins.Managed) != 1 || response.CORSOrigins.Managed[0] != "https://admin.example.com" {
		t.Fatalf("response.CORSOrigins.Managed = %#v, want managed origin", response.CORSOrigins.Managed)
	}
	if len(response.CORSOrigins.Effective) != 2 {
		t.Fatalf("response.CORSOrigins.Effective = %#v, want union of inherited + managed", response.CORSOrigins.Effective)
	}
}

func TestSecurityPolicyHandlerUpdateReplacesManagedLists(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	repo := &securityPolicyRepositoryStub{}
	svc := securitypolicy.NewService(repo, securitypolicy.ManagedPolicy{
		CORSOrigins:           []string{"https://console.example.com"},
		ExternalAPIAllowHosts: []string{"api.example.com"},
	})
	handler := NewSecurityPolicyHandler(svc)

	router := gin.New()
	router.PUT("/security-policy", func(c *gin.Context) {
		c.Set(middleware.CtxUserEmail, "owner@example.com")
		handler.Update(c)
	})

	req := httptest.NewRequest(http.MethodPut, "/security-policy", strings.NewReader(`{
		"corsOrigins":["https://admin.example.com"],
		"externalApiAllowHosts":["billing.example.com"]
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if repo.managed == nil {
		t.Fatal("repo.managed = nil, want persisted managed policy")
	}
	if repo.managed.UpdatedBy != "owner@example.com" {
		t.Fatalf("repo.managed.UpdatedBy = %q, want %q", repo.managed.UpdatedBy, "owner@example.com")
	}

	var response dto.SecurityPolicyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(response.CORSOrigins.Managed) != 1 || response.CORSOrigins.Managed[0] != "https://admin.example.com" {
		t.Fatalf("response.CORSOrigins.Managed = %#v, want updated managed origins", response.CORSOrigins.Managed)
	}
	if len(response.CORSOrigins.Effective) != 2 {
		t.Fatalf("response.CORSOrigins.Effective = %#v, want inherited + managed union", response.CORSOrigins.Effective)
	}
}

func TestSecurityPolicyHandlerUpdateRejectsInvalidOrigin(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	svc := securitypolicy.NewService(&securityPolicyRepositoryStub{}, securitypolicy.ManagedPolicy{})
	handler := NewSecurityPolicyHandler(svc)

	router := gin.New()
	router.Use(middleware.RequestID())
	router.PUT("/security-policy", handler.Update)

	req := httptest.NewRequest(http.MethodPut, "/security-policy", strings.NewReader(`{
		"corsOrigins":["https://console.example.com/path"]
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	var response dto.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if response.Error == nil || response.Error.MessageKey != "error.invalidSecurityPolicy" {
		t.Fatalf("response.Error = %#v, want invalidSecurityPolicy", response.Error)
	}
}

type securityPolicyRepositoryStub struct {
	managed *securitypolicy.ManagedPolicy
}

func (r *securityPolicyRepositoryStub) Get(_ context.Context) (*securitypolicy.ManagedPolicy, error) {
	if r.managed == nil {
		return nil, nil
	}
	copied := *r.managed
	copied.CORSOrigins = append([]string(nil), r.managed.CORSOrigins...)
	copied.ExternalAPIAllowHosts = append([]string(nil), r.managed.ExternalAPIAllowHosts...)
	return &copied, nil
}

func (r *securityPolicyRepositoryStub) Upsert(_ context.Context, policy *securitypolicy.ManagedPolicy) error {
	copied := *policy
	copied.CORSOrigins = append([]string(nil), policy.CORSOrigins...)
	copied.ExternalAPIAllowHosts = append([]string(nil), policy.ExternalAPIAllowHosts...)
	r.managed = &copied
	return nil
}

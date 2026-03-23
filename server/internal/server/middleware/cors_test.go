package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/securitypolicy"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

func TestCORSAllowsInheritedOrigin(t *testing.T) {
	t.Parallel()

	router := buildCORSTestRouter(securitypolicy.Snapshot{
		CORSOrigins: securitypolicy.ListSnapshot{
			Inherited: []string{"https://console.example.com"},
			Effective: []string{"https://console.example.com"},
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Origin", "https://console.example.com")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://console.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want allowed origin", got)
	}
	if got := rec.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("Vary = %q, want %q", got, "Origin")
	}
}

func TestCORSAllowsManagedOriginPreflight(t *testing.T) {
	t.Parallel()

	router := buildCORSTestRouter(securitypolicy.Snapshot{
		CORSOrigins: securitypolicy.ListSnapshot{
			Managed:   []string{"https://console.example.com"},
			Effective: []string{"https://console.example.com"},
		},
	})
	req := httptest.NewRequest(http.MethodOptions, "/protected", nil)
	req.Header.Set("Origin", "https://console.example.com")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://console.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want allowed origin", got)
	}
}

func TestCORSRejectsDisallowedOrigin(t *testing.T) {
	t.Parallel()

	router := buildCORSTestRouter(securitypolicy.Snapshot{
		CORSOrigins: securitypolicy.ListSnapshot{
			Inherited: []string{"https://console.example.com"},
			Effective: []string{"https://console.example.com"},
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Origin", "https://blocked.example.com")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}

	var response apierror.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if response.Error == nil || response.Error.MessageKey != "error.originNotAllowed" {
		t.Fatalf("response.Error = %#v, want originNotAllowed", response.Error)
	}
}

func TestCORSRejectsDisallowedOriginPreflight(t *testing.T) {
	t.Parallel()

	router := buildCORSTestRouter(securitypolicy.Snapshot{
		CORSOrigins: securitypolicy.ListSnapshot{
			Inherited: []string{"https://console.example.com"},
			Effective: []string{"https://console.example.com"},
		},
	})
	req := httptest.NewRequest(http.MethodOptions, "/protected", nil)
	req.Header.Set("Origin", "https://blocked.example.com")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestCORSSkipsRequestsWithoutOrigin(t *testing.T) {
	t.Parallel()

	router := buildCORSTestRouter(securitypolicy.Snapshot{
		CORSOrigins: securitypolicy.ListSnapshot{
			Inherited: []string{"https://console.example.com"},
			Effective: []string{"https://console.example.com"},
		},
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
}

func buildCORSTestRouter(snapshot securitypolicy.Snapshot) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestID(), CORS(securitypolicy.NewStaticReader(snapshot)))
	router.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	return router
}

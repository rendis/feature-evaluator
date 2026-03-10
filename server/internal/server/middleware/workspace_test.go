package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/workspace"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestWorkspaceResolver_WithHeader(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	var ginKey, ctxKey string
	r.Use(WorkspaceResolver())
	r.GET("/test", func(c *gin.Context) {
		ginKey = GetWorkspaceKey(c)
		ctxKey = workspace.KeyFromContext(c.Request.Context())
		c.String(http.StatusOK, ginKey)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Workspace", "my-ws")
	r.ServeHTTP(w, req)

	if ginKey != "my-ws" {
		t.Errorf("gin context key = %q, want %q", ginKey, "my-ws")
	}
	if ctxKey != "my-ws" {
		t.Errorf("request context key = %q, want %q", ctxKey, "my-ws")
	}
	if w.Body.String() != "my-ws" {
		t.Errorf("response body = %q, want %q", w.Body.String(), "my-ws")
	}
}

func TestWorkspaceResolver_WithoutHeader(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	var ginKey, ctxKey string
	r.Use(WorkspaceResolver())
	r.GET("/test", func(c *gin.Context) {
		ginKey = GetWorkspaceKey(c)
		ctxKey = workspace.KeyFromContext(c.Request.Context())
		c.String(http.StatusOK, ginKey)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if ginKey != workspace.DefaultKey {
		t.Errorf("gin context key = %q, want %q", ginKey, workspace.DefaultKey)
	}
	if ctxKey != workspace.DefaultKey {
		t.Errorf("request context key = %q, want %q", ctxKey, workspace.DefaultKey)
	}
	if w.Body.String() != workspace.DefaultKey {
		t.Errorf("response body = %q, want %q", w.Body.String(), workspace.DefaultKey)
	}
}

func TestGetWorkspaceKey_FromContext(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(CtxWorkspaceKey, "custom-ws")

	got := GetWorkspaceKey(c)

	if got != "custom-ws" {
		t.Errorf("GetWorkspaceKey() = %q, want %q", got, "custom-ws")
	}
}

func TestGetWorkspaceKey_NotSet(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	got := GetWorkspaceKey(c)

	if got != "" {
		t.Errorf("GetWorkspaceKey() = %q, want empty string", got)
	}
}

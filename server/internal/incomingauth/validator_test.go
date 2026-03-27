package incomingauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/rendis/feature-evaluator/internal/domain/authprofile"
)

func TestResolveSourceValueStripsPrefixFromHeader(t *testing.T) {
	t.Parallel()

	value, ok := resolveSourceValue(map[string]any{
		"headers": map[string]any{
			"authorization": "Bearer token-123",
		},
	}, mappingRow{
		SourceType:  "request_header",
		SourceName:  "Authorization",
		StripPrefix: "Bearer ",
	})
	if !ok {
		t.Fatalf("resolveSourceValue() ok = false, want true")
	}
	if got := value.(string); got != "token-123" {
		t.Fatalf("resolveSourceValue() = %q, want %q", got, "token-123")
	}
}

func TestResolveSourceValueKeepsHeaderWhenPrefixDoesNotMatch(t *testing.T) {
	t.Parallel()

	value, ok := resolveSourceValue(map[string]any{
		"headers": map[string]any{
			"authorization": "Token token-123",
		},
	}, mappingRow{
		SourceType:  "request_header",
		SourceName:  "Authorization",
		StripPrefix: "Bearer ",
	})
	if !ok {
		t.Fatalf("resolveSourceValue() ok = false, want true")
	}
	if got := value.(string); got != "Token token-123" {
		t.Fatalf("resolveSourceValue() = %q, want %q", got, "Token token-123")
	}
}

func TestMappingRowsAcceptsNormalizedMappings(t *testing.T) {
	t.Parallel()

	rows := mappingRows([]map[string]any{
		{
			"source": map[string]any{
				"type":        "request_header",
				"name":        "Authorization",
				"stripPrefix": "Bearer",
			},
			"target": map[string]any{
				"type": "header",
				"name": "key",
			},
		},
	})
	if len(rows) != 1 {
		t.Fatalf("mappingRows() len = %d, want 1", len(rows))
	}
	if rows[0].SourceName != "Authorization" || rows[0].TargetName != "key" || rows[0].StripPrefix != "Bearer" {
		t.Fatalf("mappingRows() = %#v, want normalized row", rows[0])
	}
}

func TestValidateDraftCustomSkipsRedisWhenCacheDisabled(t *testing.T) {
	t.Parallel()
	original := os.Getenv("ALLOW_PRIVATE_URLS")
	if err := os.Setenv("ALLOW_PRIVATE_URLS", "true"); err != nil {
		t.Fatalf("Setenv() error = %v", err)
	}
	t.Cleanup(func() {
		if original == "" {
			_ = os.Unsetenv("ALLOW_PRIVATE_URLS")
			return
		}
		_ = os.Setenv("ALLOW_PRIVATE_URLS", original)
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(server.Close)

	validator := NewValidator(nil, nil)
	profile := &authprofile.Profile{
		Type:            authprofile.TypeCustom,
		CacheEnabled:    false,
		CacheTTLSeconds: 999,
		Config: map[string]any{
			"url":    server.URL,
			"method": http.MethodPost,
			"headers": []any{
				map[string]any{
					"source": map[string]any{
						"type": "request_header",
						"name": "X-Test",
					},
					"target": map[string]any{
						"type": "header",
						"name": "X-Forwarded-Test",
					},
				},
			},
			"body":           []any{},
			"requestHeaders": []any{},
			"successRule": map[string]any{
				"type": "any_2xx",
			},
		},
	}

	result, err := validator.ValidateDraft(context.Background(), profile, nil, map[string]any{
		"headers": map[string]any{
			"x-test": "abc",
		},
	})
	if err != nil {
		t.Fatalf("ValidateDraft() error = %v", err)
	}
	if result == nil || !result.Authenticated || !result.Attempted {
		t.Fatalf("ValidateDraft() = %+v, want authenticated attempted result", result)
	}
	if profile.CacheEnabled {
		t.Fatal("expected cache to remain disabled after normalization")
	}
}

package incomingauth

import "testing"

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

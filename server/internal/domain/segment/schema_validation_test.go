package segment

import (
	"errors"
	"testing"

	"github.com/rendis/feature-evaluator/pkg/apierror"
)

func TestValidateSchemaRecords_AcceptsArraySchema(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id":   map[string]any{"type": "integer"},
				"name": map[string]any{"type": "string"},
			},
			"required": []any{"id", "name"},
		},
	}
	records := []map[string]any{
		{"id": 1, "name": "Ada"},
		{"id": 2, "name": "Beto"},
	}

	if _, err := validateSchemaRecords(schema, records); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSchemaRecords_RejectsInvalidRecordAgainstArraySchema(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id":   map[string]any{"type": "integer"},
				"name": map[string]any{"type": "string"},
			},
			"required": []any{"id", "name"},
		},
	}
	records := []map[string]any{
		{"id": 1},
	}

	_, err := validateSchemaRecords(schema, records)
	if err == nil {
		t.Fatal("expected schema mismatch error")
	}

	var apiErr *apierror.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected apierror.APIError, got %T", err)
	}
	if apiErr.MessageKey != "error.segmentSchemaMismatch" {
		t.Fatalf("message key = %q, want %q", apiErr.MessageKey, "error.segmentSchemaMismatch")
	}
}

func TestValidateSchemaRecords_AcceptsObjectSchema(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":   map[string]any{"type": "integer"},
			"name": map[string]any{"type": "string"},
		},
		"required": []any{"id", "name"},
	}
	records := []map[string]any{
		{"id": 1, "name": "Ada"},
	}

	if _, err := validateSchemaRecords(schema, records); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSchemaRecords_NormalizesNullAndDedupesAnyOf(t *testing.T) {
	t.Parallel()

	schema := map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"telefono": map[string]any{
					"anyOf": []any{
						map[string]any{"type": "integer"},
						map[string]any{"type": "string"},
						map[string]any{"type": "integer"},
					},
				},
			},
			"required": []any{"telefono"},
		},
	}
	records := []map[string]any{
		{"telefono": 123.0},
		{"telefono": nil},
		{"telefono": "abc"},
	}

	normalized, err := validateSchemaRecords(schema, records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items := normalized["items"].(map[string]any)
	properties := items["properties"].(map[string]any)
	telefono := properties["telefono"].(map[string]any)

	if _, exists := telefono["anyOf"]; exists {
		t.Fatal("expected telefono schema to collapse anyOf into type union")
	}

	gotTypes := telefono["type"].([]any)
	wantTypes := []any{"integer", "string", "null"}
	if len(gotTypes) != len(wantTypes) {
		t.Fatalf("len(types) = %d, want %d", len(gotTypes), len(wantTypes))
	}

	for idx := range wantTypes {
		if gotTypes[idx] != wantTypes[idx] {
			t.Fatalf("type[%d] = %v, want %v", idx, gotTypes[idx], wantTypes[idx])
		}
	}
}

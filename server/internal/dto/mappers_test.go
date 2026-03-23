package dto

import (
	"testing"

	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/engine"
)

func TestToRuleResponse_BackfillsLegacyBuilderMetadata(t *testing.T) {
	t.Parallel()

	resp := ToRuleResponse(&feature.Rule{
		ID:         "rule-1",
		Name:       "Legacy",
		Expression: `authenticated == true`,
		Metadata: map[string]any{
			"source": "mcp",
		},
	})

	if got := resp.Metadata["source"]; got != "mcp" {
		t.Fatalf("Metadata[source] = %v, want mcp", got)
	}

	builderMetadata, ok := resp.Metadata[engine.ConditionsBuilderMetadataKey].(map[string]any)
	if !ok {
		t.Fatalf("expected %s metadata to be generated", engine.ConditionsBuilderMetadataKey)
	}

	if got := builderMetadata["version"]; got != 2 {
		t.Fatalf("builder version = %v, want 2", got)
	}
}

func TestToRuleResponse_PreservesPersistedBuilderMetadata(t *testing.T) {
	t.Parallel()

	persisted := map[string]any{
		"version": 999,
		"root": map[string]any{
			"id":   "persisted-root",
			"kind": "group",
		},
	}

	resp := ToRuleResponse(&feature.Rule{
		ID:         "rule-2",
		Name:       "Persisted",
		Expression: `authenticated == true`,
		Metadata: map[string]any{
			engine.ConditionsBuilderMetadataKey: persisted,
		},
	})

	if got, ok := resp.Metadata[engine.ConditionsBuilderMetadataKey].(map[string]any); !ok {
		t.Fatalf("expected %s metadata to remain a map", engine.ConditionsBuilderMetadataKey)
	} else if got["version"] != persisted["version"] {
		t.Fatalf("builder metadata was overwritten: got %v, want %v", got, persisted)
	}
}

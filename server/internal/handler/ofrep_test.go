package handler

import (
	"testing"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/evaluation"
	"github.com/rendis/feature-evaluator/internal/dto"
)

func TestMapOFREPContext_FlatContext(t *testing.T) {
	t.Parallel()

	ctx := map[string]any{
		"targetingKey": "user-123",
		"tenantId":     "tenant-1",
		"campusId":     "campus-1",
		"programId":    "program-1",
		"email":        "user@example.com",
		"role":         "student",
	}

	result := mapOFREPContext(ctx)

	// targetingKey -> user.id
	userNS, ok := result["user"].(map[string]any)
	if !ok {
		t.Fatal("expected user namespace to be map[string]any")
	}
	if userNS["id"] != "user-123" {
		t.Errorf("user.id = %v, want user-123", userNS["id"])
	}
	// Extra flat keys go under user namespace
	if userNS["email"] != "user@example.com" {
		t.Errorf("user.email = %v, want user@example.com", userNS["email"])
	}
	if userNS["role"] != "student" {
		t.Errorf("user.role = %v, want student", userNS["role"])
	}

	// tenantId -> tenant.id
	tenantNS, ok := result["tenant"].(map[string]any)
	if !ok {
		t.Fatal("expected tenant namespace")
	}
	if tenantNS["id"] != "tenant-1" {
		t.Errorf("tenant.id = %v, want tenant-1", tenantNS["id"])
	}

	// campusId -> campus.id
	campusNS, ok := result["campus"].(map[string]any)
	if !ok {
		t.Fatal("expected campus namespace")
	}
	if campusNS["id"] != "campus-1" {
		t.Errorf("campus.id = %v, want campus-1", campusNS["id"])
	}

	// programId -> program.id
	programNS, ok := result["program"].(map[string]any)
	if !ok {
		t.Fatal("expected program namespace")
	}
	if programNS["id"] != "program-1" {
		t.Errorf("program.id = %v, want program-1", programNS["id"])
	}
}

func TestMapOFREPContext_NamespacedPassthrough(t *testing.T) {
	t.Parallel()

	ctx := map[string]any{
		"targetingKey": "user-456",
		"user": map[string]any{
			"id":    "user-456",
			"email": "user@test.com",
		},
		"tenant": map[string]any{
			"id":   "tenant-2",
			"name": "Acme Corp",
		},
	}

	result := mapOFREPContext(ctx)

	// Namespaced should pass through directly
	userNS, ok := result["user"].(map[string]any)
	if !ok {
		t.Fatal("expected user namespace")
	}
	if userNS["id"] != "user-456" {
		t.Errorf("user.id = %v, want user-456", userNS["id"])
	}
	if userNS["email"] != "user@test.com" {
		t.Errorf("user.email = %v, want user@test.com", userNS["email"])
	}

	tenantNS, ok := result["tenant"].(map[string]any)
	if !ok {
		t.Fatal("expected tenant namespace")
	}
	if tenantNS["id"] != "tenant-2" {
		t.Errorf("tenant.id = %v, want tenant-2", tenantNS["id"])
	}
	if tenantNS["name"] != "Acme Corp" {
		t.Errorf("tenant.name = %v, want Acme Corp", tenantNS["name"])
	}
}

func TestMapOFREPContext_NamespacedWithTargetingKeyFillsUserID(t *testing.T) {
	t.Parallel()

	// When context is namespaced but user namespace has no "id", targetingKey should fill it
	ctx := map[string]any{
		"targetingKey": "user-789",
		"user": map[string]any{
			"email": "no-id@test.com",
		},
		"tenant": map[string]any{
			"id": "t1",
		},
	}

	result := mapOFREPContext(ctx)

	userNS, ok := result["user"].(map[string]any)
	if !ok {
		t.Fatal("expected user namespace")
	}
	if userNS["id"] != "user-789" {
		t.Errorf("user.id = %v, want user-789 (from targetingKey)", userNS["id"])
	}
	if userNS["email"] != "no-id@test.com" {
		t.Errorf("user.email = %v, want no-id@test.com", userNS["email"])
	}
}

func TestMapOFREPContext_NamespacedUserIDNotOverwritten(t *testing.T) {
	t.Parallel()

	// When user namespace already has an "id", targetingKey should NOT overwrite it
	ctx := map[string]any{
		"targetingKey": "targeting-key-val",
		"user": map[string]any{
			"id": "existing-user-id",
		},
		"tenant": map[string]any{
			"id": "t1",
		},
	}

	result := mapOFREPContext(ctx)

	userNS := result["user"].(map[string]any)
	if userNS["id"] != "existing-user-id" {
		t.Errorf("user.id = %v, want existing-user-id (should not be overwritten by targetingKey)", userNS["id"])
	}
}

func TestMapOFREPContext_EmptyContext(t *testing.T) {
	t.Parallel()

	ctx := map[string]any{}
	result := mapOFREPContext(ctx)

	// With no entries, should get user namespace with no id
	userNS, ok := result["user"].(map[string]any)
	if !ok {
		t.Fatal("expected user namespace even for empty context")
	}
	if len(userNS) != 0 {
		t.Errorf("expected empty user namespace, got %v", userNS)
	}
}

func TestMapOFREPContext_OnlyTargetingKey(t *testing.T) {
	t.Parallel()

	ctx := map[string]any{
		"targetingKey": "solo-user",
	}

	result := mapOFREPContext(ctx)

	userNS := result["user"].(map[string]any)
	if userNS["id"] != "solo-user" {
		t.Errorf("user.id = %v, want solo-user", userNS["id"])
	}
	if len(result) != 1 {
		t.Errorf("expected only user namespace, got %d namespaces", len(result))
	}
}

func TestComputeETag_Deterministic(t *testing.T) {
	t.Parallel()

	flags := []any{
		dto.OFREPEvalResponse{Key: "flag-1", Value: true, Reason: "STATIC"},
		dto.OFREPEvalResponse{Key: "flag-2", Value: "dark", Reason: "TARGETING_MATCH"},
	}

	etag1 := computeETag(flags)
	etag2 := computeETag(flags)

	if etag1 == "" {
		t.Fatal("expected non-empty ETag")
	}
	if etag1 != etag2 {
		t.Errorf("ETag not deterministic: %q != %q", etag1, etag2)
	}
	// Should be a quoted hex string
	if etag1[0] != '"' || etag1[len(etag1)-1] != '"' {
		t.Errorf("ETag should be quoted, got %q", etag1)
	}
}

func TestComputeETag_DifferentInputsDifferentETags(t *testing.T) {
	t.Parallel()

	flags1 := []any{
		dto.OFREPEvalResponse{Key: "flag-1", Value: true, Reason: "STATIC"},
	}
	flags2 := []any{
		dto.OFREPEvalResponse{Key: "flag-1", Value: false, Reason: "STATIC"},
	}

	etag1 := computeETag(flags1)
	etag2 := computeETag(flags2)

	if etag1 == etag2 {
		t.Error("expected different ETags for different inputs")
	}
}

func TestComputeETag_EmptyFlags(t *testing.T) {
	t.Parallel()

	flags := []any{}
	etag := computeETag(flags)

	if etag == "" {
		t.Fatal("expected non-empty ETag even for empty flags")
	}
}

func TestToOFREPReason_AllMappings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input evaluation.Reason
		want  string
	}{
		{evaluation.ReasonMatchedRule, "TARGETING_MATCH"},
		{evaluation.ReasonDefaultValue, "STATIC"},
		{evaluation.ReasonFeatureDisabled, "DISABLED"},
		{evaluation.ReasonNotYetActive, "DISABLED"},
		{evaluation.ReasonExpired, "DISABLED"},
		{evaluation.ReasonEnvironmentMismatch, "DISABLED"},
		{evaluation.ReasonRolloutExcluded, "DEFAULT"},
		{evaluation.ReasonError, "ERROR"},
		{evaluation.Reason("something_unexpected"), "UNKNOWN"},
	}

	for _, tt := range tests {
		got := dto.ToOFREPReason(tt.input)
		if got != tt.want {
			t.Errorf("ToOFREPReason(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestToOFREPReason_AllReasons(t *testing.T) {
	t.Parallel()

	// Exhaustive table covering every defined Reason constant.
	tests := []struct {
		name   string
		reason evaluation.Reason
		want   string
	}{
		{"matched_rule", evaluation.ReasonMatchedRule, "TARGETING_MATCH"},
		{"default_value", evaluation.ReasonDefaultValue, "STATIC"},
		{"experiment", evaluation.ReasonExperiment, "SPLIT"},
		{"feature_disabled", evaluation.ReasonFeatureDisabled, "DISABLED"},
		{"not_yet_active", evaluation.ReasonNotYetActive, "DISABLED"},
		{"expired", evaluation.ReasonExpired, "DISABLED"},
		{"environment_mismatch", evaluation.ReasonEnvironmentMismatch, "DISABLED"},
		{"rollout_excluded", evaluation.ReasonRolloutExcluded, "DEFAULT"},
		{"error", evaluation.ReasonError, "ERROR"},
		{"unknown_reason", evaluation.Reason("totally_unknown"), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := dto.ToOFREPReason(tt.reason)
			if got != tt.want {
				t.Errorf("ToOFREPReason(%q) = %q, want %q", tt.reason, got, tt.want)
			}
		})
	}
}

func TestToOFREPResponse_WithMatchedRule(t *testing.T) {
	t.Parallel()

	result := evaluation.Result{
		FeatureKey: "dark-mode",
		Value:      true,
		Reason:     evaluation.ReasonMatchedRule,
		MatchedRule: &evaluation.MatchedRule{
			ID:   "rule-1",
			Name: "Beta Users",
		},
		Metadata:    map[string]any{"source": "rule"},
		EvaluatedAt: time.Now(),
	}

	resp := dto.ToOFREPResponse(result)

	if resp.Key != "dark-mode" {
		t.Errorf("Key = %q, want dark-mode", resp.Key)
	}
	if resp.Value != true {
		t.Errorf("Value = %v, want true", resp.Value)
	}
	if resp.Variant != "Beta Users" {
		t.Errorf("Variant = %q, want Beta Users", resp.Variant)
	}
	if resp.Reason != "TARGETING_MATCH" {
		t.Errorf("Reason = %q, want TARGETING_MATCH", resp.Reason)
	}
	if resp.Metadata["source"] != "rule" {
		t.Errorf("Metadata[source] = %v, want rule", resp.Metadata["source"])
	}
}

func TestMapOFREPContext_MixedFlatAndNested(t *testing.T) {
	t.Parallel()

	// Context that has both flat keys and nested objects.
	ctx := map[string]any{
		"targetingKey": "user-mixed",
		"user": map[string]any{
			"email": "mixed@test.com",
			"role":  "admin",
		},
		"tenant": map[string]any{
			"id":   "tenant-mixed",
			"name": "Mixed Tenant",
		},
		"flatKey": "should-be-ignored-when-namespaced",
	}

	result := mapOFREPContext(ctx)

	// Namespaced objects should be copied.
	userNS, ok := result["user"].(map[string]any)
	if !ok {
		t.Fatal("expected user namespace to be map[string]any")
	}
	// targetingKey should fill user.id since the user namespace has no "id".
	if userNS["id"] != "user-mixed" {
		t.Errorf("user.id = %v, want user-mixed (from targetingKey)", userNS["id"])
	}
	if userNS["email"] != "mixed@test.com" {
		t.Errorf("user.email = %v, want mixed@test.com", userNS["email"])
	}

	tenantNS, ok := result["tenant"].(map[string]any)
	if !ok {
		t.Fatal("expected tenant namespace")
	}
	if tenantNS["id"] != "tenant-mixed" {
		t.Errorf("tenant.id = %v, want tenant-mixed", tenantNS["id"])
	}

	// Flat non-map keys should be ignored in namespaced mode.
	if _, exists := result["flatKey"]; exists {
		t.Error("flat keys should not appear in result when context has nested objects")
	}
}

func TestToOFREPResponse_DisabledFeature(t *testing.T) {
	t.Parallel()

	result := evaluation.Result{
		FeatureKey:  "hidden-feature",
		Value:       false,
		Reason:      evaluation.ReasonFeatureDisabled,
		MatchedRule: nil,
		Experiment:  nil,
		Metadata:    map[string]any{"disabled": true},
		EvaluatedAt: time.Now(),
	}

	resp := dto.ToOFREPResponse(result)

	if resp.Key != "hidden-feature" {
		t.Errorf("Key = %q, want hidden-feature", resp.Key)
	}
	if resp.Value != false {
		t.Errorf("Value = %v, want false", resp.Value)
	}
	if resp.Reason != "DISABLED" {
		t.Errorf("Reason = %q, want DISABLED", resp.Reason)
	}
	if resp.Variant != "" {
		t.Errorf("Variant = %q, want empty (no matched rule or experiment)", resp.Variant)
	}
}

func TestToOFREPResponse_ExperimentResult(t *testing.T) {
	t.Parallel()

	result := evaluation.Result{
		FeatureKey: "checkout-flow",
		Value:      "variant-b",
		Reason:     evaluation.ReasonExperiment,
		Experiment: &evaluation.ExperimentInfo{
			ExperimentID: "exp-123",
			VariantKey:   "variant-b",
		},
		MatchedRule: &evaluation.MatchedRule{
			ID:   "rule-1",
			Name: "Experiment Rule",
		},
		Metadata:    map[string]any{"experiment": true},
		EvaluatedAt: time.Now(),
	}

	resp := dto.ToOFREPResponse(result)

	if resp.Key != "checkout-flow" {
		t.Errorf("Key = %q, want checkout-flow", resp.Key)
	}
	if resp.Value != "variant-b" {
		t.Errorf("Value = %v, want variant-b", resp.Value)
	}
	if resp.Reason != "SPLIT" {
		t.Errorf("Reason = %q, want SPLIT", resp.Reason)
	}
	// When Experiment is present, variant should come from Experiment.VariantKey, not MatchedRule.Name
	if resp.Variant != "variant-b" {
		t.Errorf("Variant = %q, want variant-b (from Experiment.VariantKey)", resp.Variant)
	}
	if resp.Metadata["experiment"] != true {
		t.Errorf("Metadata[experiment] = %v, want true", resp.Metadata["experiment"])
	}
}

func TestToOFREPResponse_WithoutMatchedRule(t *testing.T) {
	t.Parallel()

	result := evaluation.Result{
		FeatureKey:  "feature-x",
		Value:       "default-val",
		Reason:      evaluation.ReasonDefaultValue,
		MatchedRule: nil,
		Metadata:    nil,
		EvaluatedAt: time.Now(),
	}

	resp := dto.ToOFREPResponse(result)

	if resp.Key != "feature-x" {
		t.Errorf("Key = %q, want feature-x", resp.Key)
	}
	if resp.Value != "default-val" {
		t.Errorf("Value = %v, want default-val", resp.Value)
	}
	if resp.Variant != "" {
		t.Errorf("Variant = %q, want empty", resp.Variant)
	}
	if resp.Reason != "STATIC" {
		t.Errorf("Reason = %q, want STATIC", resp.Reason)
	}
	// nil metadata should be replaced with empty map
	if resp.Metadata == nil {
		t.Error("Metadata should not be nil (should be empty map)")
	}
}

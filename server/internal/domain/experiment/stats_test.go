package experiment

import (
	"fmt"
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// WilsonScore
// ---------------------------------------------------------------------------

func TestWilsonScore_ZeroExposures(t *testing.T) {
	t.Parallel()
	low, high := WilsonScore(0, 0, 1.96)
	if low != 0 || high != 0 {
		t.Fatalf("expected (0, 0), got (%f, %f)", low, high)
	}
}

func TestWilsonScore_AllConversions(t *testing.T) {
	t.Parallel()
	low, high := WilsonScore(100, 100, 1.96)
	if high < 0.95 {
		t.Fatalf("expected high >= 0.95 for 100/100 conversions, got %f", high)
	}
	if low <= 0 {
		t.Fatalf("expected low > 0 for 100/100 conversions, got %f", low)
	}
}

func TestWilsonScore_NoConversions(t *testing.T) {
	t.Parallel()
	low, high := WilsonScore(0, 100, 1.96)
	if low < 0 || low > 0.001 {
		t.Fatalf("expected low near 0, got %f", low)
	}
	if high > 0.05 {
		t.Fatalf("expected high < 0.05 for 0/100 conversions, got %f", high)
	}
}

func TestWilsonScore_NormalCase(t *testing.T) {
	t.Parallel()
	low, high := WilsonScore(50, 100, 1.96)
	if low < 0.39 || low > 0.41 {
		t.Fatalf("expected low ~0.40, got %f", low)
	}
	if high < 0.59 || high > 0.61 {
		t.Fatalf("expected high ~0.60, got %f", high)
	}
}

func TestWilsonScore_ClampedBounds(t *testing.T) {
	t.Parallel()
	cases := []struct {
		conversions int64
		exposures   int64
	}{
		{0, 1},
		{1, 1},
		{0, 100},
		{100, 100},
		{50, 100},
	}
	for _, tc := range cases {
		low, high := WilsonScore(tc.conversions, tc.exposures, 1.96)
		if low < 0 {
			t.Errorf("low < 0 for %d/%d: %f", tc.conversions, tc.exposures, low)
		}
		if high > 1 {
			t.Errorf("high > 1 for %d/%d: %f", tc.conversions, tc.exposures, high)
		}
		if !math.IsNaN(low) && !math.IsNaN(high) && low > high {
			t.Errorf("low > high for %d/%d: %f > %f", tc.conversions, tc.exposures, low, high)
		}
	}
}

// ---------------------------------------------------------------------------
// AssignVariant
// ---------------------------------------------------------------------------

func TestAssignVariant_Deterministic(t *testing.T) {
	t.Parallel()
	variants := []Variant{
		{Key: "control", Weight: 50},
		{Key: "treatment", Weight: 50},
	}
	first := AssignVariant("exp1", "user42", variants)
	for i := 0; i < 100; i++ {
		got := AssignVariant("exp1", "user42", variants)
		if got != first {
			t.Fatalf("iteration %d: expected %q, got %q", i, first, got)
		}
	}
}

func TestAssignVariant_Distribution(t *testing.T) {
	t.Parallel()
	variants := []Variant{
		{Key: "control", Weight: 50},
		{Key: "treatment", Weight: 50},
	}
	counts := map[string]int{}
	n := 10000
	for i := 0; i < n; i++ {
		uid := fmt.Sprintf("user-%d", i)
		key := AssignVariant("exp-dist", uid, variants)
		counts[key]++
	}
	for _, v := range variants {
		pct := float64(counts[v.Key]) / float64(n) * 100
		if pct < 45 || pct > 55 {
			t.Errorf("variant %q: expected ~50%%, got %.1f%%", v.Key, pct)
		}
	}
}

func TestAssignVariant_WeightedDistribution(t *testing.T) {
	t.Parallel()
	variants := []Variant{
		{Key: "a", Weight: 70},
		{Key: "b", Weight: 20},
		{Key: "c", Weight: 10},
	}
	counts := map[string]int{}
	n := 10000
	for i := 0; i < n; i++ {
		uid := fmt.Sprintf("user-%d", i)
		key := AssignVariant("exp-weighted", uid, variants)
		counts[key]++
	}
	expected := map[string]float64{"a": 70, "b": 20, "c": 10}
	for key, expPct := range expected {
		gotPct := float64(counts[key]) / float64(n) * 100
		if gotPct < expPct-5 || gotPct > expPct+5 {
			t.Errorf("variant %q: expected ~%.0f%%, got %.1f%%", key, expPct, gotPct)
		}
	}
}

func TestAssignVariant_FallbackToLast(t *testing.T) {
	t.Parallel()
	variants := []Variant{
		{Key: "control", Weight: 50},
		{Key: "treatment", Weight: 50},
	}
	// Every user must get assigned to some variant.
	for i := 0; i < 1000; i++ {
		uid := fmt.Sprintf("user-fallback-%d", i)
		key := AssignVariant("exp-fb", uid, variants)
		if key != "control" && key != "treatment" {
			t.Fatalf("unexpected variant %q for user %s", key, uid)
		}
	}
}

// ---------------------------------------------------------------------------
// validateVariants
// ---------------------------------------------------------------------------

func TestValidateVariants_Valid(t *testing.T) {
	t.Parallel()
	variants := []Variant{
		{Key: "a", Weight: 60},
		{Key: "b", Weight: 40},
	}
	if err := validateVariants(variants); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateVariants_TooFew(t *testing.T) {
	t.Parallel()
	// validateVariants alone does not check count; that's in Service.Create.
	// But we test with 0 variants to ensure weights sum check triggers.
	variants := []Variant{
		{Key: "only", Weight: 100},
	}
	// With 1 variant summing to 100, validateVariants should pass weight check,
	// but the service layer rejects < 2. Test pure validation here.
	if err := validateVariants(variants); err != nil {
		t.Fatalf("validateVariants with 1 variant should not fail on weights: %v", err)
	}
}

func TestValidateVariants_DuplicateKeys(t *testing.T) {
	t.Parallel()
	variants := []Variant{
		{Key: "a", Weight: 50},
		{Key: "a", Weight: 50},
	}
	err := validateVariants(variants)
	if err == nil {
		t.Fatal("expected error for duplicate keys")
	}
	if got := err.Error(); !contains(got, "duplicate") {
		t.Fatalf("expected 'duplicate' in error, got: %s", got)
	}
}

func TestValidateVariants_BadWeight_Negative(t *testing.T) {
	t.Parallel()
	variants := []Variant{
		{Key: "a", Weight: -10},
		{Key: "b", Weight: 110},
	}
	err := validateVariants(variants)
	if err == nil {
		t.Fatal("expected error for negative weight")
	}
}

func TestValidateVariants_BadWeight_Over100(t *testing.T) {
	t.Parallel()
	variants := []Variant{
		{Key: "a", Weight: 101},
		{Key: "b", Weight: 0},
	}
	err := validateVariants(variants)
	if err == nil {
		t.Fatal("expected error for weight > 100")
	}
}

func TestValidateVariants_WeightSum_Not100(t *testing.T) {
	t.Parallel()
	variants := []Variant{
		{Key: "a", Weight: 30},
		{Key: "b", Weight: 30},
	}
	err := validateVariants(variants)
	if err == nil {
		t.Fatal("expected error for weights not summing to 100")
	}
	if got := err.Error(); !contains(got, "sum to 100") {
		t.Fatalf("expected 'sum to 100' in error, got: %s", got)
	}
}

func TestValidateVariants_EmptyKey(t *testing.T) {
	t.Parallel()
	variants := []Variant{
		{Key: "", Weight: 50},
		{Key: "b", Weight: 50},
	}
	err := validateVariants(variants)
	if err == nil {
		t.Fatal("expected error for empty key")
	}
	if got := err.Error(); !contains(got, "key is required") {
		t.Fatalf("expected 'key is required' in error, got: %s", got)
	}
}

// contains checks if s contains substr (simple helper to avoid importing strings).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

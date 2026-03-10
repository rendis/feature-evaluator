package evaluation

import (
	"testing"
)

func TestIsInRollout_Deterministic(t *testing.T) {
	// Same input should always produce the same result
	for i := 0; i < 100; i++ {
		got := isInRollout("feature-x", "salt123", "user-42", 50)
		want := isInRollout("feature-x", "salt123", "user-42", 50)
		if got != want {
			t.Fatalf("isInRollout is not deterministic: iteration %d", i)
		}
	}
}

func TestIsInRollout_ZeroPercent(t *testing.T) {
	// 0% should exclude all users
	users := []string{"user-1", "user-2", "user-3", "user-100", "user-999"}
	for _, uid := range users {
		if isInRollout("feat", "salt", uid, 0) {
			t.Errorf("isInRollout(0%%) should exclude user %s", uid)
		}
	}
}

func TestIsInRollout_HundredPercent(t *testing.T) {
	// 100% should include all users
	users := []string{"user-1", "user-2", "user-3", "user-100", "user-999"}
	for _, uid := range users {
		if !isInRollout("feat", "salt", uid, 100) {
			t.Errorf("isInRollout(100%%) should include user %s", uid)
		}
	}
}

func TestIsInRollout_Monotonic(t *testing.T) {
	// Increasing percentage should always include previously included users.
	// Test with many users to ensure monotonicity.
	featureKey := "my-feature"
	salt := "a1b2c3d4e5f6a7b8"

	for i := 0; i < 200; i++ {
		uid := "user-" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		lastIncluded := -1
		for pct := 0; pct <= 100; pct++ {
			included := isInRollout(featureKey, salt, uid, pct)
			if included && lastIncluded == -1 {
				lastIncluded = pct
			}
			if !included && lastIncluded != -1 {
				t.Fatalf("monotonicity violated for user %s: included at %d%% but excluded at %d%%",
					uid, lastIncluded, pct)
			}
		}
	}
}

func TestIsInRollout_Distribution(t *testing.T) {
	// At 50%, roughly half the users should be included (within tolerance).
	featureKey := "dist-test"
	salt := "randomsalt123456"
	total := 10000
	included := 0

	for i := 0; i < total; i++ {
		uid := "user-" + string(rune(i/256)) + string(rune(i%256))
		if isInRollout(featureKey, salt, uid, 50) {
			included++
		}
	}

	ratio := float64(included) / float64(total)
	if ratio < 0.40 || ratio > 0.60 {
		t.Errorf("at 50%% rollout, expected ~50%% inclusion but got %.1f%% (%d/%d)",
			ratio*100, included, total)
	}
}

func TestIsInRollout_DifferentSaltsProduceDifferentResults(t *testing.T) {
	// Different salts should (likely) produce different rollout assignments
	uid := "user-test"
	pct := 50
	result1 := isInRollout("feat", "salt-aaa", uid, pct)
	result2 := isInRollout("feat", "salt-bbb", uid, pct)

	// With different salts, the same user may get different results.
	// We can't guarantee they differ for a single user, but we can check
	// that across many users, the assignments differ.
	differ := 0
	for i := 0; i < 1000; i++ {
		u := "u-" + string(rune(i/256)) + string(rune(i%256))
		r1 := isInRollout("feat", "salt-aaa", u, pct)
		r2 := isInRollout("feat", "salt-bbb", u, pct)
		if r1 != r2 {
			differ++
		}
	}
	// Expect some difference (not 0 and not all 1000)
	if differ == 0 {
		t.Error("different salts produced identical results for all users")
	}
	_ = result1
	_ = result2
}

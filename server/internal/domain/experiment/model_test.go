package experiment

import "testing"

func TestExperimentNormalizeCacheConfig(t *testing.T) {
	t.Parallel()

	exp := &Experiment{
		LookupCacheEnabled:    true,
		LookupCacheTTLSeconds: 0,
	}

	exp.NormalizeCacheConfig()

	if exp.LookupCacheTTLSeconds != defaultExperimentLookupCacheTTLSeconds {
		t.Fatalf("LookupCacheTTLSeconds = %d, want %d", exp.LookupCacheTTLSeconds, defaultExperimentLookupCacheTTLSeconds)
	}

	exp.LookupCacheEnabled = false
	exp.LookupCacheTTLSeconds = 42
	exp.NormalizeCacheConfig()
	if exp.LookupCacheTTLSeconds != 0 {
		t.Fatalf("LookupCacheTTLSeconds = %d, want 0 when disabled", exp.LookupCacheTTLSeconds)
	}
}

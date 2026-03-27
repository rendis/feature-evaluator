package feature

import "testing"

func TestNormalizeCacheConfig(t *testing.T) {
	t.Parallel()

	feature := &Feature{
		EvalCacheEnabled:    true,
		EvalCacheTTLSeconds: 0,
		Rules: []Rule{
			{
				ExternalAPIBindings: []ExternalAPIBinding{
					{CacheEnabled: true, CacheTTL: 0},
					{CacheEnabled: true, CacheTTL: 9999},
					{CacheEnabled: false, CacheTTL: 90},
				},
			},
		},
	}

	feature.NormalizeCacheConfig()

	if feature.EvalCacheTTLSeconds != defaultFeatureEvalCacheTTL {
		t.Fatalf("EvalCacheTTLSeconds = %d, want %d", feature.EvalCacheTTLSeconds, defaultFeatureEvalCacheTTL)
	}
	if got := feature.Rules[0].ExternalAPIBindings[0].CacheTTL; got != defaultExternalBindingCacheTTL {
		t.Fatalf("binding[0].CacheTTL = %d, want %d", got, defaultExternalBindingCacheTTL)
	}
	if got := feature.Rules[0].ExternalAPIBindings[1].CacheTTL; got != maxCacheTTLSeconds {
		t.Fatalf("binding[1].CacheTTL = %d, want %d", got, maxCacheTTLSeconds)
	}
	if got := feature.Rules[0].ExternalAPIBindings[2].CacheTTL; got != 0 {
		t.Fatalf("binding[2].CacheTTL = %d, want 0", got)
	}
}

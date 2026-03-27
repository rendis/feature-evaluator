package authprofile

import "testing"

func TestTypeValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value Type
		valid bool
	}{
		{name: "api_key", value: TypeAPIKey, valid: true},
		{name: "oidc_standard", value: TypeOIDCStandard, valid: true},
		{name: "custom", value: TypeCustom, valid: true},
		{name: "oauth2_passthrough removed", value: Type("oauth2_passthrough"), valid: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.value.Valid(); got != tt.valid {
				t.Fatalf("Type.Valid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestProfileNormalizeCacheConfig(t *testing.T) {
	t.Parallel()

	custom := &Profile{
		Type:            TypeCustom,
		CacheEnabled:    true,
		CacheTTLSeconds: 1,
	}
	custom.Normalize()
	if !custom.CacheEnabled {
		t.Fatal("expected custom profile cache to remain enabled")
	}
	if custom.CacheTTLSeconds != 30 {
		t.Fatalf("CacheTTLSeconds = %d, want 30", custom.CacheTTLSeconds)
	}

	apiKey := &Profile{
		Type:            TypeAPIKey,
		CacheEnabled:    true,
		CacheTTLSeconds: 120,
	}
	apiKey.Normalize()
	if apiKey.CacheEnabled {
		t.Fatal("expected api_key cache to be disabled")
	}
	if apiKey.CacheTTLSeconds != 0 {
		t.Fatalf("CacheTTLSeconds = %d, want 0", apiKey.CacheTTLSeconds)
	}
}

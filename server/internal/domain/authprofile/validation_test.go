package authprofile

import "testing"

func TestValidateProfileOIDCStandardRequiresIssuerAndAudience(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  map[string]any
		wantErr bool
	}{
		{
			name: "valid",
			config: map[string]any{
				"issuer":   "https://issuer.example.com/",
				"audience": "feature-evaluator",
			},
			wantErr: false,
		},
		{
			name: "missing issuer",
			config: map[string]any{
				"audience": "feature-evaluator",
			},
			wantErr: true,
		},
		{
			name: "missing audience",
			config: map[string]any{
				"issuer": "https://issuer.example.com",
			},
			wantErr: true,
		},
		{
			name: "legacy url field rejected",
			config: map[string]any{
				"issuer":   "https://issuer.example.com",
				"audience": "feature-evaluator",
				"url":      "https://issuer.example.com/oauth/validate",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			profile := &Profile{
				Type:   TypeOIDCStandard,
				Config: tt.config,
			}
			err := ValidateProfile(profile, nil, true, false)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateProfile() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				if got := profile.Config["issuer"]; got != "https://issuer.example.com" {
					t.Fatalf("issuer = %v, want https://issuer.example.com", got)
				}
				if got := profile.Config["audience"]; got != "feature-evaluator" {
					t.Fatalf("audience = %v, want feature-evaluator", got)
				}
				if profile.CacheTTLSeconds != 0 {
					t.Fatalf("CacheTTLSeconds = %d, want 0", profile.CacheTTLSeconds)
				}
			}
		})
	}
}

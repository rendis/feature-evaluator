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

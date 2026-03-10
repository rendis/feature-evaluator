package resourcekey

import "testing"

func TestNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "snake_case", in: "Mi Feature.Á_Test@2026---", want: "mi_feature_a_test_2026"},
		{name: "prefixes_digit", in: "2026", want: "k_2026"},
		{name: "empty_stays_empty", in: "   ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Normalize(tt.in); got != tt.want {
				t.Fatalf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
		want bool
	}{
		{name: "valid", key: "my_feature", want: true},
		{name: "hyphen", key: "my-feature", want: false},
		{name: "digit_prefix", key: "2026_feature", want: false},
		{name: "single_letter", key: "a", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := IsValid(tt.key); got != tt.want {
				t.Fatalf("IsValid(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

package resourcekey

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const (
	Prefix = "k_"
	MaxLen = 128
)

var (
	pattern     = regexp.MustCompile(`^[a-z][a-z0-9_]{1,127}$`)
	nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)
)

// Normalize converts arbitrary input into the canonical snake_case key shape.
func Normalize(raw string) string {
	decomposed := norm.NFD.String(strings.ToLower(strings.TrimSpace(raw)))
	var b strings.Builder
	lastUnderscore := false

	for _, r := range decomposed {
		switch {
		case unicode.Is(unicode.Mn, r):
			continue
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastUnderscore = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastUnderscore = false
		default:
			if !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		}
	}

	normalized := nonAlphaNum.ReplaceAllString(b.String(), "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		return ""
	}
	if normalized[0] < 'a' || normalized[0] > 'z' {
		normalized = Prefix + normalized
	}
	if len(normalized) > MaxLen {
		normalized = strings.TrimRight(normalized[:MaxLen], "_")
	}
	return strings.Trim(normalized, "_")
}

// IsValid reports whether a key is already in canonical snake_case form.
func IsValid(key string) bool {
	return pattern.MatchString(key)
}

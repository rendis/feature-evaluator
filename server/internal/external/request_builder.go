package external

import (
	"strings"
)

// ResolveValue extracts a value from a map using dot notation.
func ResolveValue(path string, context map[string]any) any {
	if path == "" {
		return nil
	}
	parts := strings.Split(path, ".")
	var current any = context

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current, ok = m[part]
		if !ok {
			return nil
		}
	}
	return current
}

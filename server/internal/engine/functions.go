package engine

import (
	"fmt"
	"strings"
	"time"
)

// SegmentChecker is a function that checks segment membership.
// It is injected at evaluation time with pre-resolved memberships.
type SegmentChecker func(segmentKey string) bool

// ExternalAPIChecker is a function that checks if an external API call passes.
// It is injected at evaluation time with pre-resolved results.
type ExternalAPIChecker func(apiKey string) bool

// BuiltinFunctions returns the custom functions available in expressions.
func BuiltinFunctions(segmentChecker SegmentChecker, externalAPIChecker ExternalAPIChecker) map[string]any {
	funcs := map[string]any{
		"now": func() time.Time { return time.Now().UTC() },
		"dateBefore": func(dateValue, refValue any) bool {
			d, err := parseDateAny(dateValue)
			if err != nil {
				return false
			}
			ref, err := parseDateAny(refValue)
			if err != nil {
				return false
			}
			return d.Before(ref)
		},
		"dateAfter": func(dateValue, refValue any) bool {
			d, err := parseDateAny(dateValue)
			if err != nil {
				return false
			}
			ref, err := parseDateAny(refValue)
			if err != nil {
				return false
			}
			return d.After(ref)
		},
		"contains": func(value, needle any) bool {
			return strings.Contains(fmt.Sprint(value), fmt.Sprint(needle))
		},
		"startsWith": func(value, prefix any) bool {
			return strings.HasPrefix(fmt.Sprint(value), fmt.Sprint(prefix))
		},
		"endsWith": func(value, suffix any) bool {
			return strings.HasSuffix(fmt.Sprint(value), fmt.Sprint(suffix))
		},
	}
	if segmentChecker != nil {
		funcs["inSegment"] = segmentChecker
	}
	if externalAPIChecker != nil {
		funcs["externalApi"] = externalAPIChecker
	}
	return funcs
}

func parseDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		t, err := time.Parse(f, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}

func parseDateAny(value any) (time.Time, error) {
	switch typed := value.(type) {
	case time.Time:
		return typed, nil
	case string:
		return parseDate(typed)
	default:
		return time.Time{}, fmt.Errorf("unsupported date value: %T", value)
	}
}

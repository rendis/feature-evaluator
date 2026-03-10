package postgres

import "time"

func parseCreatedAtFilters(from, to string) (*time.Time, *time.Time) {
	var fromPtr *time.Time
	var toPtr *time.Time

	if from != "" {
		if parsed, err := time.Parse(time.RFC3339, from); err == nil {
			value := parsed.UTC()
			fromPtr = &value
		}
	}

	if to != "" {
		if parsed, err := time.Parse(time.RFC3339, to); err == nil {
			value := parsed.UTC()
			toPtr = &value
		}
	}

	return fromPtr, toPtr
}

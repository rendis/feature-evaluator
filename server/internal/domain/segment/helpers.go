package segment

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/rendis/feature-evaluator/internal/domain/resourcekey"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// NormalizeKey forces a segment key into a safe snake_case namespace.
func NormalizeKey(raw string) string {
	return resourcekey.Normalize(raw)
}

// ValidateNormalizedKey checks the persisted key shape.
func ValidateNormalizedKey(key string) error {
	if !resourcekey.IsValid(key) {
		return apierror.NewBadRequest(
			fmt.Sprintf("invalid segment key format: %s", key),
			"error.invalidSegmentKey",
		)
	}
	return nil
}

// ExtractRecordKey resolves and canonicalizes the configured record key from a JSON object.
func ExtractRecordKey(record map[string]any, path string) (string, error) {
	value, ok := ResolvePath(record, path)
	if !ok {
		return "", apierror.NewBadRequest(
			fmt.Sprintf("record key path %q not found", path),
			"error.segmentRecordKeyMissing",
		)
	}

	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return "", apierror.NewBadRequest("record key cannot be empty", "error.segmentRecordKeyInvalid")
		}
		return v, nil
	case int:
		return strconv.Itoa(v), nil
	case int8:
		return strconv.FormatInt(int64(v), 10), nil
	case int16:
		return strconv.FormatInt(int64(v), 10), nil
	case int32:
		return strconv.FormatInt(int64(v), 10), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case uint:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(v), 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	default:
		return "", apierror.NewBadRequest(
			fmt.Sprintf("record key path %q must resolve to string or number", path),
			"error.segmentRecordKeyInvalid",
		)
	}
}

// ResolvePath navigates a dot-separated path over a nested JSON object.
func ResolvePath(record map[string]any, path string) (any, bool) {
	if path == "" {
		return nil, false
	}

	current := any(record)
	parts := strings.Split(path, ".")
	for _, part := range parts {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, exists := obj[part]
		if !exists {
			return nil, false
		}
		current = next
	}

	return current, true
}

// DerivePreviewFields collects a few scalar leaf paths for tabular preview.
func DerivePreviewFields(records []map[string]any) []string {
	if len(records) == 0 {
		return []string{}
	}

	fields := make([]string, 0, 4)
	var walk func(prefix string, value any)
	walk = func(prefix string, value any) {
		if len(fields) >= 4 {
			return
		}
		switch v := value.(type) {
		case map[string]any:
			keys := make([]string, 0, len(v))
			for key := range v {
				keys = append(keys, key)
			}
			slices.Sort(keys)
			for _, key := range keys {
				next := key
				if prefix != "" {
					next = prefix + "." + key
				}
				walk(next, v[key])
				if len(fields) >= 4 {
					return
				}
			}
		case string, float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			fields = append(fields, prefix)
		}
	}

	walk("", records[0])
	return fields
}

// NewDatasetVersion returns a new active dataset version identifier.
func NewDatasetVersion() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}

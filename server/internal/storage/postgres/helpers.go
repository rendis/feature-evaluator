package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rendis/feature-evaluator/internal/domain/workspace"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

func wsKey(ctx context.Context) string {
	return workspace.KeyFromContext(ctx)
}

func newID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate uuid v7: %w", err)
	}

	return id.String(), nil
}

func parseUUID(id string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, apierror.NewBadRequest("invalid id", "error.invalidId")
	}

	return parsed, nil
}

func jsonBytes(value any, fallback string) ([]byte, error) {
	if value == nil {
		return []byte(fallback), nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if string(data) == "null" {
		return []byte(fallback), nil
	}

	return data, nil
}

func decodeJSON(data []byte, target any) error {
	if len(data) == 0 {
		return nil
	}

	return json.Unmarshal(data, target)
}

func sanitizeSearch(value string) string {
	const maxLen = 200
	value = strings.TrimSpace(value)
	if len(value) > maxLen {
		return value[:maxLen]
	}

	return value
}

func parseUUIDStrings(values []string) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		parsed, err := uuid.Parse(value)
		if err != nil {
			return nil, err
		}
		ids = append(ids, parsed)
	}

	return ids, nil
}

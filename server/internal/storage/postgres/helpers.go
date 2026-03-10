package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func nullableString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return &value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}

	return *value
}

func sanitizeSearch(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if len(value) > maxLen {
		return value[:maxLen]
	}

	return value
}

func normalizeStringSlice(values []string) []string {
	if values == nil {
		return []string{}
	}

	cloned := slices.Clone(values)
	slices.Sort(cloned)
	return cloned
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

func closeRows(rows pgx.Rows, err *error) {
	if rows == nil {
		return
	}

	rows.Close()
	if rowsErr := rows.Err(); rowsErr != nil && *err == nil {
		*err = rowsErr
	}
}

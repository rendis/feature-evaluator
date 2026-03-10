package feature

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/rendis/feature-evaluator/internal/domain/resourcekey"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

var reservedExpressionNamespaces = map[string]struct{}{
	"authenticated": {},
	"headers":       {},
	"requestBody":   {},
	"derived":       {},
	"user":          {},
	"tenant":        {},
	"campus":        {},
	"program":       {},
}

func normalizeInputContract(contract InputContract) (InputContract, error) {
	headers := make([]InputHeader, 0, len(contract.Headers))
	seen := make(map[string]struct{}, len(contract.Headers))

	for _, header := range contract.Headers {
		headerName := strings.TrimSpace(header.HeaderName)
		if headerName == "" {
			return InputContract{}, apierror.NewBadRequest(
				"input header name is required",
				"error.invalidFeatureInputContract",
			)
		}
		expressionKey := strings.TrimSpace(header.ExpressionKey)
		if expressionKey == "" {
			expressionKey = resourcekey.Normalize(headerName)
		}
		if !resourcekey.IsValid(expressionKey) {
			return InputContract{}, apierror.NewBadRequest(
				fmt.Sprintf("invalid input header expressionKey %q", expressionKey),
				"error.invalidFeatureInputContract",
			)
		}
		if _, reserved := reservedExpressionNamespaces[expressionKey]; reserved {
			return InputContract{}, apierror.NewBadRequest(
				fmt.Sprintf("header expressionKey %q is reserved", expressionKey),
				"error.invalidFeatureInputContract",
			)
		}
		if _, exists := seen[expressionKey]; exists {
			return InputContract{}, apierror.NewBadRequest(
				fmt.Sprintf("duplicate header expressionKey %q", expressionKey),
				"error.invalidFeatureInputContract",
			)
		}
		seen[expressionKey] = struct{}{}

		if header.Type == "" {
			header.Type = InputValueTypeString
		}
		if !header.Type.Valid() {
			return InputContract{}, apierror.NewBadRequest(
				fmt.Sprintf("invalid input header type %q", header.Type),
				"error.invalidFeatureInputContract",
			)
		}

		headers = append(headers, InputHeader{
			HeaderName:    headerName,
			ExpressionKey: expressionKey,
			Label:         strings.TrimSpace(header.Label),
			Type:          header.Type,
			Required:      header.Required,
			Description:   strings.TrimSpace(header.Description),
		})
	}

	normalized := InputContract{
		Headers: headers,
	}
	if len(contract.RequestBodyExample) > 0 {
		normalized.RequestBodyExample = contract.RequestBodyExample
		normalized.RequestBodySchema = inferSchemaFromExample(contract.RequestBodyExample)
	}
	return normalized, nil
}

func validateRuleSourceBindings(bindings SourceBindings) error {
	seen := make(map[string]struct{}, len(bindings.Segments))
	for _, binding := range bindings.Segments {
		segmentKey := resourcekey.Normalize(binding.SegmentKey)
		if !resourcekey.IsValid(segmentKey) {
			return apierror.NewBadRequest(
				fmt.Sprintf("invalid segment key %q in sourceBindings", binding.SegmentKey),
				"error.invalidRuleSourceBindings",
			)
		}
		if _, reserved := reservedExpressionNamespaces[segmentKey]; reserved {
			return apierror.NewBadRequest(
				fmt.Sprintf("segment key %q conflicts with a reserved namespace", segmentKey),
				"error.invalidRuleSourceBindings",
			)
		}
		if _, exists := seen[segmentKey]; exists {
			return apierror.NewBadRequest(
				fmt.Sprintf("duplicate segment binding %q", segmentKey),
				"error.invalidRuleSourceBindings",
			)
		}
		seen[segmentKey] = struct{}{}
		if strings.TrimSpace(binding.LookupPath) == "" {
			return apierror.NewBadRequest(
				fmt.Sprintf("lookupPath is required for segment %q", segmentKey),
				"error.invalidRuleSourceBindings",
			)
		}
	}

	return nil
}

func inferSchemaFromExample(value any) map[string]any {
	return inferSchemaNode(value)
}

func inferSchemaNode(value any) map[string]any {
	switch v := value.(type) {
	case map[string]any:
		properties := make(map[string]any, len(v))
		required := make([]string, 0, len(v))
		for key, nested := range v {
			properties[key] = inferSchemaNode(nested)
			required = append(required, key)
		}
		return map[string]any{
			"type":       "object",
			"properties": properties,
			"required":   required,
		}
	case []any:
		schema := map[string]any{"type": "array"}
		if len(v) > 0 {
			schema["items"] = inferSchemaNode(v[0])
		} else {
			schema["items"] = map[string]any{"type": "string"}
		}
		return schema
	case bool:
		return map[string]any{"type": "boolean"}
	case float64, float32:
		return map[string]any{"type": "number"}
	case int, int8, int16, int32, int64:
		return map[string]any{"type": "integer"}
	case uint, uint8, uint16, uint32, uint64:
		return map[string]any{"type": "integer"}
	case json.Number:
		if _, err := strconv.ParseInt(string(v), 10, 64); err == nil {
			return map[string]any{"type": "integer"}
		}
		return map[string]any{"type": "number"}
	case nil:
		return map[string]any{"type": "null"}
	default:
		return map[string]any{"type": "string"}
	}
}

package segment

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

func validateSchemaRecords(schemaDoc map[string]any, records []map[string]any) (map[string]any, error) {
	normalizedSchema := normalizeSchemaForRecords(schemaDoc, records)

	schemaResource := normalizedSchema
	if itemSchema, ok := schemaRecordRoot(normalizedSchema); ok {
		schemaResource = itemSchema
	}

	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	if err := compiler.AddResource("segment-schema.json", schemaResource); err != nil {
		return nil, apierror.NewBadRequest(
			fmt.Sprintf("invalid segment schema resource: %v", err),
			"error.segmentSchemaInvalid",
		)
	}

	compiled, err := compiler.Compile("segment-schema.json")
	if err != nil {
		return nil, apierror.NewBadRequest(
			fmt.Sprintf("invalid segment schema: %v", err),
			"error.segmentSchemaInvalid",
		)
	}

	for idx, record := range records {
		if err := compiled.Validate(record); err != nil {
			return nil, apierror.NewBadRequest(
				fmt.Sprintf("record %d does not match schema: %v", idx+1, err),
				"error.segmentSchemaMismatch",
			)
		}
	}

	return normalizedSchema, nil
}

func schemaRecordRoot(schemaDoc map[string]any) (map[string]any, bool) {
	typeValue, _ := schemaDoc["type"].(string)
	if typeValue != "array" {
		return nil, false
	}

	items, ok := schemaDoc["items"].(map[string]any)
	if !ok {
		return nil, false
	}

	return items, true
}

func normalizeSchemaForRecords(schemaDoc map[string]any, records []map[string]any) map[string]any {
	rootSamples := []any{recordsToAnySlice(records)}
	return normalizeSchemaNode(schemaDoc, rootSamples)
}

func normalizeSchemaNode(node map[string]any, samples []any) map[string]any {
	normalized := cloneSchemaMap(node)
	observedTypes := collectObservedTypes(samples)

	if properties, ok := node["properties"].(map[string]any); ok {
		normalizedProperties := make(map[string]any, len(properties))
		for key, child := range properties {
			childSchema, ok := child.(map[string]any)
			if !ok {
				normalizedProperties[key] = child
				continue
			}
			normalizedProperties[key] = normalizeSchemaNode(childSchema, collectPropertySamples(samples, key))
		}
		normalized["properties"] = normalizedProperties
	}

	if items, ok := node["items"].(map[string]any); ok {
		normalized["items"] = normalizeSchemaNode(items, collectArraySamples(samples))
	}

	if anyOf, ok := node["anyOf"].([]any); ok {
		normalizedAlternatives := make([]any, 0, len(anyOf))
		for _, alternative := range anyOf {
			alternativeSchema, ok := alternative.(map[string]any)
			if !ok {
				normalizedAlternatives = append(normalizedAlternatives, alternative)
				continue
			}
			normalizedAlternatives = append(normalizedAlternatives, normalizeSchemaNode(alternativeSchema, samples))
		}

		normalizedAlternatives = dedupeSchemaAlternatives(normalizedAlternatives)
		if observedTypes["null"] && !alternativesAllowType(normalizedAlternatives, "null") {
			normalizedAlternatives = append(normalizedAlternatives, map[string]any{"type": "null"})
		}

		if collapsedType, ok := collapseSimpleSchemaAlternatives(normalizedAlternatives); ok {
			delete(normalized, "anyOf")
			normalized["type"] = mergeSchemaTypeDeclaration(normalized["type"], collapsedType)
		} else {
			normalized["anyOf"] = normalizedAlternatives
		}
	}

	if typeValue, ok := normalized["type"]; ok {
		normalized["type"] = mergeSchemaTypeDeclaration(typeValue, observedTypes)
	}

	return normalized
}

func cloneSchemaMap(source map[string]any) map[string]any {
	clone := make(map[string]any, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func recordsToAnySlice(records []map[string]any) []any {
	items := make([]any, 0, len(records))
	for _, record := range records {
		items = append(items, record)
	}
	return items
}

func collectPropertySamples(samples []any, key string) []any {
	values := make([]any, 0, len(samples))
	for _, sample := range samples {
		record, ok := sample.(map[string]any)
		if !ok {
			continue
		}
		value, exists := record[key]
		if !exists {
			continue
		}
		values = append(values, value)
	}
	return values
}

func collectArraySamples(samples []any) []any {
	values := make([]any, 0)
	for _, sample := range samples {
		switch typed := sample.(type) {
		case []any:
			values = append(values, typed...)
		case []map[string]any:
			for _, record := range typed {
				values = append(values, record)
			}
		}
	}
	return values
}

func collectObservedTypes(samples []any) map[string]bool {
	types := make(map[string]bool, len(samples))
	for _, sample := range samples {
		types[jsonValueType(sample)] = true
	}
	delete(types, "unknown")
	return types
}

func jsonValueType(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case string:
		return "string"
	case bool:
		return "boolean"
	case float64:
		if math.Trunc(typed) == typed {
			return "integer"
		}
		return "number"
	case float32:
		if math.Trunc(float64(typed)) == float64(typed) {
			return "integer"
		}
		return "number"
	case int, int8, int16, int32, int64:
		return "integer"
	case uint, uint8, uint16, uint32, uint64:
		return "integer"
	case map[string]any:
		return "object"
	case []any:
		return "array"
	default:
		return "unknown"
	}
}

func dedupeSchemaAlternatives(alternatives []any) []any {
	seen := make(map[string]bool, len(alternatives))
	unique := make([]any, 0, len(alternatives))
	for _, alternative := range alternatives {
		encoded, err := json.Marshal(alternative)
		if err != nil {
			unique = append(unique, alternative)
			continue
		}
		key := string(encoded)
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, alternative)
	}
	return unique
}

func alternativesAllowType(alternatives []any, expectedType string) bool {
	for _, alternative := range alternatives {
		alternativeSchema, ok := alternative.(map[string]any)
		if !ok {
			continue
		}
		if extractSchemaTypes(alternativeSchema["type"])[expectedType] {
			return true
		}
	}
	return false
}

func collapseSimpleSchemaAlternatives(alternatives []any) (any, bool) {
	types := make(map[string]bool)
	for _, alternative := range alternatives {
		alternativeSchema, ok := alternative.(map[string]any)
		if !ok || len(alternativeSchema) != 1 {
			return nil, false
		}

		typeValue, exists := alternativeSchema["type"]
		if !exists {
			return nil, false
		}

		alternativeTypes := extractSchemaTypes(typeValue)
		if len(alternativeTypes) == 0 {
			return nil, false
		}

		for schemaType := range alternativeTypes {
			types[schemaType] = true
		}
	}

	return compactSchemaTypes(types), true
}

func mergeSchemaTypeDeclaration(currentType any, observedTypeValue any) any {
	types := extractSchemaTypes(currentType)

	switch typed := observedTypeValue.(type) {
	case map[string]bool:
		for schemaType, exists := range typed {
			if exists {
				types[schemaType] = true
			}
		}
	default:
		for schemaType := range extractSchemaTypes(typed) {
			types[schemaType] = true
		}
	}

	if len(types) == 0 {
		return currentType
	}

	return compactSchemaTypes(types)
}

func extractSchemaTypes(typeValue any) map[string]bool {
	types := make(map[string]bool)
	switch typed := typeValue.(type) {
	case string:
		types[typed] = true
	case []any:
		for _, value := range typed {
			if schemaType, ok := value.(string); ok {
				types[schemaType] = true
			}
		}
	case []string:
		for _, value := range typed {
			types[value] = true
		}
	}
	delete(types, "unknown")
	return types
}

func compactSchemaTypes(types map[string]bool) any {
	if len(types) == 0 {
		return nil
	}

	ordered := make([]string, 0, len(types))
	for schemaType, exists := range types {
		if exists {
			ordered = append(ordered, schemaType)
		}
	}

	slices.SortFunc(ordered, compareSchemaTypeNames)
	if len(ordered) == 1 {
		return ordered[0]
	}

	values := make([]any, 0, len(ordered))
	for _, schemaType := range ordered {
		values = append(values, schemaType)
	}
	return values
}

func compareSchemaTypeNames(left, right string) int {
	order := map[string]int{
		"integer": 0,
		"number":  1,
		"string":  2,
		"boolean": 3,
		"object":  4,
		"array":   5,
		"null":    6,
	}

	leftIndex, leftKnown := order[left]
	rightIndex, rightKnown := order[right]

	switch {
	case leftKnown && rightKnown:
		return leftIndex - rightIndex
	case leftKnown:
		return -1
	case rightKnown:
		return 1
	default:
		return slices.Compare([]string{left}, []string{right})
	}
}

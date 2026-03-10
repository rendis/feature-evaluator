package evaluation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/domain/segment"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// PreparedExpressionInput contains the normalized inputs exposed to the rule engine.
type PreparedExpressionInput struct {
	Headers     map[string]any
	RequestBody map[string]any
	Derived     map[string]any
	Sources     map[string]map[string]any
}

// ResolvedSegmentSource describes the source resolution performed for a segment binding.
type ResolvedSegmentSource struct {
	SegmentKey  string
	LookupPath  string
	LookupValue any
	Found       bool
	Data        map[string]any
}

func PrepareExpressionInput(
	contract feature.InputContract,
	rawInput map[string]any,
	authState AuthValidationResult,
	context map[string]any,
) PreparedExpressionInput {
	headers := normalizeHeaders(contract.Headers, asAnyMap(rawInput["headers"]))
	requestBody := asAnyMap(rawInput["body"])
	derived := deriveExpressionFields(rawInput, authState, context)
	return PreparedExpressionInput{
		Headers:     headers,
		RequestBody: requestBody,
		Derived:     derived,
		Sources:     map[string]map[string]any{},
	}
}

func ResolveSegmentSources(
	ctx context.Context,
	segmentSvc *segment.Service,
	rawBindings feature.SourceBindings,
	prepared *PreparedExpressionInput,
) ([]ResolvedSegmentSource, error) {
	if prepared == nil {
		return nil, nil
	}
	if prepared.Sources == nil {
		prepared.Sources = map[string]map[string]any{}
	}
	results := make([]ResolvedSegmentSource, 0, len(rawBindings.Segments))
	for _, binding := range rawBindings.Segments {
		segmentKey := strings.TrimSpace(binding.SegmentKey)
		lookupValue, found := resolveLookupPath(prepared, binding.LookupPath)
		resolved := ResolvedSegmentSource{
			SegmentKey:  segmentKey,
			LookupPath:  binding.LookupPath,
			LookupValue: lookupValue,
			Found:       false,
			Data:        map[string]any{},
		}
		prepared.Sources[segmentKey] = map[string]any{}

		if segmentSvc == nil {
			return nil, fmt.Errorf("segment service is not configured")
		}
		if !found {
			results = append(results, resolved)
			continue
		}

		recordKey := strings.TrimSpace(fmt.Sprintf("%v", lookupValue))
		if recordKey == "" {
			results = append(results, resolved)
			continue
		}

		record, err := segmentSvc.GetRecordByKey(ctx, segmentKey, recordKey)
		if err != nil {
			var apiErr *apierror.APIError
			if errAs(err, &apiErr) && apiErr.Code == apierror.CodeNotFound {
				results = append(results, resolved)
				continue
			}
			return nil, fmt.Errorf("resolving segment %s with key %s: %w", segmentKey, recordKey, err)
		}
		attributes := record.Attributes
		if attributes == nil {
			attributes = map[string]any{}
		}
		prepared.Sources[segmentKey] = attributes
		resolved.Found = true
		resolved.Data = attributes
		results = append(results, resolved)
	}
	return results, nil
}

func normalizeHeaders(headers []feature.InputHeader, raw map[string]any) map[string]any {
	normalized := make(map[string]any, len(headers))
	for _, header := range headers {
		value, ok := raw[strings.ToLower(header.HeaderName)]
		if !ok {
			continue
		}
		normalized[header.ExpressionKey] = coerceHeaderValue(value, header.Type)
	}
	return normalized
}

func coerceHeaderValue(value any, valueType feature.InputValueType) any {
	raw := strings.TrimSpace(fmt.Sprintf("%v", value))
	if raw == "" {
		return raw
	}
	switch valueType {
	case feature.InputValueTypeBoolean:
		if parsed, err := strconv.ParseBool(raw); err == nil {
			return parsed
		}
	case feature.InputValueTypeNumber:
		if parsed, err := strconv.ParseFloat(raw, 64); err == nil {
			return parsed
		}
	}
	return raw
}

func deriveExpressionFields(rawInput map[string]any, authState AuthValidationResult, context map[string]any) map[string]any {
	derived := map[string]any{
		"authenticated":      authState.Authenticated,
		"bearerTokenPresent": false,
		"apiKeyPresent":      false,
	}

	auth := asAnyMap(rawInput["auth"])
	if bearerToken := strings.TrimSpace(fmt.Sprintf("%v", auth["bearerToken"])); bearerToken != "" {
		derived["bearerTokenPresent"] = true
		claims := decodeJWTClaimsWithoutVerification(bearerToken)
		if subject, ok := claims["sub"].(string); ok && subject != "" {
			derived["subject"] = subject
			derived["userId"] = subject
		}
		if email, ok := claims["email"].(string); ok && email != "" {
			derived["email"] = email
		}
		if name, ok := claims["name"].(string); ok && name != "" {
			derived["name"] = name
		}
	}
	if rawAPIKey := strings.TrimSpace(fmt.Sprintf("%v", auth["apiKey"])); rawAPIKey != "" {
		derived["apiKeyPresent"] = true
	}

	userNS := asAnyMap(context["user"])
	if _, exists := derived["userId"]; !exists {
		if userID := strings.TrimSpace(fmt.Sprintf("%v", userNS["id"])); userID != "" {
			derived["userId"] = userID
		}
	}
	if _, exists := derived["email"]; !exists {
		if email := strings.TrimSpace(fmt.Sprintf("%v", userNS["email"])); email != "" {
			derived["email"] = email
		}
	}

	return derived
}

func resolveLookupPath(prepared *PreparedExpressionInput, path string) (any, bool) {
	switch {
	case strings.HasPrefix(path, "headers."):
		return resolveMapPath(prepared.Headers, strings.TrimPrefix(path, "headers."))
	case strings.HasPrefix(path, "requestBody."):
		return resolveMapPath(prepared.RequestBody, strings.TrimPrefix(path, "requestBody."))
	case strings.HasPrefix(path, "derived."):
		return resolveMapPath(prepared.Derived, strings.TrimPrefix(path, "derived."))
	default:
		return nil, false
	}
}

func resolveMapPath(root map[string]any, path string) (any, bool) {
	if path == "" {
		return nil, false
	}
	current := any(root)
	for _, part := range strings.Split(path, ".") {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, exists := m[part]
		if !exists {
			return nil, false
		}
		current = next
	}
	return current, true
}

func asAnyMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case map[string]string:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = item
		}
		return out
	default:
		return map[string]any{}
	}
}

func decodeJWTClaimsWithoutVerification(token string) map[string]any {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return map[string]any{}
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return map[string]any{}
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return map[string]any{}
	}
	return claims
}

func errAs(err error, target any) bool {
	return errors.As(err, target)
}

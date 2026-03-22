package authprofile

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/rendis/feature-evaluator/pkg/apierror"
)

const defaultExternalTimeoutMS = 5000

// ValidateProfile validates and normalizes type-specific auth profile config.
func ValidateProfile(profile *Profile, secretPayload map[string]string, creating bool, requireSecret bool) error {
	if profile == nil {
		return apierror.NewBadRequest("auth profile is required", "error.invalidAuthProfile")
	}

	switch profile.Type {
	case TypeAPIKey:
		return validateAPIKeyProfile(profile, secretPayload, creating, requireSecret)
	case TypeOIDCStandard:
		return validateOIDCStandardProfile(profile, secretPayload)
	case TypeCustom:
		return validateCustomProfile(profile, secretPayload, creating, requireSecret)
	default:
		return apierror.NewBadRequest(
			fmt.Sprintf("invalid auth profile type: %s", profile.Type),
			"error.invalidAuthProfileType",
		)
	}
}

func validateAPIKeyProfile(profile *Profile, secretPayload map[string]string, creating bool, requireSecret bool) error {
	location := strings.ToLower(strings.TrimSpace(stringConfig(profile.Config, "location")))
	if location != "header" && location != "query" {
		return apierror.NewBadRequest(
			"api_key auth profile requires config.location to be 'header' or 'query'",
			"error.invalidAuthProfileConfig",
		)
	}

	name := strings.TrimSpace(stringConfig(profile.Config, "name"))
	if name == "" {
		return apierror.NewBadRequest(
			"api_key auth profile requires config.name",
			"error.invalidAuthProfileConfig",
		)
	}

	profile.Config["location"] = location
	profile.Config["name"] = name
	profile.Config["prefix"] = strings.TrimSpace(stringConfig(profile.Config, "prefix"))
	profile.CacheTTLSeconds = 0

	if requireSecret || creating {
		if strings.TrimSpace(secretPayload["apiKey"]) == "" {
			return apierror.NewBadRequest(
				"api_key auth profile requires secretPayload.apiKey",
				"error.invalidAuthProfileSecret",
			)
		}
	}
	return nil
}

func validateOIDCStandardProfile(profile *Profile, secretPayload map[string]string) error {
	issuer := strings.TrimSpace(stringConfig(profile.Config, "issuer"))
	if issuer == "" {
		return apierror.NewBadRequest(
			"oidc_standard auth profile requires config.issuer",
			"error.invalidAuthProfileConfig",
		)
	}

	audience := strings.TrimSpace(stringConfig(profile.Config, "audience"))
	if audience == "" {
		return apierror.NewBadRequest(
			"oidc_standard auth profile requires config.audience",
			"error.invalidAuthProfileConfig",
		)
	}

	if len(secretPayload) > 0 {
		return apierror.NewBadRequest(
			"oidc_standard auth profile does not accept secret payload",
			"error.invalidAuthProfileConfig",
		)
	}

	for key := range profile.Config {
		switch key {
		case "issuer", "audience":
		default:
			return apierror.NewBadRequest(
				fmt.Sprintf("oidc_standard auth profile does not accept config.%s", key),
				"error.invalidAuthProfileConfig",
			)
		}
	}

	profile.Config = map[string]any{
		"issuer":   strings.TrimRight(issuer, "/"),
		"audience": audience,
	}
	profile.CacheTTLSeconds = 0
	return nil
}

func validateCustomProfile(profile *Profile, secretPayload map[string]string, creating bool, requireSecret bool) error {
	url := strings.TrimSpace(stringConfig(profile.Config, "url"))
	if url == "" {
		return apierror.NewBadRequest(
			"custom auth profile requires config.url",
			"error.invalidAuthProfileConfig",
		)
	}

	method := normalizeMethod(stringConfigDefault(profile.Config, "method", http.MethodPost))
	if method == "" {
		return apierror.NewBadRequest(
			"custom auth profile requires config.method to be GET or POST",
			"error.invalidAuthProfileConfig",
		)
	}

	headers, ok := normalizeMappings(profile.Config["headers"])
	if !ok {
		return apierror.NewBadRequest(
			"custom auth profile requires config.headers to be an array of source/destination mappings",
			"error.invalidAuthProfileConfig",
		)
	}
	body, ok := normalizeMappings(profile.Config["body"])
	if !ok {
		return apierror.NewBadRequest(
			"custom auth profile requires config.body to be an array of source/destination mappings",
			"error.invalidAuthProfileConfig",
		)
	}

	requestHeaders, ok := normalizeRequestHeaders(profile.Config["requestHeaders"])
	if !ok {
		return apierror.NewBadRequest(
			"custom auth profile requires config.requestHeaders to be an array of {key, value} pairs with unique keys",
			"error.invalidAuthProfileConfig",
		)
	}

	successRule, ok := normalizeSuccessRule(profile.Config["successRule"])
	if !ok {
		return apierror.NewBadRequest(
			"custom auth profile requires a valid config.successRule",
			"error.invalidAuthProfileConfig",
		)
	}

	outboundHeader := strings.TrimSpace(stringConfig(profile.Config, "outboundAuthHeaderName"))
	if outboundHeader != "" && (creating || requireSecret) && strings.TrimSpace(secretPayload["outboundApiKey"]) == "" {
		return apierror.NewBadRequest(
			"custom auth profile requires secretPayload.outboundApiKey when outboundAuthHeaderName is set",
			"error.invalidAuthProfileSecret",
		)
	}

	profile.Config["url"] = url
	profile.Config["method"] = method
	profile.Config["timeout"] = normalizeTimeout(intConfig(profile.Config, "timeout"))
	profile.Config["headers"] = headers
	profile.Config["body"] = body
	profile.Config["requestHeaders"] = requestHeaders
	profile.Config["successRule"] = successRule
	if outboundHeader != "" {
		profile.Config["outboundAuthHeaderName"] = outboundHeader
	}
	profile.Normalize()
	return nil
}

func normalizeMethod(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case http.MethodGet:
		return http.MethodGet
	case http.MethodPost:
		return http.MethodPost
	default:
		return ""
	}
}

func normalizeTimeout(timeout int) int {
	if timeout <= 0 {
		return defaultExternalTimeoutMS
	}
	if timeout > 30000 {
		return 30000
	}
	return timeout
}

func stringConfig(config map[string]any, key string) string {
	if config == nil {
		return ""
	}
	raw, ok := config[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return value
}

func stringConfigDefault(config map[string]any, key, fallback string) string {
	if value := stringConfig(config, key); value != "" {
		return value
	}
	return fallback
}

func intConfig(config map[string]any, key string) int {
	if config == nil {
		return 0
	}
	switch value := config[key].(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func normalizeMappings(raw any) ([]map[string]any, bool) { //nolint:gocognit,cyclop // mapping normalization
	if raw == nil {
		return []map[string]any{}, true
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, false
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		mapping, ok := item.(map[string]any)
		if !ok {
			return nil, false
		}
		source := nestedString(mapping, "source", "type")
		target := nestedString(mapping, "target", "type")
		if source == "" || target == "" {
			return nil, false
		}
		switch source {
		case "request_header":
			if nestedString(mapping, "source", "name") == "" {
				return nil, false
			}
			if stripPrefix := nestedString(mapping, "source", "stripPrefix"); stripPrefix != "" {
				nested, ok := mapping["source"].(map[string]any)
				if !ok {
					return nil, false
				}
				nested["stripPrefix"] = stripPrefix
			}
		case "request_body":
			if nestedString(mapping, "source", "path") == "" {
				return nil, false
			}
		default:
			return nil, false
		}
		switch target {
		case "header":
			if nestedString(mapping, "target", "name") == "" {
				return nil, false
			}
		case "body":
			if nestedString(mapping, "target", "path") == "" {
				return nil, false
			}
		default:
			return nil, false
		}
		result = append(result, mapping)
	}
	return result, true
}

const maxRequestHeaders = 20

func normalizeRequestHeaders(raw any) ([]map[string]any, bool) {
	if raw == nil {
		return []map[string]any{}, true
	}
	var items []map[string]any
	switch typed := raw.(type) {
	case []any:
		items = make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			entry, ok := item.(map[string]any)
			if !ok {
				return nil, false
			}
			items = append(items, entry)
		}
	case []map[string]any:
		items = typed
	default:
		return nil, false
	}
	if len(items) > maxRequestHeaders {
		return nil, false
	}
	seen := make(map[string]bool, len(items))
	result := make([]map[string]any, 0, len(items))
	for _, entry := range items {
		key := strings.TrimSpace(stringConfig(entry, "key"))
		if key == "" {
			return nil, false
		}
		lower := strings.ToLower(key)
		if seen[lower] {
			return nil, false
		}
		seen[lower] = true
		value := strings.TrimSpace(stringConfig(entry, "value"))
		result = append(result, map[string]any{"key": key, "value": value})
	}
	return result, true
}

func normalizeSuccessRule(raw any) (map[string]any, bool) {
	if raw == nil {
		return map[string]any{"type": "any_2xx"}, true
	}
	rule, ok := raw.(map[string]any)
	if !ok {
		return nil, false
	}
	ruleType := strings.TrimSpace(fmt.Sprint(rule["type"]))
	switch ruleType {
	case "", "any_2xx":
		return map[string]any{"type": "any_2xx"}, true
	case "status":
		if intConfig(rule, "status") <= 0 {
			return nil, false
		}
		return map[string]any{"type": "status", "status": intConfig(rule, "status")}, true
	case "json_field":
		path := strings.TrimSpace(fmt.Sprint(rule["path"]))
		if path == "" {
			return nil, false
		}
		return map[string]any{
			"type":     "json_field",
			"path":     path,
			"operator": strings.TrimSpace(fmt.Sprint(rule["operator"])),
			"value":    rule["value"],
		}, true
	case "response_header":
		header := strings.TrimSpace(fmt.Sprint(rule["header"]))
		if header == "" {
			return nil, false
		}
		return map[string]any{
			"type":   "response_header",
			"header": header,
			"value":  fmt.Sprint(rule["value"]),
		}, true
	case "text_contains":
		value := strings.TrimSpace(fmt.Sprint(rule["value"]))
		if value == "" {
			return nil, false
		}
		return map[string]any{"type": "text_contains", "value": value}, true
	default:
		return nil, false
	}
}

func nestedString(mapping map[string]any, top, key string) string {
	nested, ok := mapping[top].(map[string]any)
	if !ok {
		return ""
	}
	value, ok := nested[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

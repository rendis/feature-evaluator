package externalapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/rendis/feature-evaluator/pkg/apierror"
)

var (
	paramNamePattern   = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	placeholderPattern = regexp.MustCompile(`\{\{\s*([^}]+?)\s*\}\}`)
)

var allowedMethods = map[string]bool{
	http.MethodGet:    true,
	http.MethodPost:   true,
	http.MethodPut:    true,
	http.MethodPatch:  true,
	http.MethodDelete: true,
}

// Validate verifies structural correctness and normalizes defaults.
func Validate(api *ExternalAPI) error {
	if api == nil {
		return apierror.NewBadRequest("external api is required", "error.invalidExternalAPI")
	}
	if strings.TrimSpace(api.Name) == "" {
		return apierror.NewBadRequest("external api name is required", "error.externalAPINameRequired")
	}
	method := strings.ToUpper(strings.TrimSpace(api.Request.Method))
	if !allowedMethods[method] {
		return apierror.NewBadRequest("external api method is invalid", "error.invalidExternalAPIRequest")
	}
	if strings.TrimSpace(api.Request.URLTemplate) == "" {
		return apierror.NewBadRequest("external api url template is required", "error.invalidExternalAPIRequest")
	}
	api.Request.Method = method
	if api.Request.Headers == nil {
		api.Request.Headers = []HeaderTemplate{}
	}
	if api.Params == nil {
		api.Params = []Param{}
	}
	if api.ExpressionVariables == nil {
		api.ExpressionVariables = []ExpressionVariable{}
	}
	if err := validateParams(api); err != nil {
		return err
	}
	if err := validateExpressionVariables(api); err != nil {
		return err
	}
	if err := validateBodyTemplate(api.Request.BodyTemplate); err != nil {
		return err
	}
	if err := validateResponseValidation(&api.ResponseValidation); err != nil {
		return err
	}
	return nil
}

func validateExpressionVariables(api *ExternalAPI) error {
	declared := make(map[string]bool, len(api.Params))
	for i := range api.Params {
		declared[api.Params[i].Name] = true
	}

	expressionVariables := make(map[string]bool, len(api.ExpressionVariables))
	for i := range api.ExpressionVariables {
		variable := api.ExpressionVariables[i]
		variable.Name = strings.TrimSpace(variable.Name)
		if strings.HasPrefix(variable.Name, "secret.") {
			return apierror.NewBadRequest(
				"expression variables cannot use the secret. prefix",
				"error.invalidExternalAPIParam",
			)
		}
		if !paramNamePattern.MatchString(variable.Name) {
			return apierror.NewBadRequest(
				"expression variable names must match [a-zA-Z][a-zA-Z0-9_]*",
				"error.invalidExternalAPIParam",
			)
		}
		if declared[variable.Name] {
			return apierror.NewBadRequest(
				fmt.Sprintf("expression variable %q conflicts with a request param", variable.Name),
				"error.invalidExternalAPIParam",
			)
		}
		if expressionVariables[variable.Name] {
			return apierror.NewBadRequest(
				"expression variable names must be unique",
				"error.invalidExternalAPIParam",
			)
		}
		if !variable.Type.Valid() {
			variable.Type = ParamTypeAny
		}

		api.ExpressionVariables[i] = variable
		expressionVariables[variable.Name] = true
	}

	return nil
}

func validateParams(api *ExternalAPI) error { //nolint:gocognit,cyclop // parameter validation
	detectedParams, secretRefs := CollectTemplateReferences(api.Request)
	declared := make(map[string]Param, len(api.Params))
	for i := range api.Params {
		param := api.Params[i]
		param.Name = strings.TrimSpace(param.Name)
		if !paramNamePattern.MatchString(param.Name) {
			return apierror.NewBadRequest("parameter names must match [a-zA-Z][a-zA-Z0-9_]*", "error.invalidExternalAPIParam")
		}
		if _, exists := declared[param.Name]; exists {
			return apierror.NewBadRequest("parameter names must be unique", "error.invalidExternalAPIParam")
		}
		if !param.Type.Valid() {
			param.Type = ParamTypeAny
		}
		if len(param.Locations) == 0 {
			return apierror.NewBadRequest("parameter locations are required", "error.invalidExternalAPIParam")
		}
		for _, location := range param.Locations {
			if !location.Valid() {
				return apierror.NewBadRequest("parameter location is invalid", "error.invalidExternalAPIParam")
			}
		}
		if param.URLKind != nil {
			if !param.URLKind.Valid() {
				return apierror.NewBadRequest("parameter urlKind is invalid", "error.invalidExternalAPIParam")
			}
			if *param.URLKind == URLKindDomain || *param.URLKind == URLKindPath {
				param.Required = true
			}
		}
		api.Params[i] = param
		declared[param.Name] = param
	}

	for detectedName, usage := range detectedParams {
		param, ok := declared[detectedName]
		if !ok {
			return apierror.NewBadRequest(
				fmt.Sprintf("missing param definition for %q", detectedName),
				"error.invalidExternalAPIParam",
			)
		}
		for _, location := range usage.Locations {
			if !slices.Contains(param.Locations, location) {
				return apierror.NewBadRequest(
					fmt.Sprintf("param %q is missing location %q", detectedName, location),
					"error.invalidExternalAPIParam",
				)
			}
		}
		if usage.URLKind != nil {
			if param.URLKind == nil || *param.URLKind != *usage.URLKind {
				return apierror.NewBadRequest(
					fmt.Sprintf("param %q has an invalid urlKind", detectedName),
					"error.invalidExternalAPIParam",
				)
			}
			if (*usage.URLKind == URLKindDomain || *usage.URLKind == URLKindPath) && !param.Required {
				return apierror.NewBadRequest(
					fmt.Sprintf("param %q must be required", detectedName),
					"error.invalidExternalAPIParam",
				)
			}
		}
	}

	for declaredName := range declared {
		if _, ok := detectedParams[declaredName]; ok {
			continue
		}
		if strings.HasPrefix(declaredName, "secret.") || secretRefs[strings.TrimPrefix(declaredName, "secret.")] {
			continue
		}
		return apierror.NewBadRequest(
			fmt.Sprintf("param %q is not referenced by the request template", declaredName),
			"error.invalidExternalAPIParam",
		)
	}

	return nil
}

func validateBodyTemplate(bodyTemplate any) error {
	if bodyTemplate == nil {
		return nil
	}
	if _, err := json.Marshal(bodyTemplate); err != nil {
		return apierror.NewBadRequest("request body template must be valid JSON", "error.invalidExternalAPIRequest")
	}
	return nil
}

func validateResponseValidation(validation *ResponseValidation) error { //nolint:cyclop // validates response config fields
	if validation == nil {
		return apierror.NewBadRequest("response validation is required", "error.invalidExternalAPIResponseValidation")
	}
	switch validation.Mode {
	case ValidationModeHTTPCode, ValidationModeResponseBody, ValidationModeBoth:
	default:
		return apierror.NewBadRequest("response validation mode is invalid", "error.invalidExternalAPIResponseValidation")
	}
	if validation.HTTP.Mode == "" {
		validation.HTTP.Mode = HTTPValidationModeAny2xx
	}
	switch validation.HTTP.Mode {
	case HTTPValidationModeAny2xx:
		validation.HTTP.Codes = nil
	case HTTPValidationModeStatusCode:
		if len(validation.HTTP.Codes) == 0 {
			return apierror.NewBadRequest("http validation requires at least one status code", "error.invalidExternalAPIResponseValidation")
		}
		for _, code := range validation.HTTP.Codes {
			if code < 100 || code > 599 {
				return apierror.NewBadRequest("http validation contains an invalid status code", "error.invalidExternalAPIResponseValidation")
			}
		}
	default:
		return apierror.NewBadRequest("http validation mode is invalid", "error.invalidExternalAPIResponseValidation")
	}
	if validation.Mode == ValidationModeResponseBody || validation.Mode == ValidationModeBoth {
		if strings.TrimSpace(validation.Body.Expression) == "" {
			return apierror.NewBadRequest("body validation expression is required", "error.invalidExternalAPIResponseValidation")
		}
	}
	if validation.Body.Schema == nil {
		validation.Body.Schema = map[string]any{}
	}
	return nil
}

// Valid reports whether the param type is supported.
func (t ParamType) Valid() bool {
	switch t {
	case ParamTypeAny, ParamTypeString, ParamTypeNumber, ParamTypeBool:
		return true
	default:
		return false
	}
}

// Valid reports whether the location value is supported.
func (l Location) Valid() bool {
	switch l {
	case LocationURL, LocationHeader, LocationBody:
		return true
	default:
		return false
	}
}

// Valid reports whether the URL kind value is supported.
func (k URLKind) Valid() bool {
	switch k {
	case URLKindDomain, URLKindPath, URLKindQuery:
		return true
	default:
		return false
	}
}

// PlaceholderUsage describes where one placeholder is referenced.
type PlaceholderUsage struct {
	Locations []Location
	URLKind   *URLKind
}

// CollectTemplateReferences walks the request template and returns non-secret placeholders and secret refs.
func CollectTemplateReferences(request RequestConfig) (map[string]PlaceholderUsage, map[string]bool) { //nolint:gocognit // template reference collection
	params := map[string]PlaceholderUsage{}
	secrets := map[string]bool{}

	recordParam := func(name string, location Location, kind *URLKind) {
		usage := params[name]
		if !slices.Contains(usage.Locations, location) {
			usage.Locations = append(usage.Locations, location)
		}
		if kind != nil {
			usage.URLKind = kind
		}
		params[name] = usage
	}

	for _, match := range collectMatches(request.URLTemplate) {
		if strings.HasPrefix(match.Name, "secret.") {
			secrets[strings.TrimPrefix(match.Name, "secret.")] = true
			continue
		}
		if match.URLKind != nil {
			recordParam(match.Name, LocationURL, match.URLKind)
		}
	}
	for _, header := range request.Headers {
		for _, ref := range collectGeneralPlaceholders(header.KeyTemplate) {
			if strings.HasPrefix(ref, "secret.") {
				secrets[strings.TrimPrefix(ref, "secret.")] = true
				continue
			}
			recordParam(ref, LocationHeader, nil)
		}
		for _, ref := range collectGeneralPlaceholders(header.ValueTemplate) {
			if strings.HasPrefix(ref, "secret.") {
				secrets[strings.TrimPrefix(ref, "secret.")] = true
				continue
			}
			recordParam(ref, LocationHeader, nil)
		}
	}
	collectBodyPlaceholders(request.BodyTemplate, func(ref string) {
		if strings.HasPrefix(ref, "secret.") {
			secrets[strings.TrimPrefix(ref, "secret.")] = true
			return
		}
		recordParam(ref, LocationBody, nil)
	})

	return params, secrets
}

type urlPlaceholderMatch struct {
	Name    string
	URLKind *URLKind
}

func collectMatches(urlTemplate string) []urlPlaceholderMatch {
	matches := make([]urlPlaceholderMatch, 0)
	if strings.TrimSpace(urlTemplate) == "" {
		return matches
	}
	queryIndex := strings.Index(urlTemplate, "?")
	schemeIndex := strings.Index(urlTemplate, "://")
	hostStart := 0
	if schemeIndex >= 0 {
		hostStart = schemeIndex + 3
	}
	pathIndex := strings.Index(urlTemplate[hostStart:], "/")
	if pathIndex >= 0 {
		pathIndex += hostStart
	} else {
		pathIndex = len(urlTemplate)
	}

	for _, match := range placeholderPattern.FindAllStringSubmatchIndex(urlTemplate, -1) {
		if len(match) < 4 {
			continue
		}
		name := strings.TrimSpace(urlTemplate[match[2]:match[3]])
		var kind URLKind
		switch {
		case queryIndex >= 0 && match[0] >= queryIndex:
			kind = URLKindQuery
		case match[0] < pathIndex:
			kind = URLKindDomain
		default:
			kind = URLKindPath
		}
		kindCopy := kind
		matches = append(matches, urlPlaceholderMatch{Name: name, URLKind: &kindCopy})
	}
	return matches
}

func collectGeneralPlaceholders(value string) []string {
	matches := placeholderPattern.FindAllStringSubmatch(value, -1)
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		result = append(result, strings.TrimSpace(match[1]))
	}
	return result
}

func collectBodyPlaceholders(value any, visit func(ref string)) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			for _, ref := range collectGeneralPlaceholders(key) {
				visit(ref)
			}
			collectBodyPlaceholders(child, visit)
		}
	case []any:
		for _, child := range typed {
			collectBodyPlaceholders(child, visit)
		}
	case string:
		for _, ref := range collectGeneralPlaceholders(typed) {
			visit(ref)
		}
	}
}

package incomingauth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/authprofile"
	"github.com/rendis/feature-evaluator/internal/domain/evaluation"
	"github.com/rendis/feature-evaluator/internal/external"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
	redisclient "github.com/rendis/feature-evaluator/internal/storage/redis"
)

const defaultTimeout = 5 * time.Second

// Validator validates incoming eval requests using feature-bound auth profiles.
type Validator struct {
	profiles   *authprofile.Service
	redis      *redisclient.Client
	httpClient *http.Client
	oidcMu     sync.Mutex
	oidc       map[string]*middleware.JWTValidator
}

// NewValidator creates a new inbound auth validator.
func NewValidator(redis *redisclient.Client, profiles *authprofile.Service) *Validator {
	return &Validator{
		profiles:   profiles,
		redis:      redis,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		oidc:       make(map[string]*middleware.JWTValidator),
	}
}

// Validate resolves a stored auth profile and validates the incoming request.
func (v *Validator) Validate(
	ctx context.Context,
	key string,
	input map[string]any,
) (*evaluation.AuthValidationResult, error) {
	if v.profiles == nil {
		return nil, fmt.Errorf("auth profile service is not configured")
	}

	profile, secrets, err := v.profiles.Resolve(ctx, key)
	if err != nil {
		return nil, err
	}
	if !profile.Active {
		return nil, fmt.Errorf("auth profile %q is inactive", key)
	}
	return v.validateProfile(ctx, profile, secrets, input, false)
}

// ValidateDraft validates a draft profile without persisting it.
func (v *Validator) ValidateDraft(
	ctx context.Context,
	profile *authprofile.Profile,
	secretPayload map[string]string,
	input map[string]any,
) (*AuthTestResult, error) {
	if profile == nil {
		return nil, fmt.Errorf("auth profile is required")
	}
	profile.Normalize()
	if err := authprofile.ValidateProfile(profile, secretPayload, true, true); err != nil {
		return nil, err
	}
	return v.validateDraftProfile(ctx, profile, secretPayload, input)
}

func (v *Validator) validateProfile(
	ctx context.Context,
	profile *authprofile.Profile,
	secrets map[string]string,
	input map[string]any,
	bypassCache bool,
) (*evaluation.AuthValidationResult, error) {
	switch profile.Type {
	case authprofile.TypeAPIKey:
		result, _, err := validateFixedAPIKey(profile, secrets, input)
		return result, err
	case authprofile.TypeOIDCStandard:
		result, err := v.validateOIDCStandardDraft(ctx, profile, input)
		if err != nil {
			return nil, err
		}
		return &result.AuthValidationResult, nil
	case authprofile.TypeCustom:
		result, err := v.validateCustomDraft(ctx, profile, secrets, input, bypassCache)
		if err != nil {
			return nil, err
		}
		return &result.AuthValidationResult, nil
	default:
		return nil, fmt.Errorf("unsupported auth profile type %q", profile.Type)
	}
}

// AuthTestResult includes debug details for draft profile testing.
type AuthTestResult struct {
	evaluation.AuthValidationResult
	Details map[string]any
}

func (v *Validator) validateDraftProfile(
	ctx context.Context,
	profile *authprofile.Profile,
	secrets map[string]string,
	input map[string]any,
) (*AuthTestResult, error) {
	switch profile.Type {
	case authprofile.TypeAPIKey:
		authResult, details, err := validateFixedAPIKey(profile, secrets, input)
		if err != nil {
			return nil, err
		}
		return &AuthTestResult{AuthValidationResult: *authResult, Details: details}, nil
	case authprofile.TypeOIDCStandard:
		return v.validateOIDCStandardDraft(ctx, profile, input)
	case authprofile.TypeCustom:
		return v.validateCustomDraft(ctx, profile, secrets, input, true)
	default:
		return nil, fmt.Errorf("unsupported auth profile type %q", profile.Type)
	}
}

func validateFixedAPIKey(
	profile *authprofile.Profile,
	secrets map[string]string,
	input map[string]any,
) (*evaluation.AuthValidationResult, map[string]any, error) {
	location := strings.ToLower(strings.TrimSpace(stringConfig(profile.Config, "location")))
	name := strings.TrimSpace(stringConfig(profile.Config, "name"))
	prefix := strings.TrimSpace(stringConfig(profile.Config, "prefix"))
	expected := strings.TrimSpace(secrets["apiKey"])
	if expected == "" {
		return nil, nil, fmt.Errorf("api_key auth profile requires secretPayload.apiKey")
	}

	var raw string
	switch location {
	case "header":
		raw = getHeader(input, name)
	case "query":
		raw = getQueryParam(input, name)
	default:
		return nil, nil, fmt.Errorf("unsupported api_key location %q", location)
	}

	if strings.TrimSpace(raw) == "" {
		return &evaluation.AuthValidationResult{Authenticated: false, Attempted: false}, map[string]any{
			"location": location,
			"name":     name,
			"source":   "incoming_request",
			"reason":   "missing credential in configured location",
		}, nil
	}

	candidate := strings.TrimSpace(raw)
	if prefix != "" {
		if !strings.HasPrefix(candidate, prefix) {
			return &evaluation.AuthValidationResult{Authenticated: false, Attempted: true}, map[string]any{
				"location": location,
				"name":     name,
				"prefix":   prefix,
			}, nil
		}
		candidate = strings.TrimSpace(strings.TrimPrefix(candidate, prefix))
	}

	authenticated := subtle.ConstantTimeCompare([]byte(candidate), []byte(expected)) == 1
	return &evaluation.AuthValidationResult{Authenticated: authenticated, Attempted: true}, map[string]any{
		"location": location,
		"name":     name,
		"prefix":   prefix,
	}, nil
}

func (v *Validator) validateOIDCStandardDraft(
	_ context.Context,
	profile *authprofile.Profile,
	input map[string]any,
) (*AuthTestResult, error) {
	authorization := strings.TrimSpace(getHeader(input, "authorization"))
	if authorization == "" {
		return &AuthTestResult{
			AuthValidationResult: evaluation.AuthValidationResult{Authenticated: false, Attempted: false},
			Details: map[string]any{
				"issuer":   stringConfig(profile.Config, "issuer"),
				"audience": stringConfig(profile.Config, "audience"),
				"reason":   "missing Authorization header in test request",
			},
		}, nil
	}

	token, ok := strings.CutPrefix(authorization, "Bearer ")
	if !ok || strings.TrimSpace(token) == "" {
		return &AuthTestResult{
			AuthValidationResult: evaluation.AuthValidationResult{Authenticated: false, Attempted: true},
			Details: map[string]any{
				"issuer":   stringConfig(profile.Config, "issuer"),
				"audience": stringConfig(profile.Config, "audience"),
				"reason":   "invalid Authorization header format; expected Bearer token",
			},
		}, nil
	}

	validator := v.oidcValidator(
		stringConfig(profile.Config, "issuer"),
		stringConfig(profile.Config, "audience"),
	)
	claims, err := validator.Validate(strings.TrimSpace(token))
	if err != nil {
		return &AuthTestResult{
			AuthValidationResult: evaluation.AuthValidationResult{Authenticated: false, Attempted: true},
			Details: map[string]any{
				"issuer":   stringConfig(profile.Config, "issuer"),
				"audience": stringConfig(profile.Config, "audience"),
				"reason":   err.Error(),
			},
		}, nil
	}

	return &AuthTestResult{
		AuthValidationResult: evaluation.AuthValidationResult{
			Authenticated: true,
			Attempted:     true,
		},
		Details: map[string]any{
			"issuer":   stringConfig(profile.Config, "issuer"),
			"audience": stringConfig(profile.Config, "audience"),
			"subject":  claims.Subject,
			"email":    claims.Email,
			"name":     claims.Name,
		},
	}, nil
}

func (v *Validator) oidcValidator(issuer, audience string) *middleware.JWTValidator {
	key := strings.TrimRight(strings.TrimSpace(issuer), "/") + "|" + strings.TrimSpace(audience)

	v.oidcMu.Lock()
	defer v.oidcMu.Unlock()

	if validator, ok := v.oidc[key]; ok {
		return validator
	}

	validator := middleware.NewJWTValidator(issuer, audience)
	v.oidc[key] = validator
	return validator
}

func (v *Validator) validateCustomDraft( //nolint:gocognit,cyclop,funlen // custom auth validation
	ctx context.Context,
	profile *authprofile.Profile,
	secrets map[string]string,
	input map[string]any,
	bypassCache bool,
) (*AuthTestResult, error) {
	mappingsHeaders := mappingRows(profile.Config["headers"])
	mappingsBody := mappingRows(profile.Config["body"])
	targetURL := stringConfig(profile.Config, "url")
	method := stringConfigDefault(profile.Config, "method", http.MethodPost)
	successRule, _ := profile.Config["successRule"].(map[string]any)

	headers := http.Header{}
	bodyPayload := map[string]any{}
	fingerprintSeed := map[string]any{}
	attempted := false

	for _, rh := range requestHeaderRows(profile.Config["requestHeaders"]) {
		headers.Set(rh.Key, rh.Value)
	}

	for _, mapping := range mappingsHeaders {
		value, ok := resolveSourceValue(input, mapping)
		if !ok {
			continue
		}
		attempted = true
		headers.Set(mapping.TargetName, fmt.Sprint(value))
		fingerprintSeed["header:"+mapping.TargetName] = value
	}
	for _, mapping := range mappingsBody {
		value, ok := resolveSourceValue(input, mapping)
		if !ok {
			continue
		}
		attempted = true
		setValueAtPath(bodyPayload, mapping.TargetPath, value)
		fingerprintSeed["body:"+mapping.TargetPath] = value
	}

	if !attempted {
		return &AuthTestResult{
			AuthValidationResult: evaluation.AuthValidationResult{Authenticated: false, Attempted: false},
			Details: map[string]any{
				"url":    targetURL,
				"method": method,
				"reason": "no configured mapping matched any value from the test request",
			},
		}, nil
	}

	cacheKey := ""
	ttl := time.Duration(profile.CacheTTLSeconds) * time.Second
	if ttl > 0 {
		cacheKey = redisclient.AuthProfileValidationKey(
			profile.WorkspaceKey,
			profile.Key,
			profile.Version,
			hashObject(fingerprintSeed),
		)
	}
	if cacheKey != "" && !bypassCache {
		if cached, err := v.redis.Get(ctx, cacheKey); err == nil && cached == "true" {
			return &AuthTestResult{
				AuthValidationResult: evaluation.AuthValidationResult{
					Authenticated: true,
					Attempted:     true,
					Cached:        true,
				},
				Details: map[string]any{
					"url": targetURL,
				},
			}, nil
		}
	}

	if err := isPrivateURL(targetURL); err != nil {
		return nil, fmt.Errorf("ssrf protection: %w", err)
	}

	timeout := time.Duration(intConfig(profile.Config, "timeout")) * time.Millisecond
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var bodyReader io.Reader
	if len(bodyPayload) > 0 && method != http.MethodGet {
		bodyBytes, err := json.Marshal(bodyPayload)
		if err != nil {
			return nil, fmt.Errorf("marshaling custom auth request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
		headers.Set("Content-Type", "application/json")
	}

	req, err := http.NewRequestWithContext(callCtx, method, targetURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating custom auth request: %w", err)
	}
	req.Header = headers
	if outboundHeader := strings.TrimSpace(stringConfig(profile.Config, "outboundAuthHeaderName")); outboundHeader != "" {
		if outboundKey := strings.TrimSpace(secrets["outboundApiKey"]); outboundKey != "" {
			req.Header.Set(outboundHeader, outboundKey)
		}
	}

	result, err := v.doRequest(req)
	if err != nil {
		return nil, err
	}

	passed, evalErr := evaluateSuccessRule(successRule, result.body, result.statusCode, result.header)
	if evalErr != nil {
		return nil, evalErr
	}
	if passed && cacheKey != "" && !bypassCache {
		_ = v.redis.Set(ctx, cacheKey, "true", ttl)
	}
	return &AuthTestResult{
		AuthValidationResult: evaluation.AuthValidationResult{
			Authenticated: passed,
			Attempted:     true,
			HTTPStatus:    result.statusCode,
		},
		Details: map[string]any{
			"request": map[string]any{
				"url":     targetURL,
				"method":  method,
				"headers": redactHeaders(req.Header),
				"body":    bodyPayload,
			},
			"responseText":    string(result.body),
			"responseHeaders": normalizeHeaders(result.header),
		},
	}, nil
}

type doRequestResult struct {
	statusCode int
	header     http.Header
	body       []byte
}

func (v *Validator) doRequest(req *http.Request) (*doRequestResult, error) {
	resp, err := v.httpClient.Do(req) //nolint:gosec // URL is validated by isPrivateURL before reaching here
	if err != nil {
		return nil, fmt.Errorf("executing auth validation request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading auth validation response: %w", err)
	}
	return &doRequestResult{
		statusCode: resp.StatusCode,
		header:     resp.Header.Clone(),
		body:       body,
	}, nil
}

type requestHeader struct {
	Key   string
	Value string
}

func requestHeaderRows(raw any) []requestHeader {
	var items []map[string]any
	switch typed := raw.(type) {
	case []any:
		items = make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			items = append(items, entry)
		}
	case []map[string]any:
		items = typed
	default:
		return nil
	}
	rows := make([]requestHeader, 0, len(items))
	for _, entry := range items {
		key := strings.TrimSpace(stringConfig(entry, "key"))
		if key == "" {
			continue
		}
		value := strings.TrimSpace(stringConfig(entry, "value"))
		rows = append(rows, requestHeader{Key: key, Value: value})
	}
	return rows
}

type mappingRow struct {
	SourceType  string
	SourceName  string
	SourcePath  string
	StripPrefix string
	TargetType  string
	TargetName  string
	TargetPath  string
}

func mappingRows(raw any) []mappingRow {
	var items []map[string]any
	switch typed := raw.(type) {
	case []any:
		items = make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			mapping, ok := item.(map[string]any)
			if !ok {
				continue
			}
			items = append(items, mapping)
		}
	case []map[string]any:
		items = typed
	default:
		return []mappingRow{}
	}
	rows := make([]mappingRow, 0, len(items))
	for _, mapping := range items {
		rows = append(rows, mappingRow{
			SourceType:  strings.TrimSpace(nestedString(mapping, "source", "type")),
			SourceName:  strings.TrimSpace(nestedString(mapping, "source", "name")),
			SourcePath:  strings.TrimSpace(nestedString(mapping, "source", "path")),
			StripPrefix: strings.TrimSpace(nestedString(mapping, "source", "stripPrefix")),
			TargetType:  strings.TrimSpace(nestedString(mapping, "target", "type")),
			TargetName:  strings.TrimSpace(nestedString(mapping, "target", "name")),
			TargetPath:  strings.TrimSpace(nestedString(mapping, "target", "path")),
		})
	}
	return rows
}

func resolveSourceValue(input map[string]any, row mappingRow) (any, bool) {
	switch row.SourceType {
	case "request_header":
		value := getHeader(input, row.SourceName)
		if value == "" {
			return nil, false
		}
		if row.StripPrefix != "" {
			value = stripPrefixIfPresent(value, row.StripPrefix)
		}
		return value, true
	case "request_body":
		value := external.ResolveValue(row.SourcePath, getBody(input))
		if value == nil {
			return nil, false
		}
		return value, true
	default:
		return nil, false
	}
}

func stripPrefixIfPresent(value, prefix string) string {
	if len(value) < len(prefix) || prefix == "" {
		return value
	}
	if strings.EqualFold(value[:len(prefix)], prefix) {
		return strings.TrimSpace(value[len(prefix):])
	}
	return value
}

func evaluateSuccessRule(
	rule map[string]any,
	body []byte,
	status int,
	headers http.Header,
) (bool, error) {
	if len(rule) == 0 || strings.TrimSpace(fmt.Sprint(rule["type"])) == "" || fmt.Sprint(rule["type"]) == "any_2xx" {
		return status >= 200 && status < 300, nil
	}

	switch strings.TrimSpace(fmt.Sprint(rule["type"])) {
	case "status":
		return status == intConfig(rule, "status"), nil
	case "json_field":
		var response map[string]any
		if err := json.Unmarshal(body, &response); err != nil {
			return false, fmt.Errorf("parsing json response: %w", err)
		}
		actual := external.ResolveValue(strings.TrimSpace(fmt.Sprint(rule["path"])), response)
		expected := rule["value"]
		operator := strings.TrimSpace(fmt.Sprint(rule["operator"]))
		if operator == "not_equals" {
			return fmt.Sprint(actual) != fmt.Sprint(expected), nil
		}
		return fmt.Sprint(actual) == fmt.Sprint(expected), nil
	case "response_header":
		headerName := strings.ToLower(strings.TrimSpace(fmt.Sprint(rule["header"])))
		return strings.TrimSpace(strings.Join(headers.Values(headerName), ",")) == strings.TrimSpace(fmt.Sprint(rule["value"])), nil
	case "text_contains":
		return strings.Contains(string(body), strings.TrimSpace(fmt.Sprint(rule["value"]))), nil
	default:
		return false, fmt.Errorf("unsupported success rule type %q", rule["type"])
	}
}

func getHeader(input map[string]any, name string) string {
	headers, _ := input["headers"].(map[string]any)
	if headers == nil {
		return ""
	}
	if value, ok := headers[strings.ToLower(name)]; ok {
		return strings.TrimSpace(fmt.Sprint(value))
	}
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return strings.TrimSpace(fmt.Sprint(value))
		}
	}
	return ""
}

func getQueryParam(input map[string]any, name string) string {
	query, _ := input["query"].(map[string]any)
	if query == nil {
		return ""
	}
	if value, ok := query[name]; ok {
		return strings.TrimSpace(fmt.Sprint(value))
	}
	for key, value := range query {
		if strings.EqualFold(key, name) {
			return strings.TrimSpace(fmt.Sprint(value))
		}
	}
	return ""
}

func getBody(input map[string]any) map[string]any {
	body, _ := input["body"].(map[string]any)
	if body == nil {
		return map[string]any{}
	}
	return body
}

func setValueAtPath(target map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	current := target
	for idx, part := range parts {
		if idx == len(parts)-1 {
			current[part] = value
			return
		}
		next, ok := current[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[part] = next
		}
		current = next
	}
}

func redactHeaders(headers http.Header) map[string]any {
	out := map[string]any{}
	for key, values := range headers {
		if len(values) == 0 {
			continue
		}
		lowerKey := strings.ToLower(key)
		if lowerKey == "authorization" || strings.Contains(lowerKey, "api-key") || strings.Contains(lowerKey, "token") {
			out[lowerKey] = "[redacted]"
			continue
		}
		out[lowerKey] = strings.Join(values, ",")
	}
	return out
}

func normalizeHeaders(headers http.Header) map[string]any {
	out := map[string]any{}
	for key, values := range headers {
		if len(values) == 0 {
			continue
		}
		out[strings.ToLower(key)] = strings.Join(values, ",")
	}
	return out
}

// isPrivateURL checks if a URL resolves to a private/internal IP.
// Set ALLOW_PRIVATE_URLS=true to bypass this check (dev/test only).
func isPrivateURL(rawURL string) error {
	if os.Getenv("ALLOW_PRIVATE_URLS") == "true" {
		return nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	host := parsed.Hostname()
	if host == "169.254.169.254" || strings.HasSuffix(host, ".internal") {
		return fmt.Errorf("blocked: cloud metadata endpoint")
	}
	ips, err := net.LookupHost(host)
	if err != nil {
		return fmt.Errorf("resolving host %s: %w", host, err)
	}
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return fmt.Errorf("blocked: URL resolves to private/internal IP %s", ipStr)
		}
	}
	return nil
}

func stringConfig(config map[string]any, key string) string {
	raw, ok := config[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func stringConfigDefault(config map[string]any, key, fallback string) string {
	if value := stringConfig(config, key); value != "" {
		return value
	}
	return fallback
}

func intConfig(values map[string]any, key string) int {
	switch value := values[key].(type) {
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

func nestedString(mapping map[string]any, top, key string) string {
	nested, ok := mapping[top].(map[string]any)
	if !ok {
		return ""
	}
	value, ok := nested[key]
	if !ok {
		return ""
	}
	return fmt.Sprint(value)
}

func hashFingerprint(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func hashObject(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return hashFingerprint(string(raw))
}

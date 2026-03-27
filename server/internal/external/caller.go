package external

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/externalapi"
	"github.com/rendis/feature-evaluator/internal/domain/feature"
	"github.com/rendis/feature-evaluator/internal/domain/workspace"
	"github.com/rendis/feature-evaluator/internal/secrets"
	redisclient "github.com/rendis/feature-evaluator/internal/storage/redis"
)

// LatencyFunc is called after each external HTTP call with the duration.
type LatencyFunc func(d time.Duration)

// CallResult holds the result of an external validation call.
type CallResult struct {
	Passed     bool
	Cached     bool
	HTTPStatus int
}

type httpCallResult struct {
	Body           []byte
	Headers        http.Header
	HTTPStatus     int
	RequestURL     string
	RequestMethod  string
	RequestHeaders map[string]string
	RequestBody    any
}

// APITestRequestPreview describes the rendered outbound request shown in the test UI.
type APITestRequestPreview struct {
	URL     string            `json:"url,omitempty"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body,omitempty"`
}

// APITestDetails captures the request/response debug payload returned by the test endpoint.
type APITestDetails struct {
	Request         APITestRequestPreview `json:"request"`
	ResponseText    string                `json:"responseText,omitempty"`
	ResponseHeaders map[string]string     `json:"responseHeaders,omitempty"`
	ResponseBody    any                   `json:"responseBody,omitempty"`
	Evaluations     APIEvaluationDetails  `json:"evaluations"`
}

const defaultOutboundUserAgent = "feature-evaluator/1.0"

// Caller handles external HTTP calls for rule validation.
type Caller struct {
	httpClient   *http.Client
	cbManager    *CircuitBreakerManager
	redis        *redisclient.Client
	secretCipher *secrets.Cipher
	onLatency    LatencyFunc
}

// NewCaller creates a new external validation caller.
func NewCaller(redis *redisclient.Client, secretCipher *secrets.Cipher) *Caller {
	return &Caller{
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		cbManager:    NewCircuitBreakerManager(),
		redis:        redis,
		secretCipher: secretCipher,
	}
}

// SetHTTPClient overrides the internal HTTP client. Primarily used by tests.
func (c *Caller) SetHTTPClient(client *http.Client) {
	if client == nil {
		return
	}
	c.httpClient = client
}

// SetOnLatency sets the callback invoked after each external call with its duration.
func (c *Caller) SetOnLatency(fn LatencyFunc) {
	c.onLatency = fn
}

// CBManager returns the circuit breaker manager for hooking state changes.
func (c *Caller) CBManager() *CircuitBreakerManager {
	return c.cbManager
}

// CallExternalAPI executes a reusable external API request and returns whether it passed plus the HTTP status.
func (c *Caller) CallExternalAPI(
	ctx context.Context,
	api *externalapi.ExternalAPI,
	binding feature.ExternalAPIBinding,
	paramValues map[string]any,
	secretValues map[string]string,
) (*CallResult, error) {
	if cached := c.getCachedCallResult(ctx, api, binding, paramValues); cached != nil {
		return cached, nil
	}

	expressionVars, err := buildExternalAPIExpressionVars(api, paramValues)
	if err != nil {
		return nil, err
	}

	rendered, err := RenderExternalAPIRequest(api.Request, api.Params, paramValues, secretValues)
	if err != nil {
		return nil, err
	}

	callResult, err := c.executeRenderedRequest(ctx, rendered, 10*time.Second, nil)
	if err != nil {
		return nil, err
	}

	passed, err := EvaluateExternalAPIResponse(
		api.ResponseValidation,
		callResult.Body,
		callResult.HTTPStatus,
		callResult.Headers,
		expressionVars,
	)
	if err != nil {
		// Response evaluation errors (expression runtime failures like index out of
		// bounds on empty arrays) mean the response data doesn't match the expected
		// shape. This is "condition not met" (false), not an infrastructure failure
		// that should trigger fail-mode logic.
		slog.Warn("external api response evaluation failed, treating as not-passed",
			"apiKey", api.Key,
			"error", err,
		)
		result := &CallResult{Passed: false, HTTPStatus: callResult.HTTPStatus}
		c.setCachedCallResult(ctx, api, binding, paramValues, result)
		return result, nil
	}
	result := &CallResult{Passed: passed, HTTPStatus: callResult.HTTPStatus}
	c.setCachedCallResult(ctx, api, binding, paramValues, result)
	return result, nil
}

// TestExternalAPI executes one reusable external API request and returns debug details.
func (c *Caller) TestExternalAPI(
	ctx context.Context,
	api *externalapi.ExternalAPI,
	paramValues map[string]any,
	secretValues map[string]string,
) (*CallResult, *APITestDetails, error) {
	expressionVars, err := buildExternalAPIExpressionVars(api, paramValues)
	if err != nil {
		return nil, nil, err
	}

	rendered, err := RenderExternalAPIRequest(api.Request, api.Params, paramValues, secretValues)
	if err != nil {
		return nil, nil, err
	}

	callResult, err := c.executeRenderedRequest(ctx, rendered, 10*time.Second, nil)
	if err != nil {
		return nil, nil, err
	}

	evaluations, err := EvaluateExternalAPIResponseDetailed(
		api.ResponseValidation,
		callResult.Body,
		callResult.HTTPStatus,
		callResult.Headers,
		expressionVars,
	)
	if err != nil {
		return nil, nil, err
	}

	return &CallResult{
			Passed:     evaluations.Final.Passed,
			HTTPStatus: callResult.HTTPStatus,
		}, &APITestDetails{
			Request: APITestRequestPreview{
				URL:     callResult.RequestURL,
				Method:  callResult.RequestMethod,
				Headers: callResult.RequestHeaders,
				Body:    callResult.RequestBody,
			},
			ResponseText:    string(callResult.Body),
			ResponseHeaders: previewHeaderStrings(callResult.Headers),
			ResponseBody:    requestBodyPreview(callResult.Body),
			Evaluations:     evaluations,
		}, nil
}

func requestBodyPreview(body []byte) any {
	if len(body) == 0 {
		return nil
	}
	var parsed any
	if err := json.Unmarshal(body, &parsed); err == nil {
		return parsed
	}
	return string(body)
}

type externalCallCachePayload struct {
	Passed     bool `json:"passed"`
	HTTPStatus int  `json:"httpStatus"`
}

func (c *Caller) getCachedCallResult(
	ctx context.Context,
	api *externalapi.ExternalAPI,
	binding feature.ExternalAPIBinding,
	paramValues map[string]any,
) *CallResult {
	if c.redis == nil || !binding.CacheEnabled || binding.CacheTTL <= 0 {
		return nil
	}
	key, err := externalCallCacheKey(ctx, api, paramValues)
	if err != nil {
		return nil
	}
	raw, _ := c.redis.Get(ctx, key)
	if raw == "" {
		return nil
	}
	var payload externalCallCachePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		_ = c.redis.Del(ctx, key)
		return nil
	}
	return &CallResult{
		Passed:     payload.Passed,
		Cached:     true,
		HTTPStatus: payload.HTTPStatus,
	}
}

func (c *Caller) setCachedCallResult(
	ctx context.Context,
	api *externalapi.ExternalAPI,
	binding feature.ExternalAPIBinding,
	paramValues map[string]any,
	result *CallResult,
) {
	if c.redis == nil || result == nil || !binding.CacheEnabled || binding.CacheTTL <= 0 {
		return
	}
	key, err := externalCallCacheKey(ctx, api, paramValues)
	if err != nil {
		return
	}
	payload, err := json.Marshal(externalCallCachePayload{
		Passed:     result.Passed,
		HTTPStatus: result.HTTPStatus,
	})
	if err != nil {
		return
	}
	_ = c.redis.Set(ctx, key, payload, time.Duration(binding.CacheTTL)*time.Second)
}

func externalCallCacheKey(ctx context.Context, api *externalapi.ExternalAPI, paramValues map[string]any) (string, error) {
	payload, err := json.Marshal(struct {
		Workspace string         `json:"workspace"`
		API       string         `json:"api"`
		Version   int            `json:"version"`
		Params    map[string]any `json:"params"`
	}{
		Workspace: workspace.KeyFromContext(ctx),
		API:       api.Key,
		Version:   api.Version,
		Params:    paramValues,
	})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return redisclient.ExternalCallKey(hex.EncodeToString(sum[:])), nil
}

func previewHeaderStrings(headers http.Header) map[string]string {
	normalized := normalizePreviewHeaders(headers)
	stringHeaders := make(map[string]string, len(normalized))
	for key, value := range normalized {
		stringHeaders[key] = fmt.Sprint(value)
	}
	return stringHeaders
}

func (c *Caller) executeRenderedRequest(
	ctx context.Context,
	rendered *RenderedRequest,
	timeout time.Duration,
	prepare func(*http.Request) error,
) (*httpCallResult, error) {
	if err := isPrivateURL(rendered.URL); err != nil {
		return nil, fmt.Errorf("SSRF protection: %w", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	bodyBytes, bodyReader, err := buildRequestBodyReader(rendered)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(callCtx, strings.ToUpper(rendered.Method), rendered.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	applyRenderedRequestHeaders(req, rendered.Headers, bodyReader != nil)
	if prepare != nil {
		if err := prepare(req); err != nil {
			return nil, err
		}
	}

	//nolint:gosec // The URL is validated by isPrivateURL before issuing the request.
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing external call to %s: %w", rendered.URL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return &httpCallResult{
		Body:           body,
		Headers:        resp.Header.Clone(),
		HTTPStatus:     resp.StatusCode,
		RequestURL:     req.URL.String(),
		RequestMethod:  req.Method,
		RequestHeaders: previewHeaderStrings(req.Header),
		RequestBody:    requestBodyPreview(bodyBytes),
	}, nil
}

func applyRenderedRequestHeaders(req *http.Request, renderedHeaders map[string]string, hasBody bool) {
	if hasBody {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range renderedHeaders {
		req.Header.Set(key, value)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", defaultOutboundUserAgent)
	}
}

func buildRequestBodyReader(rendered *RenderedRequest) ([]byte, io.Reader, error) {
	if rendered == nil || rendered.Body == nil {
		return nil, nil, nil
	}
	bodyBytes, err := json.Marshal(rendered.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling request body: %w", err)
	}
	return bodyBytes, bytes.NewReader(bodyBytes), nil
}

// isPrivateURL checks if a URL resolves to a private, loopback, link-local, or cloud metadata IP.
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

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rendis/feature-evaluator/apps/mcp/internal/auth"
)

// Client is a thin HTTP client for the feature-evaluator API.
type Client struct {
	baseURL       string // e.g. "http://localhost:8080/features"
	tokenProvider auth.TokenProvider
	httpClient    *http.Client
	mu            sync.RWMutex
	workspace     string
}

// New creates a Client pointing at the given API base URL.
func New(baseURL string, tp auth.TokenProvider, workspace string) *Client {
	return &Client{
		baseURL:       baseURL,
		tokenProvider: tp,
		workspace:     workspace,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Workspace returns the current workspace key.
func (c *Client) Workspace() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.workspace
}

// SetWorkspace changes the active workspace for subsequent calls.
func (c *Client) SetWorkspace(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.workspace = key
}

// Healthy pings the /healthz endpoint. Returns nil on success.
func (c *Client) Healthy() error {
	resp, err := c.do("GET", "/healthz", nil, "")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned %d", resp.StatusCode)
	}
	return nil
}

// Get performs a GET request to the admin path and returns parsed JSON.
func (c *Client) Get(path string) (any, error) {
	return c.request("GET", "/admin"+path, nil)
}

// GetEval performs a GET to the eval path (not under /admin).
func (c *Client) GetEval(path string) (any, error) {
	return c.request("GET", path, nil)
}

// Post performs a POST request with JSON body to the admin path.
func (c *Client) Post(path string, body any) (any, error) {
	return c.request("POST", "/admin"+path, body)
}

// PostEval performs a POST to the eval path (not under /admin).
func (c *Client) PostEval(path string, body any) (any, error) {
	return c.request("POST", path, body)
}

// Put performs a PUT request with JSON body to the admin path.
func (c *Client) Put(path string, body any) (any, error) {
	return c.request("PUT", "/admin"+path, body)
}

// Patch performs a PATCH request with JSON body to the admin path.
func (c *Client) Patch(path string, body any) (any, error) {
	return c.request("PATCH", "/admin"+path, body)
}

// Delete performs a DELETE request to the admin path.
func (c *Client) Delete(path string) error {
	resp, err := c.do("DELETE", "/admin"+path, nil, "application/json")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return parseAPIError(resp)
	}
	return nil
}

// DeleteWithBody performs a DELETE request with JSON body to the admin path.
func (c *Client) DeleteWithBody(path string, body any) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}
	resp, err := c.do("DELETE", "/admin"+path, bodyReader, "application/json")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return parseAPIError(resp)
	}
	return nil
}

func (c *Client) request(method, path string, body any) (any, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	resp, err := c.do(method, path, bodyReader, "application/json")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNoContent {
		return map[string]any{"status": "ok"}, nil
	}
	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp)
	}
	return decodeJSON(resp)
}

func (c *Client) do(method, path string, body io.Reader, contentType string) (*http.Response, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	token, err := c.tokenProvider.Token(context.Background())
	if err != nil {
		return nil, fmt.Errorf("obtain auth token: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	c.mu.RLock()
	ws := c.workspace
	c.mu.RUnlock()
	if ws != "" {
		req.Header.Set("X-Workspace", ws)
	}
	return c.httpClient.Do(req)
}

func decodeJSON(resp *http.Response) (any, error) {
	var result any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return result, nil
}

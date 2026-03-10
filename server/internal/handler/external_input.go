package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
)

func buildExternalInput(c *gin.Context, body map[string]any) map[string]any {
	headers := make(map[string]any, len(c.Request.Header))
	for key, values := range c.Request.Header {
		if len(values) == 0 {
			continue
		}
		headers[strings.ToLower(key)] = strings.Join(values, ",")
	}

	query := make(map[string]any, len(c.Request.URL.Query()))
	for key, values := range c.Request.URL.Query() {
		if len(values) == 0 {
			continue
		}
		query[key] = strings.Join(values, ",")
	}

	authHeader := c.GetHeader("Authorization")
	bearerToken := ""
	if strings.HasPrefix(strings.TrimSpace(authHeader), "Bearer ") {
		bearerToken = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(authHeader), "Bearer "))
	}
	rawAPIKey := c.GetHeader("X-Api-Key")
	if rawAPIKey == "" {
		rawAPIKey = c.GetHeader("X-API-Key")
	}
	if body == nil {
		body = map[string]any{}
	}

	effectiveBody := body
	effectiveHeaders := headers
	if rawInput, ok := body["input"].(map[string]any); ok {
		if nestedBody, ok := rawInput["body"].(map[string]any); ok {
			effectiveBody = nestedBody
		}
		if nestedHeaders, ok := rawInput["headers"].(map[string]any); ok {
			effectiveHeaders = make(map[string]any, len(headers)+len(nestedHeaders))
			for key, value := range headers {
				effectiveHeaders[key] = value
			}
			for key, value := range nestedHeaders {
				effectiveHeaders[strings.ToLower(key)] = value
			}
		}
	}

	return map[string]any{
		"headers": effectiveHeaders,
		"query":   query,
		"body":    effectiveBody,
		"request": map[string]any{
			"id":     middleware.GetRequestID(c),
			"ip":     c.ClientIP(),
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
		},
		"auth": map[string]any{
			"rawAuthorization": authHeader,
			"rawApiKey":        rawAPIKey,
			"bearerToken":      bearerToken,
			"apiKey":           rawAPIKey,
		},
	}
}

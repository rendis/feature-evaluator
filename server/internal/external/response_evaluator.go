package external

import (
	"net/http"
	"strings"

	"github.com/rendis/feature-evaluator/internal/domain/externalapi"
)

// EvaluateResponse evaluates the response condition expression against the HTTP response body.
func EvaluateResponse(condition string, responseBody []byte, httpStatus int, headers http.Header) (bool, error) {
	if condition == "" {
		return httpStatus >= 200 && httpStatus < 300, nil
	}
	return EvaluateExternalAPIResponse(externalapi.ResponseValidation{
		Mode: externalapi.ValidationModeResponseBody,
		HTTP: externalapi.HTTPValidation{Mode: externalapi.HTTPValidationModeAny2xx},
		Body: externalapi.BodyValidation{Expression: condition},
	}, responseBody, httpStatus, headers, nil)
}

func normalizeResponseHeaders(headers http.Header) map[string]any {
	return normalizeHeaders(headers, true)
}

func normalizePreviewHeaders(headers http.Header) map[string]any {
	return normalizeHeaders(headers, false)
}

func normalizeHeaders(headers http.Header, includeLowercaseAliases bool) map[string]any {
	result := make(map[string]any, len(headers))
	for key, values := range headers {
		if len(values) == 0 {
			continue
		}
		joined := strings.Join(values, ",")
		result[http.CanonicalHeaderKey(key)] = joined
		if includeLowercaseAliases {
			result[strings.ToLower(key)] = joined
		}
	}
	return result
}

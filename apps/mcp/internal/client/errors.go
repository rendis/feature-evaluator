package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIError represents a parsed error from the feature-evaluator API.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	MessageKey string
	Details    any
}

func (e *APIError) Error() string {
	if e.Details != nil {
		return fmt.Sprintf("API %d [%s]: %s (details: %v)", e.StatusCode, e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("API %d [%s]: %s", e.StatusCode, e.Code, e.Message)
}

func parseAPIError(resp *http.Response) *APIError {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d (failed to read body)", resp.StatusCode),
		}
	}

	// Feature-evaluator wraps errors as {"error": {"code": "...", "message": "...", "messageKey": "..."}}
	var parsed struct {
		Error struct {
			Code       string `json:"code"`
			Message    string `json:"message"`
			MessageKey string `json:"messageKey"`
			Details    any    `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
		}
	}

	msg := parsed.Error.Message
	if msg == "" {
		msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return &APIError{
		StatusCode: resp.StatusCode,
		Code:       parsed.Error.Code,
		Message:    msg,
		MessageKey: parsed.Error.MessageKey,
		Details:    parsed.Error.Details,
	}
}

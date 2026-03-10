package apierror

import (
	"fmt"
	"net/http"
)

// APIError represents a structured API error response.
type APIError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	MessageKey string `json:"messageKey"`
	Details    any    `json:"details,omitempty"`
	RequestID  string `json:"requestId,omitempty"`
	HTTPStatus int    `json:"-"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// WithRequestID returns a copy of the error with the given request ID.
func (e *APIError) WithRequestID(requestID string) *APIError {
	cp := *e
	cp.RequestID = requestID
	return &cp
}

// WithDetails returns a copy of the error with additional details.
func (e *APIError) WithDetails(details any) *APIError {
	cp := *e
	cp.Details = details
	return &cp
}

// ErrorResponse is the wire format for error responses.
type ErrorResponse struct {
	Error *APIError `json:"error"`
}

// NewBadRequest creates a 400 Bad Request error.
func NewBadRequest(message, messageKey string) *APIError {
	return &APIError{
		Code:       CodeBadRequest,
		Message:    message,
		MessageKey: messageKey,
		HTTPStatus: http.StatusBadRequest,
	}
}

// NewUnauthorized creates a 401 Unauthorized error.
func NewUnauthorized(message, messageKey string) *APIError {
	return &APIError{
		Code:       CodeUnauthorized,
		Message:    message,
		MessageKey: messageKey,
		HTTPStatus: http.StatusUnauthorized,
	}
}

// NewForbidden creates a 403 Forbidden error.
func NewForbidden(message, messageKey string) *APIError {
	return &APIError{
		Code:       CodeForbidden,
		Message:    message,
		MessageKey: messageKey,
		HTTPStatus: http.StatusForbidden,
	}
}

// NewNotFound creates a 404 Not Found error.
func NewNotFound(message, messageKey string) *APIError {
	return &APIError{
		Code:       CodeNotFound,
		Message:    message,
		MessageKey: messageKey,
		HTTPStatus: http.StatusNotFound,
	}
}

// NewConflict creates a 409 Conflict error.
func NewConflict(message, messageKey string) *APIError {
	return &APIError{
		Code:       CodeConflict,
		Message:    message,
		MessageKey: messageKey,
		HTTPStatus: http.StatusConflict,
	}
}

// NewValidation creates a 422 Validation Error.
func NewValidation(message, messageKey string) *APIError {
	return &APIError{
		Code:       CodeValidation,
		Message:    message,
		MessageKey: messageKey,
		HTTPStatus: http.StatusUnprocessableEntity,
	}
}

// NewTooManyRequests creates a 429 Too Many Requests error.
func NewTooManyRequests(message, messageKey string) *APIError {
	return &APIError{
		Code:       CodeTooManyRequests,
		Message:    message,
		MessageKey: messageKey,
		HTTPStatus: http.StatusTooManyRequests,
	}
}

// NewInternal creates a 500 Internal Server Error.
func NewInternal(message, messageKey string) *APIError {
	return &APIError{
		Code:       CodeInternal,
		Message:    message,
		MessageKey: messageKey,
		HTTPStatus: http.StatusInternalServerError,
	}
}

// NewServiceUnavailable creates a 503 Service Unavailable error.
func NewServiceUnavailable(message, messageKey string) *APIError {
	return &APIError{
		Code:       CodeServiceUnavailable,
		Message:    message,
		MessageKey: messageKey,
		HTTPStatus: http.StatusServiceUnavailable,
	}
}

// NewLastOwner creates an error for attempting to remove/demote the last owner.
func NewLastOwner() *APIError {
	return &APIError{
		Code:       CodeLastOwner,
		Message:    "Cannot remove or demote the last owner",
		MessageKey: "error.lastOwner",
		HTTPStatus: http.StatusConflict,
	}
}

// NewSelfDemotion creates an error for attempting self-demotion from owner.
func NewSelfDemotion() *APIError {
	return &APIError{
		Code:       CodeSelfDemotion,
		Message:    "Cannot demote yourself from owner role",
		MessageKey: "error.selfDemotion",
		HTTPStatus: http.StatusConflict,
	}
}

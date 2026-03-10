package dto

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// RespondError writes a structured error response.
// If the error is an *apierror.APIError, it uses its HTTP status and structure.
// Otherwise, it returns a 500 Internal Server Error.
func RespondError(c *gin.Context, err error) {
	var apiErr *apierror.APIError
	if errors.As(err, &apiErr) {
		apiErr.RequestID = middleware.GetRequestID(c)
		c.JSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
		return
	}

	if isBadRequestError(err) {
		badRequest := apierror.NewBadRequest(err.Error(), "error.badRequest")
		badRequest.RequestID = middleware.GetRequestID(c)
		c.JSON(badRequest.HTTPStatus, apierror.ErrorResponse{Error: badRequest})
		return
	}

	internal := apierror.NewInternal("internal server error", "error.internal")
	internal.RequestID = middleware.GetRequestID(c)
	c.JSON(internal.HTTPStatus, apierror.ErrorResponse{Error: internal})
}

func isBadRequestError(err error) bool {
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		return true
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return true
	}

	var unmarshalTypeErr *json.UnmarshalTypeError
	if errors.As(err, &unmarshalTypeErr) {
		return true
	}

	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
}

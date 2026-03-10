package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// BodySizeLimit returns a middleware that limits request body size.
func BodySizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()

		// Check if the body was too large (MaxBytesReader sets this error).
		if c.Errors.Last() != nil {
			for _, e := range c.Errors {
				if e.Err != nil && e.Err.Error() == "http: request body too large" {
					apiErr := apierror.NewBadRequest("request body too large", "error.bodyTooLarge")
					apiErr.RequestID = GetRequestID(c)
					c.AbortWithStatusJSON(apiErr.HTTPStatus, apierror.ErrorResponse{Error: apiErr})
					return
				}
			}
		}
	}
}

package apierror

// Error codes used across the API.
const (
	CodeBadRequest           = "BAD_REQUEST"
	CodeUnauthorized         = "UNAUTHORIZED"
	CodeForbidden            = "FORBIDDEN"
	CodeNotFound             = "NOT_FOUND"
	CodeConflict             = "CONFLICT"
	CodeValidation           = "VALIDATION_ERROR"
	CodeTooManyRequests      = "TOO_MANY_REQUESTS"
	CodeInternal             = "INTERNAL_ERROR"
	CodeServiceUnavailable   = "SERVICE_UNAVAILABLE"
	CodeExpressionInvalid    = "EXPRESSION_INVALID"
	CodeExpressionDenied     = "EXPRESSION_DENIED"
	CodeExpressionTooComplex = "EXPRESSION_TOO_COMPLEX"
	CodeLastOwner            = "LAST_OWNER"
	CodeSelfDemotion         = "SELF_DEMOTION"
	CodeFeatureDisabled      = "FEATURE_DISABLED"
	CodeEvaluationError      = "EVALUATION_ERROR"
	CodeExternalCallFailed   = "EXTERNAL_CALL_FAILED"
)

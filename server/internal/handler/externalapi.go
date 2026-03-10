package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/externalapi"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/external"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// ExternalAPIHandler handles CRUD and testing for reusable external APIs.
type ExternalAPIHandler struct {
	svc      *externalapi.Service
	executor *external.Caller
}

// NewExternalAPIHandler creates a new ExternalAPIHandler.
func NewExternalAPIHandler(svc *externalapi.Service, executor *external.Caller) *ExternalAPIHandler {
	return &ExternalAPIHandler{svc: svc, executor: executor}
}

// List returns all external APIs in the current workspace.
func (h *ExternalAPIHandler) List(c *gin.Context) {
	apis, err := h.svc.List(c.Request.Context())
	if err != nil {
		slog.Error("listing external apis", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	data := make([]dto.ExternalAPIResponse, 0, len(apis))
	for i := range apis {
		data = append(data, dto.ToExternalAPIResponse(&apis[i]))
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

// Get returns one external API.
func (h *ExternalAPIHandler) Get(c *gin.Context) {
	api, err := h.svc.GetByKey(c.Request.Context(), c.Param("key"))
	if err != nil {
		dto.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ToExternalAPIResponse(api))
}

// ExpressionProfile returns the shared semantic catalog used by the external
// API response expression editor.
func (h *ExternalAPIHandler) ExpressionProfile(c *gin.Context) {
	c.JSON(http.StatusOK, buildExternalAPIExpressionProfile())
}

// ValidateExpression validates a draft external API response expression without
// persisting any changes.
func (h *ExternalAPIHandler) ValidateExpression(c *gin.Context) {
	var req dto.ValidateExternalAPIExpressionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	if strings.TrimSpace(req.Expression) == "" {
		c.JSON(http.StatusOK, dto.ValidateExpressionResponse{Valid: true})
		return
	}

	if err := external.ValidateResponseExpression(req.Expression); err != nil {
		errStr := err.Error()
		c.JSON(http.StatusOK, dto.ValidateExpressionResponse{
			Valid: false,
			Error: &errStr,
		})
		return
	}

	c.JSON(http.StatusOK, dto.ValidateExpressionResponse{Valid: true})
}

// Create stores a new external API definition.
func (h *ExternalAPIHandler) Create(c *gin.Context) {
	var req dto.CreateExternalAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	if err := validateExternalAPIExpression(req.ResponseValidation); err != nil {
		dto.RespondError(c, err)
		return
	}

	api := &externalapi.ExternalAPI{
		Key:                 req.Key,
		Name:                req.Name,
		Active:              req.Active,
		Request:             req.Request,
		Params:              req.Params,
		ExpressionVariables: req.ExpressionVariables,
		ResponseValidation:  req.ResponseValidation,
		CreatedBy:           middleware.GetUserEmail(c),
		UpdatedBy:           middleware.GetUserEmail(c),
	}
	if err := h.svc.Create(c.Request.Context(), api, req.SecretPayload); err != nil {
		slog.Error("creating external api", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToExternalAPIResponse(api))
}

// Update updates an existing external API definition.
func (h *ExternalAPIHandler) Update(c *gin.Context) {
	currentKey := c.Param("key")
	var req dto.UpdateExternalAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	if err := validateExternalAPIExpression(req.ResponseValidation); err != nil {
		dto.RespondError(c, err)
		return
	}

	api := &externalapi.ExternalAPI{
		Key:                 req.Key,
		Name:                req.Name,
		Active:              req.Active,
		Request:             req.Request,
		Params:              req.Params,
		ExpressionVariables: req.ExpressionVariables,
		ResponseValidation:  req.ResponseValidation,
		UpdatedBy:           middleware.GetUserEmail(c),
	}
	if err := h.svc.Update(c.Request.Context(), currentKey, api, req.SecretPayload, req.ReplaceSecret); err != nil {
		slog.Error("updating external api", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	updated, err := h.svc.GetByKey(c.Request.Context(), api.Key)
	if err != nil {
		dto.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ToExternalAPIResponse(updated))
}

// Delete removes an external API definition.
func (h *ExternalAPIHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Request.Context(), c.Param("key")); err != nil {
		slog.Error("deleting external api", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "external api deleted"})
}

// Test executes an external API draft with sample param values.
func (h *ExternalAPIHandler) Test(c *gin.Context) {
	var req dto.TestExternalAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	if err := validateExternalAPIExpression(req.ResponseValidation); err != nil {
		dto.RespondError(c, err)
		return
	}

	api := &externalapi.ExternalAPI{
		Key:                 req.Key,
		Name:                req.Name,
		Request:             req.Request,
		Params:              req.Params,
		ExpressionVariables: req.ExpressionVariables,
		ResponseValidation:  req.ResponseValidation,
	}
	secretValues, err := h.svc.ResolveDraftSecrets(
		c.Request.Context(),
		req.CurrentKey,
		api,
		req.SecretPayload,
		req.ReplaceSecret,
	)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	result, details, err := h.executor.TestExternalAPI(c.Request.Context(), api, req.ParamValues, secretValues)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	var detailsResponse *dto.ExternalAPITestDetailsResponse
	if details != nil {
		detailsResponse = &dto.ExternalAPITestDetailsResponse{
			ResponseText:    details.ResponseText,
			ResponseHeaders: details.ResponseHeaders,
			ResponseBody:    details.ResponseBody,
			Evaluations:     &details.Evaluations,
		}
		detailsResponse.Request = &dto.ExternalAPITestRequestResponse{
			URL:     details.Request.URL,
			Method:  details.Request.Method,
			Headers: details.Request.Headers,
			Body:    details.Request.Body,
		}
	}

	c.JSON(http.StatusOK, dto.ExternalAPITestResponse{
		OK:         result.Passed,
		Attempted:  true,
		HTTPStatus: result.HTTPStatus,
		Details:    detailsResponse,
	})
}

func validateExternalAPIExpression(validation externalapi.ResponseValidation) error {
	if validation.Mode == externalapi.ValidationModeResponseBody || validation.Mode == externalapi.ValidationModeBoth {
		if err := external.ValidateResponseExpression(validation.Body.Expression); err != nil {
			return apierror.NewBadRequest(err.Error(), "error.invalidExternalAPIResponseValidation")
		}
	}
	return nil
}

func buildExternalAPIExpressionProfile() dto.ExternalAPIExpressionProfileResponse {
	return dto.ExternalAPIExpressionProfileResponse{
		Keywords: []string{"and", "or", "not", "true", "false", "null", "nil"},
		Symbols: []dto.ExternalAPIExpressionSymbolResponse{
			{
				Path:        "response",
				Type:        "object",
				Description: "Response envelope. Use response.status, response.header and response.body.",
			},
			{
				Path:        "response.status",
				Type:        "number",
				Description: "HTTP status code returned by the upstream service.",
			},
			{
				Path:        "response.header",
				Type:        "object",
				Description: "Normalized response headers. Access with response.header[\"x-header\"].",
			},
			{
				Path:        "response.body",
				Type:        "unknown",
				Description: "Parsed response body. Members depend on the configured response schema/sample.",
			},
		},
		Actions: []dto.ExternalAPIExpressionActionResponse{
			{
				ID:        "bool-eq-true",
				Label:     "== true",
				Category:  "comparison",
				AppliesTo: []string{"boolean"},
				Template:  "{{path}} == true",
				Priority:  100,
			},
			{
				ID:        "bool-eq-false",
				Label:     "== false",
				Category:  "comparison",
				AppliesTo: []string{"boolean"},
				Template:  "{{path}} == false",
				Priority:  99,
			},
			{
				ID:        "bool-ne-true",
				Label:     "!= true",
				Category:  "comparison",
				AppliesTo: []string{"boolean"},
				Template:  "{{path}} != true",
				Priority:  98,
			},
			{
				ID:        "bool-ne-false",
				Label:     "!= false",
				Category:  "comparison",
				AppliesTo: []string{"boolean"},
				Template:  "{{path}} != false",
				Priority:  97,
			},
			{
				ID:        "string-eq-empty",
				Label:     "== \"\"",
				Category:  "comparison",
				AppliesTo: []string{"string"},
				Template:  "{{path}} == \"\"",
				Priority:  100,
			},
			{
				ID:        "string-ne-empty",
				Label:     "!= \"\"",
				Category:  "comparison",
				AppliesTo: []string{"string"},
				Template:  "{{path}} != \"\"",
				Priority:  99,
			},
			{
				ID:        "string-contains",
				Label:     "contains \"\"",
				Detail:    "Expr infix string containment operator",
				Category:  "string-op",
				AppliesTo: []string{"string"},
				Template:  "{{path}} contains \"\"",
				Priority:  96,
			},
			{
				ID:        "string-starts-with",
				Label:     "startsWith \"\"",
				Detail:    "Expr infix prefix operator",
				Category:  "string-op",
				AppliesTo: []string{"string"},
				Template:  "{{path}} startsWith \"\"",
				Priority:  95,
			},
			{
				ID:        "string-ends-with",
				Label:     "endsWith \"\"",
				Detail:    "Expr infix suffix operator",
				Category:  "string-op",
				AppliesTo: []string{"string"},
				Template:  "{{path}} endsWith \"\"",
				Priority:  94,
			},
			{
				ID:        "string-matches",
				Label:     "matches \"\"",
				Detail:    "Expr infix regex operator",
				Category:  "string-op",
				AppliesTo: []string{"string"},
				Template:  "{{path}} matches \"\"",
				Priority:  93,
			},
			{
				ID:        "number-eq",
				Label:     "== 0",
				Category:  "comparison",
				AppliesTo: []string{"number"},
				Template:  "{{path}} == 0",
				Priority:  100,
			},
			{
				ID:        "number-ne",
				Label:     "!= 0",
				Category:  "comparison",
				AppliesTo: []string{"number"},
				Template:  "{{path}} != 0",
				Priority:  99,
			},
			{
				ID:        "number-gt",
				Label:     "> 0",
				Category:  "comparison",
				AppliesTo: []string{"number"},
				Template:  "{{path}} > 0",
				Priority:  98,
			},
			{
				ID:        "number-gte",
				Label:     ">= 0",
				Category:  "comparison",
				AppliesTo: []string{"number"},
				Template:  "{{path}} >= 0",
				Priority:  97,
			},
			{
				ID:        "number-lt",
				Label:     "< 0",
				Category:  "comparison",
				AppliesTo: []string{"number"},
				Template:  "{{path}} < 0",
				Priority:  96,
			},
			{
				ID:        "number-lte",
				Label:     "<= 0",
				Category:  "comparison",
				AppliesTo: []string{"number"},
				Template:  "{{path}} <= 0",
				Priority:  95,
			},
			{
				ID:        "array-len",
				Label:     "len(...) > 0",
				Detail:    "Preferred Expr length check for arrays",
				Category:  "array-op",
				AppliesTo: []string{"array"},
				Template:  "len({{path}}) > 0",
				Priority:  100,
			},
			{
				ID:        "array-index",
				Label:     "[0]",
				Category:  "navigation",
				AppliesTo: []string{"array"},
				Template:  "{{path}}[0]",
				Priority:  95,
			},
			{
				ID:        "nullable-eq-null",
				Label:     "== null",
				Category:  "literal",
				AppliesTo: []string{"boolean", "string", "number", "array", "object", "unknown", "null"},
				Template:  "{{path}} == null",
				Priority:  80,
			},
			{
				ID:        "nullable-ne-null",
				Label:     "!= null",
				Category:  "literal",
				AppliesTo: []string{"boolean", "string", "number", "array", "object", "unknown", "null"},
				Template:  "{{path}} != null",
				Priority:  79,
			},
		},
	}
}

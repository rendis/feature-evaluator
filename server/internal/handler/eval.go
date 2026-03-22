package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/rendis/feature-evaluator/internal/domain/evaluation"
	evalmetrics "github.com/rendis/feature-evaluator/internal/domain/metrics"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
)

// swagger type resolution
var _ dto.ErrorResponse

// EvalHandler handles feature evaluation endpoints.
type EvalHandler struct {
	svc       *evaluation.Service
	collector *evalmetrics.Collector
}

// NewEvalHandler creates a new EvalHandler.
func NewEvalHandler(svc *evaluation.Service, collector *evalmetrics.Collector) *EvalHandler {
	return &EvalHandler{svc: svc, collector: collector}
}

// Evaluate godoc
// @Summary Evaluate a single feature
// @Description Evaluates a feature flag for the given context and returns the result value
// @Tags evaluation
// @Accept json
// @Produce json
// @Param request body evaluation.Request true "Evaluation request with feature key and context"
// @Param X-Environment header string false "Environment override (takes precedence over body)"
// @Param X-Tenant-Id header string false "Tenant ID fallback (used if not in context)"
// @Param X-Campus-Id header string false "Campus ID fallback (used if not in context)"
// @Param X-Program-Id header string false "Program ID fallback (used if not in context)"
// @Success 200 {object} evaluation.Result
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /eval [post]
func (h *EvalHandler) Evaluate(c *gin.Context) {
	var req evaluation.Request
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	var rawBody map[string]any
	_ = c.ShouldBindBodyWith(&rawBody, binding.JSON)

	// Backward compat: convert legacy "user" field to context
	if req.NormalizeContext() {
		slog.Warn("deprecated eval request format: 'user' field should be moved to 'context.user'",
			"requestId", middleware.GetRequestID(c),
		)
	}

	// Header fallback: merge tenant/campus/program from headers if not in context
	mergeHeaderFallbacks(c, req.Context)

	evalCtx := evaluation.EvalContext{
		Context:     req.Context,
		Input:       buildExternalInput(c, rawBody),
		RequestID:   middleware.GetRequestID(c),
		Environment: resolveEnvironment(c, req.Environment),
	}

	result := h.svc.Evaluate(c.Request.Context(), req, evalCtx)
	if result.Error != nil && result.Error.Code == "UNAUTHORIZED" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{
			"code":    result.Error.Code,
			"message": result.Error.Message,
		}})
		return
	}
	if h.collector != nil {
		h.collector.Record(evalmetrics.Event{
			FeatureKey:  req.FeatureKey,
			Reason:      string(result.Reason),
			TenantID:    extractTenantID(evalCtx.Context),
			Environment: evalCtx.Environment,
			HasError:    result.Error != nil,
		})
	}
	c.JSON(http.StatusOK, result)
}

// BulkEvaluate godoc
// @Summary Evaluate multiple features
// @Description Evaluates multiple feature flags in a single request and returns all results
// @Tags evaluation
// @Accept json
// @Produce json
// @Param request body evaluation.BulkRequest true "Bulk evaluation request with feature list and shared context"
// @Param X-Environment header string false "Environment override (takes precedence over body)"
// @Param X-Tenant-Id header string false "Tenant ID fallback (used if not in context)"
// @Param X-Campus-Id header string false "Campus ID fallback (used if not in context)"
// @Param X-Program-Id header string false "Program ID fallback (used if not in context)"
// @Success 200 {object} evaluation.BulkResult
// @Failure 400 {object} dto.ErrorResponse
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /eval/bulk [post]
func (h *EvalHandler) BulkEvaluate(c *gin.Context) {
	var req evaluation.BulkRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	var rawBody map[string]any
	_ = c.ShouldBindBodyWith(&rawBody, binding.JSON)

	// Normalize all requests
	requestID := middleware.GetRequestID(c)
	for i := range req.Features {
		if req.Features[i].NormalizeContext() {
			slog.Warn("deprecated eval request format: 'user' field should be moved to 'context.user'",
				"requestId", requestID,
			)
		}
		mergeHeaderFallbacks(c, req.Features[i].Context)
	}

	// Resolve environment: header takes precedence, then fall back to per-request body field
	headerEnv := c.GetHeader("X-Environment")

	evalCtx := evaluation.EvalContext{
		Context:     req.Features[0].Context,
		Input:       buildExternalInput(c, rawBody),
		RequestID:   requestID,
		Environment: resolveEnvironment(c, req.Features[0].Environment),
	}

	// For bulk, each request may have its own environment in the body
	// but header always overrides
	if headerEnv != "" {
		for i := range req.Features {
			req.Features[i].Environment = headerEnv
		}
	}

	result := h.svc.BulkEvaluate(c.Request.Context(), req, evalCtx)
	if h.collector != nil {
		for i, r := range result.Results {
			env := evalCtx.Environment
			if i < len(req.Features) && req.Features[i].Environment != "" {
				env = req.Features[i].Environment
			}
			h.collector.Record(evalmetrics.Event{
				FeatureKey:  r.FeatureKey,
				Reason:      string(r.Reason),
				TenantID:    extractTenantID(evalCtx.Context),
				Environment: env,
				HasError:    r.Error != nil,
			})
		}
	}
	c.JSON(http.StatusOK, result)
}

// EvaluateAll godoc
// @Summary Evaluate all enabled features
// @Description Evaluates all enabled feature flags and returns only the active ones
// @Tags evaluation
// @Accept json
// @Produce json
// @Param request body evaluation.AllRequest true "Context for evaluating all features"
// @Param X-Environment header string false "Environment override (takes precedence over body)"
// @Param X-Tenant-Id header string false "Tenant ID fallback (used if not in context)"
// @Param X-Campus-Id header string false "Campus ID fallback (used if not in context)"
// @Param X-Program-Id header string false "Program ID fallback (used if not in context)"
// @Success 200 {object} evaluation.AllResult
// @Failure 400 {object} dto.ErrorResponse
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /eval/active [post]
func (h *EvalHandler) EvaluateAll(c *gin.Context) {
	var req evaluation.AllRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "BAD_REQUEST", "message": err.Error()}})
		return
	}
	var rawBody map[string]any
	_ = c.ShouldBindBodyWith(&rawBody, binding.JSON)

	if req.Context == nil {
		req.Context = map[string]any{}
	}

	mergeHeaderFallbacks(c, req.Context)

	evalCtx := evaluation.EvalContext{
		Context:     req.Context,
		Input:       buildExternalInput(c, rawBody),
		RequestID:   middleware.GetRequestID(c),
		Environment: resolveEnvironment(c, req.Environment),
	}

	result := h.svc.EvaluateAll(c.Request.Context(), req, evalCtx)
	if h.collector != nil {
		for _, r := range result.Features {
			h.collector.Record(evalmetrics.Event{
				FeatureKey:  r.FeatureKey,
				Reason:      string(r.Reason),
				TenantID:    extractTenantID(evalCtx.Context),
				Environment: evalCtx.Environment,
				HasError:    r.Error != nil,
			})
		}
	}
	c.JSON(http.StatusOK, result)
}

// mergeHeaderFallbacks merges tenant/campus/program from request headers
// into the context map if not already present.
func mergeHeaderFallbacks(c *gin.Context, ctx map[string]any) {
	mergeHeaderNamespace(ctx, "tenant", middleware.GetTenantID(c))
	mergeHeaderNamespace(ctx, "campus", middleware.GetCampusID(c))
	mergeHeaderNamespace(ctx, "program", middleware.GetProgramID(c))
}

// mergeHeaderNamespace sets ctx[namespace] = {"id": headerValue} if
// the namespace is not already present and headerValue is non-empty.
func mergeHeaderNamespace(ctx map[string]any, namespace, headerValue string) {
	if headerValue == "" {
		return
	}
	if _, ok := ctx[namespace]; ok {
		return
	}
	ctx[namespace] = map[string]any{"id": headerValue}
}

// resolveEnvironment returns the environment: header X-Environment takes
// precedence, then the body field value.
func resolveEnvironment(c *gin.Context, bodyEnv string) string {
	if h := c.GetHeader("X-Environment"); h != "" {
		return h
	}
	return bodyEnv
}

// extractTenantID extracts the tenant ID from the evaluation context map.
func extractTenantID(ctx map[string]any) string {
	if ctx == nil {
		return ""
	}
	ns, ok := ctx["tenant"]
	if !ok {
		return ""
	}
	m, ok := ns.(map[string]any)
	if !ok {
		return ""
	}
	if v, ok := m["id"]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

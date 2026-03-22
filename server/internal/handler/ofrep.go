package handler

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/rendis/feature-evaluator/internal/domain/evaluation"
	evalmetrics "github.com/rendis/feature-evaluator/internal/domain/metrics"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
)

// OFREPHandler handles OpenFeature Remote Evaluation Protocol endpoints.
type OFREPHandler struct {
	evalSvc   *evaluation.Service
	collector *evalmetrics.Collector
}

// NewOFREPHandler creates a new OFREPHandler.
func NewOFREPHandler(evalSvc *evaluation.Service, collector *evalmetrics.Collector) *OFREPHandler {
	return &OFREPHandler{evalSvc: evalSvc, collector: collector}
}

// EvaluateSingle godoc
// @Summary Evaluate a single flag (OFREP)
// @Description OpenFeature Remote Evaluation Protocol endpoint for evaluating a single flag. NOTE: This endpoint is served under /ofrep/v1, not under the /features BasePath.
// @Tags ofrep
// @Accept json
// @Produce json
// @Param key path string true "Feature flag key"
// @Param request body dto.OFREPEvalRequest true "OFREP evaluation request with context"
// @Param X-Environment header string false "Environment override"
// @Success 200 {object} dto.OFREPEvalResponse
// @Failure 400 {object} dto.OFREPErrorResponse
// @Failure 401 {object} dto.OFREPErrorResponse
// @Failure 404 {object} dto.OFREPErrorResponse
// @Failure 500 {object} dto.OFREPErrorResponse
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /ofrep/v1/evaluate/flags/{key} [post]
func (h *OFREPHandler) EvaluateSingle(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, dto.OFREPErrorResponse{
			Key:          "",
			ErrorCode:    "PARSE_ERROR",
			ErrorDetails: "missing flag key in path",
		})
		return
	}

	var req dto.OFREPEvalRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, dto.OFREPErrorResponse{
			Key:          key,
			ErrorCode:    "PARSE_ERROR",
			ErrorDetails: err.Error(),
		})
		return
	}
	var rawBody map[string]any
	_ = c.ShouldBindBodyWith(&rawBody, binding.JSON)

	evalContext, errResp := h.buildEvalContext(c, req.Context, rawBody, key)
	if errResp != nil {
		c.JSON(errResp.status, errResp.body)
		return
	}

	evalReq := evaluation.Request{
		FeatureKey:  key,
		Context:     evalContext.Context,
		Environment: evalContext.Environment,
	}

	result := h.evalSvc.Evaluate(c.Request.Context(), evalReq, *evalContext)
	h.recordMetrics(result, *evalContext)

	if result.Error != nil {
		httpStatus := http.StatusInternalServerError
		errorCode := "GENERAL"
		switch result.Error.Code {
		case "FEATURE_NOT_FOUND":
			httpStatus = http.StatusNotFound
			errorCode = "FLAG_NOT_FOUND"
		case "UNAUTHORIZED":
			httpStatus = http.StatusUnauthorized
		}
		c.JSON(httpStatus, dto.OFREPErrorResponse{
			Key:          key,
			ErrorCode:    errorCode,
			ErrorDetails: result.Error.Message,
		})
		return
	}

	c.JSON(http.StatusOK, dto.ToOFREPResponse(result))
}

// EvaluateBulk godoc
// @Summary Evaluate all flags in bulk (OFREP)
// @Description OpenFeature Remote Evaluation Protocol endpoint for bulk flag evaluation. Returns all enabled flags with ETag support. NOTE: This endpoint is served under /ofrep/v1, not under the /features BasePath.
// @Tags ofrep
// @Accept json
// @Produce json
// @Param request body dto.OFREPBulkRequest true "OFREP bulk evaluation request with context"
// @Param If-None-Match header string false "ETag for conditional request (returns 304 if unchanged)"
// @Param X-Environment header string false "Environment override"
// @Success 200 {object} dto.OFREPBulkResponse
// @Failure 304 "Not Modified"
// @Failure 400 {object} dto.OFREPBulkErrorResponse
// @Security BearerAuth
// @Security ApiKeyAuth
// @Router /ofrep/v1/evaluate/flags [post]
func (h *OFREPHandler) EvaluateBulk(c *gin.Context) {
	var req dto.OFREPBulkRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, dto.OFREPBulkErrorResponse{
			ErrorCode:    "PARSE_ERROR",
			ErrorDetails: err.Error(),
		})
		return
	}
	var rawBody map[string]any
	_ = c.ShouldBindBodyWith(&rawBody, binding.JSON)

	evalContext, errResp := h.buildBulkEvalContext(c, req.Context, rawBody)
	if errResp != nil {
		c.JSON(errResp.status, errResp.body)
		return
	}

	allReq := evaluation.AllRequest{
		Context:     evalContext.Context,
		Environment: evalContext.Environment,
	}

	allResult := h.evalSvc.EvaluateAll(c.Request.Context(), allReq, *evalContext)

	flags := make([]any, 0, len(allResult.Features))
	for _, result := range allResult.Features {
		h.recordMetrics(result, *evalContext)
		if result.Error != nil {
			flags = append(flags, dto.OFREPErrorResponse{
				Key:          result.FeatureKey,
				ErrorCode:    "GENERAL",
				ErrorDetails: result.Error.Message,
			})
		} else {
			flags = append(flags, dto.ToOFREPResponse(result))
		}
	}

	etag := computeETag(flags)

	if ifNoneMatch := c.GetHeader("If-None-Match"); ifNoneMatch == etag {
		c.Status(http.StatusNotModified)
		return
	}

	c.Header("ETag", etag)
	c.JSON(http.StatusOK, dto.OFREPBulkResponse{
		Flags: flags,
	})
}

type ofrepErrResponse struct {
	status int
	body   any
}

func (h *OFREPHandler) buildEvalContext(
	c *gin.Context,
	ctx map[string]any,
	rawBody map[string]any,
	flagKey string,
) (*evaluation.EvalContext, *ofrepErrResponse) {
	if ctx == nil {
		ctx = map[string]any{}
	}

	targetingKey, _ := ctx["targetingKey"].(string)
	if targetingKey == "" {
		return nil, &ofrepErrResponse{
			status: http.StatusBadRequest,
			body: dto.OFREPErrorResponse{
				Key:          flagKey,
				ErrorCode:    "TARGETING_KEY_MISSING",
				ErrorDetails: "context.targetingKey is required",
			},
		}
	}

	// Map OFREP flat context to our namespaced context format.
	evalCtxMap := mapOFREPContext(ctx)

	// Merge header fallbacks (tenant/campus/program).
	mergeHeaderFallbacks(c, evalCtxMap)

	return &evaluation.EvalContext{
		Context:     evalCtxMap,
		Input:       buildExternalInput(c, rawBody),
		RequestID:   middleware.GetRequestID(c),
		Environment: resolveEnvironment(c, ""),
	}, nil
}

func (h *OFREPHandler) buildBulkEvalContext(
	c *gin.Context,
	ctx map[string]any,
	rawBody map[string]any,
) (*evaluation.EvalContext, *ofrepErrResponse) {
	if ctx == nil {
		ctx = map[string]any{}
	}

	targetingKey, _ := ctx["targetingKey"].(string)
	if targetingKey == "" {
		return nil, &ofrepErrResponse{
			status: http.StatusBadRequest,
			body: dto.OFREPBulkErrorResponse{
				ErrorCode:    "TARGETING_KEY_MISSING",
				ErrorDetails: "context.targetingKey is required",
			},
		}
	}

	evalCtxMap := mapOFREPContext(ctx)
	mergeHeaderFallbacks(c, evalCtxMap)

	return &evaluation.EvalContext{
		Context:     evalCtxMap,
		Input:       buildExternalInput(c, rawBody),
		RequestID:   middleware.GetRequestID(c),
		Environment: resolveEnvironment(c, ""),
	}, nil
}

// mapOFREPContext maps an OFREP flat context to the internal namespaced format.
// targetingKey -> user.id, tenantId -> tenant.id, campusId -> campus.id,
// programId -> program.id. Remaining flat keys go under user namespace.
func mapOFREPContext(ctx map[string]any) map[string]any {
	// Check if context already has namespaced structure (nested objects).
	// If so, use as-is but ensure targetingKey maps to user.id.
	for _, v := range ctx {
		if _, ok := v.(map[string]any); ok {
			return mapNamespacedOFREPContext(ctx)
		}
	}

	// Flat context: map known keys to namespaces.
	result := map[string]any{}
	userMap := map[string]any{}

	for k, v := range ctx {
		switch k {
		case "targetingKey":
			userMap["id"] = v
		case "tenantId":
			result["tenant"] = map[string]any{"id": v}
		case "campusId":
			result["campus"] = map[string]any{"id": v}
		case "programId":
			result["program"] = map[string]any{"id": v}
		default:
			userMap[k] = v
		}
	}

	result["user"] = userMap
	return result
}

func mapNamespacedOFREPContext(ctx map[string]any) map[string]any {
	result := map[string]any{}

	for k, v := range ctx {
		if nested, ok := v.(map[string]any); ok {
			result[k] = nested
		}
	}

	targetingKey, _ := ctx["targetingKey"].(string)
	if targetingKey == "" {
		return result
	}

	userNS, ok := result["user"].(map[string]any)
	if !ok {
		userNS = map[string]any{}
	}
	if _, hasID := userNS["id"]; !hasID {
		userNS["id"] = targetingKey
	}
	result["user"] = userNS

	return result
}

func (h *OFREPHandler) recordMetrics(result evaluation.Result, evalCtx evaluation.EvalContext) {
	if h.collector == nil {
		return
	}
	h.collector.Record(evalmetrics.Event{
		FeatureKey:  result.FeatureKey,
		Reason:      string(result.Reason),
		TenantID:    extractTenantID(evalCtx.Context),
		Environment: evalCtx.Environment,
		HasError:    result.Error != nil,
	})
}

func computeETag(flags []any) string {
	data, err := json.Marshal(flags)
	if err != nil {
		slog.Warn("computing OFREP ETag", "error", err)
		return ""
	}
	hash := sha256.Sum256(data)
	return fmt.Sprintf(`"%x"`, hash)
}

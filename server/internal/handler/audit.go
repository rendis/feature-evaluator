package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/audit"
	"github.com/rendis/feature-evaluator/internal/dto"
)

// AuditHandler handles audit/evaluation error endpoints.
type AuditHandler struct {
	svc *audit.Service
}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler(svc *audit.Service) *AuditHandler {
	return &AuditHandler{svc: svc}
}

// ListErrors godoc
// @Summary List evaluation errors
// @Description Returns a paginated list of evaluation errors, optionally filtered by feature, tenant, error type, and date range
// @Tags audit
// @Produce json
// @Param featureKey query string false "Filter by feature key"
// @Param tenantId query string false "Filter by tenant ID"
// @Param errorType query string false "Filter by error type"
// @Param from query string false "Start date (RFC3339)"
// @Param to query string false "End date (RFC3339)"
// @Param page query int false "Page number (default 1)"
// @Param pageSize query int false "Page size (default 20, max 100)"
// @Success 200 {object} dto.ListResponse[dto.AuditErrorResponse]
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/audit/errors [get]
func (h *AuditHandler) ListErrors(c *gin.Context) {
	params := audit.ListParams{
		FeatureKey: c.Query("featureKey"),
		TenantID:   c.Query("tenantId"),
		ErrorType:  c.Query("errorType"),
		From:       c.Query("from"),
		To:         c.Query("to"),
	}
	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		params.Page = page
	}
	if ps, err := strconv.Atoi(c.DefaultQuery("pageSize", "20")); err == nil {
		params.PageSize = ps
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	result, err := h.svc.List(c.Request.Context(), params)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	data := make([]dto.AuditErrorResponse, 0, len(result.Data))
	for i := range result.Data {
		data = append(data, dto.ToAuditErrorResponse(&result.Data[i]))
	}

	c.JSON(http.StatusOK, dto.ListResponse[dto.AuditErrorResponse]{
		Data: data,
		Pagination: dto.PaginationResponse{
			Page:       result.Page,
			PageSize:   result.PageSize,
			Total:      result.Total,
			TotalPages: result.TotalPages,
		},
	})
}

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

// ListErrors returns a paginated list of evaluation errors.
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

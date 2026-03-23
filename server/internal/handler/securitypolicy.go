package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/securitypolicy"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
)

// SecurityPolicyHandler handles the global security policy endpoints.
type SecurityPolicyHandler struct {
	svc *securitypolicy.Service
}

// NewSecurityPolicyHandler creates a new SecurityPolicyHandler.
func NewSecurityPolicyHandler(svc *securitypolicy.Service) *SecurityPolicyHandler {
	return &SecurityPolicyHandler{svc: svc}
}

// Get godoc
// @Summary Get global security policy
// @Description Returns inherited, managed, and effective security policy values
// @Tags security-policy
// @Produce json
// @Success 200 {object} dto.SecurityPolicyResponse
// @Failure 403 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/system/security-policy [get]
func (h *SecurityPolicyHandler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, toSecurityPolicyResponse(h.svc.Snapshot()))
}

// Update godoc
// @Summary Update global security policy
// @Description Replaces the app-managed global security policy lists
// @Tags security-policy
// @Accept json
// @Produce json
// @Param request body dto.UpdateSecurityPolicyRequest true "Security policy payload"
// @Success 200 {object} dto.SecurityPolicyResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/system/security-policy [put]
func (h *SecurityPolicyHandler) Update(c *gin.Context) {
	var req dto.UpdateSecurityPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	snapshot, err := h.svc.Update(c.Request.Context(), securitypolicy.ManagedPolicy{
		CORSOrigins:           req.CORSOrigins,
		ExternalAPIAllowHosts: req.ExternalAPIAllowHosts,
		UpdatedBy:             middleware.GetUserEmail(c),
	})
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, toSecurityPolicyResponse(snapshot))
}

func toSecurityPolicyResponse(snapshot securitypolicy.Snapshot) dto.SecurityPolicyResponse {
	response := dto.SecurityPolicyResponse{
		CORSOrigins: dto.SecurityPolicyListResponse{
			Managed:   nonNilStrings(snapshot.CORSOrigins.Managed),
			Inherited: nonNilStrings(snapshot.CORSOrigins.Inherited),
			Effective: nonNilStrings(snapshot.CORSOrigins.Effective),
		},
		ExternalAPIAllowHosts: dto.SecurityPolicyListResponse{
			Managed:   nonNilStrings(snapshot.ExternalAPIAllowHosts.Managed),
			Inherited: nonNilStrings(snapshot.ExternalAPIAllowHosts.Inherited),
			Effective: nonNilStrings(snapshot.ExternalAPIAllowHosts.Effective),
		},
		UpdatedBy: snapshot.UpdatedBy,
	}
	if !snapshot.UpdatedAt.IsZero() {
		formatted := snapshot.UpdatedAt.UTC().Format(time.RFC3339)
		response.UpdatedAt = &formatted
	}

	return response
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}

	return values
}

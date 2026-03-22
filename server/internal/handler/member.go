package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/member"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
)

// MemberHandler handles member CRUD endpoints.
type MemberHandler struct {
	svc *member.Service
}

// NewMemberHandler creates a new MemberHandler.
func NewMemberHandler(svc *member.Service) *MemberHandler {
	return &MemberHandler{svc: svc}
}

// List godoc
// @Summary      List members
// @Description  Returns all team members in the current workspace
// @Tags         members
// @Produce      json
// @Success      200  {object}  dto.DataResponse[[]dto.MemberResponse]
// @Failure      500  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/members [get]
func (h *MemberHandler) List(c *gin.Context) {
	members, err := h.svc.List(c.Request.Context())
	if err != nil {
		slog.Error("listing members", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	resp := make([]dto.MemberResponse, 0, len(members))
	for i := range members {
		resp = append(resp, dto.ToMemberResponse(&members[i]))
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// GetMe godoc
// @Summary      Get current member
// @Description  Returns the authenticated user's member information
// @Tags         members
// @Produce      json
// @Success      200  {object}  dto.MemberResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/members/me [get]
func (h *MemberHandler) GetMe(c *gin.Context) {
	email := middleware.GetUserEmail(c)
	m, err := h.svc.GetByEmail(c.Request.Context(), email)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToMemberResponse(m))
}

// Create godoc
// @Summary      Create member
// @Description  Registers a new team member
// @Tags         members
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateMemberRequest  true  "Member creation payload"
// @Success      201   {object}  dto.MemberResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      409   {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/members [post]
func (h *MemberHandler) Create(c *gin.Context) {
	var req dto.CreateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	m := &member.Member{
		Email:       req.Email,
		Role:        member.Role(req.Role),
		DisplayName: req.DisplayName,
		AddedBy:     middleware.GetUserEmail(c),
	}

	if err := h.svc.Create(c.Request.Context(), m); err != nil {
		slog.Error("creating member", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.ToMemberResponse(m))
}

// UpdateRole godoc
// @Summary      Update member role
// @Description  Changes a member's role
// @Tags         members
// @Accept       json
// @Produce      json
// @Param        id    path      string                true  "Member ID"
// @Param        body  body      dto.UpdateRoleRequest  true  "Role update payload"
// @Success      200   {object}  dto.MessageResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/members/{id}/role [put]
func (h *MemberHandler) UpdateRole(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	actorEmail := middleware.GetUserEmail(c)
	if err := h.svc.UpdateRole(c.Request.Context(), id, member.Role(req.Role), actorEmail); err != nil {
		slog.Error("updating member role", "error", err, "memberId", id, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role updated"})
}

// Delete godoc
// @Summary      Delete member
// @Description  Removes a team member
// @Tags         members
// @Produce      json
// @Param        id   path      string  true  "Member ID"
// @Success      200  {object}  dto.MessageResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/members/{id} [delete]
func (h *MemberHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		slog.Error("deleting member", "error", err, "memberId", id, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "member deleted"})
}

// TransferOwnership godoc
// @Summary      Transfer ownership
// @Description  Transfers the owner role from one member to another
// @Tags         members
// @Accept       json
// @Produce      json
// @Param        id    path      string                          true  "Current owner member ID"
// @Param        body  body      dto.TransferOwnershipRequest     true  "Transfer payload"
// @Success      200   {object}  dto.MessageResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/members/{id}/transfer-ownership [post]
func (h *MemberHandler) TransferOwnership(c *gin.Context) {
	fromID := c.Param("id")
	var req dto.TransferOwnershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	if err := h.svc.TransferOwnership(c.Request.Context(), fromID, req.ToMemberID); err != nil {
		slog.Error("transferring ownership", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ownership transferred"})
}

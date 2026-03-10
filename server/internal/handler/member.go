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

// List returns all team members.
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

// GetMe returns the current authenticated user's member info.
func (h *MemberHandler) GetMe(c *gin.Context) {
	email := middleware.GetUserEmail(c)
	m, err := h.svc.GetByEmail(c.Request.Context(), email)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.ToMemberResponse(m))
}

// Create registers a new team member.
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

// UpdateRole changes a member's role.
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

// Delete removes a team member.
func (h *MemberHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		slog.Error("deleting member", "error", err, "memberId", id, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "member deleted"})
}

// TransferOwnership transfers the owner role to another member.
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

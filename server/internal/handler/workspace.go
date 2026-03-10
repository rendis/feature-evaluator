package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/member"
	"github.com/rendis/feature-evaluator/internal/domain/workspace"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
	"github.com/rendis/feature-evaluator/pkg/apierror"
)

// WorkspaceHandler handles workspace CRUD endpoints.
type WorkspaceHandler struct {
	svc       *workspace.Service
	memberSvc *member.Service
}

// NewWorkspaceHandler creates a new WorkspaceHandler.
func NewWorkspaceHandler(svc *workspace.Service, memberSvc *member.Service) *WorkspaceHandler {
	return &WorkspaceHandler{svc: svc, memberSvc: memberSvc}
}

// List returns all workspaces.
func (h *WorkspaceHandler) List(c *gin.Context) {
	includeArchived, _ := strconv.ParseBool(c.DefaultQuery("includeArchived", "false"))
	workspaces, err := h.svc.List(c.Request.Context(), includeArchived)
	if err != nil {
		dto.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": workspaces})
}

// Get returns a single workspace by key.
func (h *WorkspaceHandler) Get(c *gin.Context) {
	key := c.Param("key")
	ws, err := h.svc.GetByKey(c.Request.Context(), key)
	if err != nil {
		dto.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, ws)
}

// Create creates a new workspace.
func (h *WorkspaceHandler) Create(c *gin.Context) {
	var req struct {
		Key         string `json:"key" binding:"required"`
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}
	activeCount, err := h.svc.CountActive(c.Request.Context())
	if err != nil {
		dto.RespondError(c, err)
		return
	}
	ws := &workspace.Workspace{
		Key:         req.Key,
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   middleware.GetUserEmail(c),
	}
	if err := h.svc.Create(c.Request.Context(), ws); err != nil {
		dto.RespondError(c, err)
		return
	}
	if activeCount == 0 {
		memberCtx := workspace.WithKey(c.Request.Context(), ws.Key)
		userEmail := middleware.GetUserEmail(c)
		if err := h.memberSvc.Create(memberCtx, &member.Member{
			Email:       userEmail,
			Role:        member.RoleOwner,
			DisplayName: userEmail,
			AddedBy:     userEmail,
		}); err != nil {
			dto.RespondError(c, err)
			return
		}
	}
	c.JSON(http.StatusCreated, ws)
}

// Update updates an existing workspace.
func (h *WorkspaceHandler) Update(c *gin.Context) {
	key := c.Param("key")
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}
	ws, err := h.svc.GetByKey(c.Request.Context(), key)
	if err != nil {
		dto.RespondError(c, err)
		return
	}
	ws.Name = req.Name
	ws.Description = req.Description
	if err := h.svc.Update(c.Request.Context(), ws); err != nil {
		dto.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, ws)
}

// Archive archives a workspace.
func (h *WorkspaceHandler) Archive(c *gin.Context) {
	key := c.Param("key")
	if err := h.svc.Archive(c.Request.Context(), key, middleware.GetUserEmail(c)); err != nil {
		dto.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "workspace archived"})
}

// Restore restores an archived workspace.
func (h *WorkspaceHandler) Restore(c *gin.Context) {
	key := c.Param("key")
	activeCount, err := h.svc.CountActive(c.Request.Context())
	if err != nil {
		dto.RespondError(c, err)
		return
	}
	if err := h.svc.Restore(c.Request.Context(), key); err != nil {
		dto.RespondError(c, err)
		return
	}
	if err := h.ensureBootstrapOwner(c, key, activeCount); err != nil {
		dto.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "workspace restored"})
}

// Delete archives a workspace for backwards compatibility.
func (h *WorkspaceHandler) Delete(c *gin.Context) {
	h.Archive(c)
}

func (h *WorkspaceHandler) ensureBootstrapOwner(c *gin.Context, key string, activeCount int64) error {
	if activeCount != 0 {
		return nil
	}

	memberCtx := workspace.WithKey(c.Request.Context(), key)
	userEmail := middleware.GetUserEmail(c)
	if _, err := h.memberSvc.GetByEmail(memberCtx, userEmail); err != nil {
		var apiErr *apierror.APIError
		if !errors.As(err, &apiErr) || apiErr.Code != apierror.CodeNotFound {
			return err
		}
		return h.memberSvc.Create(memberCtx, &member.Member{
			Email:       userEmail,
			Role:        member.RoleOwner,
			DisplayName: userEmail,
			AddedBy:     userEmail,
		})
	}

	return nil
}

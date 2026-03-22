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

// List godoc
// @Summary List workspaces
// @Description Returns all workspaces, optionally including archived ones
// @Tags workspaces
// @Produce json
// @Param includeArchived query bool false "Include archived workspaces (default false)"
// @Success 200 {object} dto.DataResponse[[]workspace.Workspace]
// @Failure 500 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/workspaces [get]
func (h *WorkspaceHandler) List(c *gin.Context) {
	includeArchived, _ := strconv.ParseBool(c.DefaultQuery("includeArchived", "false"))
	workspaces, err := h.svc.List(c.Request.Context(), includeArchived)
	if err != nil {
		dto.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": workspaces})
}

// Get godoc
// @Summary Get a workspace
// @Description Returns a single workspace by key
// @Tags workspaces
// @Produce json
// @Param key path string true "Workspace key"
// @Success 200 {object} workspace.Workspace
// @Failure 404 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/workspaces/{key} [get]
func (h *WorkspaceHandler) Get(c *gin.Context) {
	key := c.Param("key")
	ws, err := h.svc.GetByKey(c.Request.Context(), key)
	if err != nil {
		dto.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, ws)
}

// Create godoc
// @Summary Create a workspace
// @Description Creates a new workspace; if this is the first workspace the caller is added as owner
// @Tags workspaces
// @Accept json
// @Produce json
// @Param request body object{key=string,name=string,description=string} true "Workspace definition"
// @Success 201 {object} workspace.Workspace
// @Failure 400 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/workspaces [post]
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

// Update godoc
// @Summary Update a workspace
// @Description Updates a workspace's name and description by key
// @Tags workspaces
// @Accept json
// @Produce json
// @Param key path string true "Workspace key"
// @Param request body object{name=string,description=string} true "Updated workspace fields"
// @Success 200 {object} workspace.Workspace
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/workspaces/{key} [put]
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

// Archive godoc
// @Summary Archive a workspace
// @Description Archives a workspace by key, making it inactive
// @Tags workspaces
// @Produce json
// @Param key path string true "Workspace key"
// @Success 200 {object} dto.MessageResponse
// @Failure 404 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/workspaces/{key}/archive [post]
func (h *WorkspaceHandler) Archive(c *gin.Context) {
	key := c.Param("key")
	if err := h.svc.Archive(c.Request.Context(), key, middleware.GetUserEmail(c)); err != nil {
		dto.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "workspace archived"})
}

// Restore godoc
// @Summary Restore a workspace
// @Description Restores an archived workspace by key
// @Tags workspaces
// @Produce json
// @Param key path string true "Workspace key"
// @Success 200 {object} dto.MessageResponse
// @Failure 404 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/workspaces/{key}/restore [post]
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

// Delete godoc
// @Summary Delete a workspace
// @Description Archives a workspace for backwards compatibility (same as Archive)
// @Tags workspaces
// @Produce json
// @Param key path string true "Workspace key"
// @Success 200 {object} dto.MessageResponse
// @Failure 404 {object} dto.ErrorResponse
// @Security BearerAuth
// @Router /admin/workspaces/{key} [delete]
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

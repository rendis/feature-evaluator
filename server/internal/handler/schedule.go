package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/schedule"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
)

// ScheduleHandler handles scheduled change endpoints.
type ScheduleHandler struct {
	svc *schedule.Service
}

// NewScheduleHandler creates a new ScheduleHandler.
func NewScheduleHandler(svc *schedule.Service) *ScheduleHandler {
	return &ScheduleHandler{svc: svc}
}

// Create creates a new scheduled change for a feature.
func (h *ScheduleHandler) Create(c *gin.Context) {
	featureKey := c.Param("key")

	var req struct {
		ChangeType  schedule.ChangeType `json:"changeType" binding:"required"`
		Payload     map[string]any      `json:"payload" binding:"required"`
		ScheduledAt time.Time           `json:"scheduledAt" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	sc := &schedule.ScheduledChange{
		FeatureKey:  featureKey,
		ChangeType:  req.ChangeType,
		Payload:     req.Payload,
		ScheduledAt: req.ScheduledAt.UTC(),
		CreatedBy:   middleware.GetUserEmail(c),
	}

	if err := h.svc.Create(c.Request.Context(), sc); err != nil {
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, sc)
}

// List returns all scheduled changes for a feature.
func (h *ScheduleHandler) List(c *gin.Context) {
	featureKey := c.Param("key")

	changes, err := h.svc.ListByFeature(c.Request.Context(), featureKey)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": changes})
}

// Cancel cancels a pending scheduled change.
func (h *ScheduleHandler) Cancel(c *gin.Context) {
	id := c.Param("id")

	if err := h.svc.Cancel(c.Request.Context(), id); err != nil {
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "schedule cancelled"})
}

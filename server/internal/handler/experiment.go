package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/domain/changelog"
	"github.com/rendis/feature-evaluator/internal/domain/experiment"
	"github.com/rendis/feature-evaluator/internal/dto"
	"github.com/rendis/feature-evaluator/internal/server/middleware"
)

// ExperimentHandler handles experiment CRUD and lifecycle endpoints.
type ExperimentHandler struct {
	svc          *experiment.Service
	changelogSvc *changelog.Service
}

// NewExperimentHandler creates a new ExperimentHandler.
func NewExperimentHandler(svc *experiment.Service, changelogSvc *changelog.Service) *ExperimentHandler {
	return &ExperimentHandler{svc: svc, changelogSvc: changelogSvc}
}

// Create godoc
// @Summary      Create experiment
// @Description  Creates a new A/B experiment for a feature
// @Tags         experiments
// @Accept       json
// @Produce      json
// @Param        body  body      dto.CreateExperimentRequest  true  "Experiment creation payload"
// @Success      201   {object}  dto.ExperimentResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      409   {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/experiments [post]
func (h *ExperimentHandler) Create(c *gin.Context) {
	var req dto.CreateExperimentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	variants := make([]experiment.Variant, 0, len(req.Variants))
	for _, v := range req.Variants {
		variants = append(variants, experiment.Variant{Key: v.Key, Value: v.Value, Weight: v.Weight})
	}

	metrics := make([]experiment.Metric, 0, len(req.Metrics))
	for _, m := range req.Metrics {
		metrics = append(metrics, experiment.Metric{Key: m.Key, Name: m.Name, Description: m.Description})
	}

	exp := &experiment.Experiment{
		FeatureKey:  req.FeatureKey,
		Name:        req.Name,
		Description: req.Description,
		Variants:    variants,
		Metrics:     metrics,
		CreatedBy:   middleware.GetUserEmail(c),
	}

	if err := h.svc.Create(c.Request.Context(), exp); err != nil {
		slog.Error("creating experiment", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntityExperiment,
		EntityKey:  exp.ID,
		Action:     changelog.ActionCreate,
	})

	c.JSON(http.StatusCreated, dto.ToExperimentResponse(exp))
}

// List godoc
// @Summary      List experiments
// @Description  Returns all experiments in the current workspace
// @Tags         experiments
// @Produce      json
// @Success      200  {object}  dto.DataResponse[[]dto.ExperimentResponse]
// @Failure      500  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/experiments [get]
func (h *ExperimentHandler) List(c *gin.Context) {
	experiments, err := h.svc.List(c.Request.Context())
	if err != nil {
		slog.Error("listing experiments", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	data := make([]dto.ExperimentResponse, 0, len(experiments))
	for i := range experiments {
		data = append(data, dto.ToExperimentResponse(&experiments[i]))
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

// Get godoc
// @Summary      Get experiment
// @Description  Returns a single experiment by ID
// @Tags         experiments
// @Produce      json
// @Param        id   path      string  true  "Experiment ID"
// @Success      200  {object}  dto.ExperimentResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/experiments/{id} [get]
func (h *ExperimentHandler) Get(c *gin.Context) {
	id := c.Param("id")
	exp, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		dto.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.ToExperimentResponse(exp))
}

// Update godoc
// @Summary      Update experiment
// @Description  Updates a draft experiment by ID
// @Tags         experiments
// @Accept       json
// @Produce      json
// @Param        id    path      string                       true  "Experiment ID"
// @Param        body  body      dto.UpdateExperimentRequest   true  "Experiment update payload"
// @Success      200   {object}  dto.ExperimentResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/experiments/{id} [put]
func (h *ExperimentHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req dto.UpdateExperimentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	variants := make([]experiment.Variant, 0, len(req.Variants))
	for _, v := range req.Variants {
		variants = append(variants, experiment.Variant{Key: v.Key, Value: v.Value, Weight: v.Weight})
	}

	metrics := make([]experiment.Metric, 0, len(req.Metrics))
	for _, m := range req.Metrics {
		metrics = append(metrics, experiment.Metric{Key: m.Key, Name: m.Name, Description: m.Description})
	}

	exp := &experiment.Experiment{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Variants:    variants,
		Metrics:     metrics,
	}

	if err := h.svc.Update(c.Request.Context(), exp); err != nil {
		slog.Error("updating experiment", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	// Re-fetch the updated experiment to return full response.
	updated, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntityExperiment,
		EntityKey:  id,
		Action:     changelog.ActionUpdate,
	})

	c.JSON(http.StatusOK, dto.ToExperimentResponse(updated))
}

// Start godoc
// @Summary      Start experiment
// @Description  Starts a draft or paused experiment
// @Tags         experiments
// @Produce      json
// @Param        id   path      string  true  "Experiment ID"
// @Success      200  {object}  dto.MessageResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/experiments/{id}/start [post]
func (h *ExperimentHandler) Start(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Start(c.Request.Context(), id); err != nil {
		slog.Error("starting experiment", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntityExperiment,
		EntityKey:  id,
		Action:     changelog.ActionUpdate,
		FieldChanges: []changelog.FieldChange{
			{Field: "status", NewValue: experiment.StatusRunning},
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "experiment started"})
}

// Pause godoc
// @Summary      Pause experiment
// @Description  Pauses a running experiment
// @Tags         experiments
// @Produce      json
// @Param        id   path      string  true  "Experiment ID"
// @Success      200  {object}  dto.MessageResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/experiments/{id}/pause [post]
func (h *ExperimentHandler) Pause(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Pause(c.Request.Context(), id); err != nil {
		slog.Error("pausing experiment", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntityExperiment,
		EntityKey:  id,
		Action:     changelog.ActionUpdate,
		FieldChanges: []changelog.FieldChange{
			{Field: "status", NewValue: experiment.StatusPaused},
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "experiment paused"})
}

// Complete godoc
// @Summary      Complete experiment
// @Description  Completes a running or paused experiment
// @Tags         experiments
// @Produce      json
// @Param        id   path      string  true  "Experiment ID"
// @Success      200  {object}  dto.MessageResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      404  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/experiments/{id}/complete [post]
func (h *ExperimentHandler) Complete(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.Complete(c.Request.Context(), id); err != nil {
		slog.Error("completing experiment", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntityExperiment,
		EntityKey:  id,
		Action:     changelog.ActionUpdate,
		FieldChanges: []changelog.FieldChange{
			{Field: "status", NewValue: experiment.StatusCompleted},
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "experiment completed"})
}

// DeclareWinner godoc
// @Summary      Declare winner
// @Description  Declares a winning variant for a completed experiment
// @Tags         experiments
// @Accept       json
// @Produce      json
// @Param        id    path      string                     true  "Experiment ID"
// @Param        body  body      dto.DeclareWinnerRequest    true  "Winner declaration payload"
// @Success      200   {object}  dto.DeclareWinnerResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/experiments/{id}/declare-winner [post]
func (h *ExperimentHandler) DeclareWinner(c *gin.Context) {
	id := c.Param("id")
	var req dto.DeclareWinnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	if err := h.svc.DeclareWinner(c.Request.Context(), id, req.VariantKey); err != nil {
		slog.Error("declaring winner", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	recordChange(c, h.changelogSvc, &changelog.ChangeEntry{
		EntityType: changelog.EntityExperiment,
		EntityKey:  id,
		Action:     changelog.ActionUpdate,
		FieldChanges: []changelog.FieldChange{
			{Field: "winnerKey", NewValue: req.VariantKey},
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "winner declared", "variantKey": req.VariantKey})
}

// GetResults godoc
// @Summary      Get experiment results
// @Description  Returns computed results and statistics for an experiment
// @Tags         experiments
// @Produce      json
// @Param        id   path      string  true  "Experiment ID"
// @Success      200  {object}  experiment.Results
// @Failure      404  {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Router       /admin/experiments/{id}/results [get]
func (h *ExperimentHandler) GetResults(c *gin.Context) {
	id := c.Param("id")
	results, err := h.svc.GetResults(c.Request.Context(), id)
	if err != nil {
		slog.Error("getting experiment results", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}
	c.JSON(http.StatusOK, results)
}

// RecordConversion godoc
// @Summary      Record conversion
// @Description  Records a conversion event for an experiment metric
// @Tags         experiments
// @Accept       json
// @Produce      json
// @Param        body  body      dto.RecordConversionRequest  true  "Conversion event payload"
// @Success      201   {object}  dto.MessageResponse
// @Failure      400   {object}  dto.ErrorResponse
// @Failure      404   {object}  dto.ErrorResponse
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /eval/conversions [post]
func (h *ExperimentHandler) RecordConversion(c *gin.Context) {
	var req dto.RecordConversionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.RespondError(c, err)
		return
	}

	conv := &experiment.Conversion{
		ExperimentID: req.ExperimentID,
		UserID:       req.UserID,
		MetricKey:    req.MetricKey,
		Value:        req.Value,
	}

	if err := h.svc.RecordConversion(c.Request.Context(), conv); err != nil {
		slog.Error("recording conversion", "error", err, "requestId", middleware.GetRequestID(c))
		dto.RespondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "conversion recorded"})
}

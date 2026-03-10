package dto

import (
	"time"

	"github.com/rendis/feature-evaluator/internal/domain/experiment"
)

// CreateExperimentRequest is the request body for creating an experiment.
type CreateExperimentRequest struct {
	FeatureKey  string           `json:"featureKey" binding:"required"`
	Name        string           `json:"name" binding:"required"`
	Description string           `json:"description"`
	Variants    []VariantRequest `json:"variants" binding:"required,min=2"`
	Metrics     []MetricRequest  `json:"metrics"`
}

// UpdateExperimentRequest is the request body for updating a draft experiment.
type UpdateExperimentRequest struct {
	Name        string           `json:"name" binding:"required"`
	Description string           `json:"description"`
	Variants    []VariantRequest `json:"variants" binding:"required,min=2"`
	Metrics     []MetricRequest  `json:"metrics"`
}

// VariantRequest is a variant in a create/update request.
type VariantRequest struct {
	Key    string `json:"key" binding:"required"`
	Value  any    `json:"value"`
	Weight int    `json:"weight"`
}

// MetricRequest is a metric in a create/update request.
type MetricRequest struct {
	Key         string `json:"key" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// DeclareWinnerRequest is the request body for declaring an experiment winner.
type DeclareWinnerRequest struct {
	VariantKey string `json:"variantKey" binding:"required"`
}

// RecordConversionRequest is the request body for recording a conversion event.
type RecordConversionRequest struct {
	ExperimentID string  `json:"experimentId" binding:"required"`
	UserID       string  `json:"userId" binding:"required"`
	MetricKey    string  `json:"metricKey" binding:"required"`
	Value        float64 `json:"value"`
}

// ExperimentResponse is the response for an experiment.
type ExperimentResponse struct {
	ID           string            `json:"id"`
	WorkspaceKey string            `json:"workspaceKey"`
	FeatureKey   string            `json:"featureKey"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Status       string            `json:"status"`
	Variants     []VariantResponse `json:"variants"`
	Metrics      []MetricResponse  `json:"metrics"`
	WinnerKey    string            `json:"winnerKey,omitempty"`
	StartedAt    *time.Time        `json:"startedAt,omitempty"`
	CompletedAt  *time.Time        `json:"completedAt,omitempty"`
	CreatedBy    string            `json:"createdBy"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

// VariantResponse is a variant in a response.
type VariantResponse struct {
	Key    string `json:"key"`
	Value  any    `json:"value"`
	Weight int    `json:"weight"`
}

// MetricResponse is a metric in a response.
type MetricResponse struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ToExperimentResponse maps an experiment domain model to a response DTO.
func ToExperimentResponse(exp *experiment.Experiment) ExperimentResponse {
	variants := make([]VariantResponse, 0, len(exp.Variants))
	for _, v := range exp.Variants {
		variants = append(variants, VariantResponse{
			Key:    v.Key,
			Value:  v.Value,
			Weight: v.Weight,
		})
	}

	metrics := make([]MetricResponse, 0, len(exp.Metrics))
	for _, m := range exp.Metrics {
		metrics = append(metrics, MetricResponse{
			Key:         m.Key,
			Name:        m.Name,
			Description: m.Description,
		})
	}

	return ExperimentResponse{
		ID:           exp.ID,
		WorkspaceKey: exp.WorkspaceKey,
		FeatureKey:   exp.FeatureKey,
		Name:         exp.Name,
		Description:  exp.Description,
		Status:       string(exp.Status),
		Variants:     variants,
		Metrics:      metrics,
		WinnerKey:    exp.WinnerKey,
		StartedAt:    exp.StartedAt,
		CompletedAt:  exp.CompletedAt,
		CreatedBy:    exp.CreatedBy,
		CreatedAt:    exp.CreatedAt,
		UpdatedAt:    exp.UpdatedAt,
	}
}

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rendis/feature-evaluator/internal/dto"
)

// swagger type resolution
var _ dto.HealthResponse

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	probe *dependencyProbe
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(postgres postgresPinger, redis redisDependencyChecker) *HealthHandler {
	return &HealthHandler{probe: newDependencyProbe(postgres, redis)}
}

// Liveness godoc
// @Summary Liveness probe
// @Description Returns 200 if the server process is running
// @Tags health
// @Produce json
// @Success 200 {object} dto.HealthResponse
// @Router /healthz [get]
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readiness godoc
// @Summary Readiness probe
// @Description Checks PostgreSQL and Redis connectivity and returns component health status
// @Tags health
// @Produce json
// @Success 200 {object} dto.HealthResponse
// @Failure 503 {object} dto.HealthResponse
// @Router /readyz [get]
func (h *HealthHandler) Readiness(c *gin.Context) {
	snapshot := h.probe.Check(c.Request.Context())
	checks := gin.H{}

	if snapshot.PostgreSQL.Status != "healthy" {
		checks["postgresql"] = "unhealthy"
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "checks": checks})
		return
	}
	checks["postgresql"] = "healthy"

	if snapshot.Redis.Status != "healthy" {
		checks["redis"] = "unhealthy (degraded)"
	} else {
		checks["redis"] = "healthy"
	}

	c.JSON(http.StatusOK, gin.H{"status": "ready", "checks": checks})
}

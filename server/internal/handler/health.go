package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	probe *dependencyProbe
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(postgres postgresPinger, redis redisDependencyChecker) *HealthHandler {
	return &HealthHandler{probe: newDependencyProbe(postgres, redis)}
}

// Liveness returns 200 if the server is running.
func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readiness checks PostgreSQL and Redis connectivity.
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

package handler

import (
	"context"
	"time"

	redisclient "github.com/rendis/feature-evaluator/internal/storage/redis"
)

const dependencyCheckTimeout = 2 * time.Second

type postgresPinger interface {
	Ping(ctx context.Context) error
}

type redisDependencyChecker interface {
	Ping(ctx context.Context) error
	CircuitState() redisclient.CircuitState
}

type dependencyProbe struct {
	postgres postgresPinger
	redis    redisDependencyChecker
	timeout  time.Duration
}

type dependencyCheck struct {
	Status    string `json:"status"`
	LatencyMs *int64 `json:"latencyMs"`
}

type redisDependencyCheck struct {
	Status      string     `json:"status"`
	LatencyMs   *int64     `json:"latencyMs"`
	CircuitOpen bool       `json:"circuitOpen"`
	OpenUntil   *time.Time `json:"openUntil"`
}

type dependencySnapshot struct {
	CheckedAt     time.Time            `json:"checkedAt"`
	OverallStatus string               `json:"overallStatus"`
	PostgreSQL    dependencyCheck      `json:"postgresql"`
	Redis         redisDependencyCheck `json:"redis"`
}

func newDependencyProbe(postgres postgresPinger, redis redisDependencyChecker) *dependencyProbe {
	return &dependencyProbe{
		postgres: postgres,
		redis:    redis,
		timeout:  dependencyCheckTimeout,
	}
}

func (p *dependencyProbe) Check(ctx context.Context) dependencySnapshot {
	checkedAt := time.Now().UTC()

	postgresLatency, postgresErr := measurePing(ctx, p.timeout, p.postgres.Ping)
	postgresStatus := dependencyCheck{
		Status:    "healthy",
		LatencyMs: postgresLatency,
	}
	if postgresErr != nil {
		postgresStatus.Status = "unhealthy"
	}

	redisLatency, redisErr := measurePing(ctx, p.timeout, p.redis.Ping)
	redisCircuit := p.redis.CircuitState()
	redisStatus := redisDependencyCheck{
		Status:      "healthy",
		LatencyMs:   redisLatency,
		CircuitOpen: redisCircuit.Open,
		OpenUntil:   timePtr(redisCircuit.OpenUntil),
	}
	if redisErr != nil || redisCircuit.Open {
		redisStatus.Status = "degraded"
	}

	overallStatus := "healthy"
	if postgresStatus.Status != "healthy" {
		overallStatus = "unhealthy"
	} else if redisStatus.Status != "healthy" {
		overallStatus = "degraded"
	}

	return dependencySnapshot{
		CheckedAt:     checkedAt,
		OverallStatus: overallStatus,
		PostgreSQL:    postgresStatus,
		Redis:         redisStatus,
	}
}

func measurePing(
	ctx context.Context,
	timeout time.Duration,
	ping func(context.Context) error,
) (*int64, error) {
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startedAt := time.Now()
	if err := ping(probeCtx); err != nil {
		return nil, err
	}

	latency := time.Since(startedAt).Milliseconds()
	return &latency, nil
}

func timePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}

	t := value.UTC()
	return &t
}

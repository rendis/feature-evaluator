package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/rendis/feature-evaluator/internal/config"
	"github.com/rendis/feature-evaluator/internal/server"
	"github.com/rendis/feature-evaluator/internal/storage/postgres"
	redisclient "github.com/rendis/feature-evaluator/internal/storage/redis"
)

// @title Feature Evaluator API
// @version 1.0
// @description Feature flag system with rule-based evaluation, segment targeting, pack-based feature bundling, and A/B experiments.
// @BasePath /features

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-Api-Key
// @description API key for evaluation or admin access

// @securityDefinitions.apikey WorkspaceHeader
// @in header
// @name X-Workspace
// @description Workspace key for multi-tenant isolation

func main() {
	ctx := context.Background()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("loading config", "error", err)
		os.Exit(1)
	}

	// Setup structured logging
	setupLogging(cfg.Log)

	if err := runStartupPreflight(ctx, cfg); err != nil {
		slog.Error("startup preflight failed", "error", err)
		os.Exit(1)
	}

	// Connect to PostgreSQL
	postgresDB, err := postgres.NewClient(ctx, postgres.Config{
		DatabaseURL:       cfg.Postgres.DatabaseURL,
		MaxConns:          cfg.Postgres.MaxConns,
		MinConns:          cfg.Postgres.MinConns,
		MaxConnLifetime:   cfg.Postgres.MaxConnLifetime,
		MaxConnIdleTime:   cfg.Postgres.MaxConnIdleTime,
		HealthcheckPeriod: cfg.Postgres.HealthcheckPeriod,
		ConnectionTimeout: cfg.Postgres.ConnectTimeout,
	})
	if err != nil {
		slog.Error("connecting to PostgreSQL", "error", err)
		os.Exit(1)
	}

	// Run migrations
	if err := postgres.RunMigrations(ctx, postgresDB); err != nil {
		slog.Error("running migrations", "error", err)
		os.Exit(1)
	}

	// Connect to Redis
	redis, err := redisclient.NewClient(ctx, cfg.Redis.URI, cfg.Redis.RedisPassword)
	if err != nil {
		slog.Error("connecting to Redis", "error", err)
		os.Exit(1)
	}

	// Create and start server
	srv := server.New(cfg, postgresDB, redis)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	slog.Info("server started, waiting for shutdown signal")
	<-quit
	slog.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(ctx, cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	postgresDB.Close()

	if err := redis.Close(); err != nil {
		slog.Error("Redis disconnect error", "error", err)
	}

	slog.Info("server stopped")
}

func setupLogging(cfg config.LogConfig) {
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var h slog.Handler
	if strings.ToLower(cfg.Format) == "json" {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(h))
}

package main

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rendis/feature-evaluator/internal/config"
)

const defaultStartupPreflightTimeout = 3 * time.Second

func runStartupPreflight(ctx context.Context, cfg *config.Config) error {
	if strings.TrimSpace(cfg.Postgres.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required; set it in server/.env before starting the backend")
	}
	if strings.TrimSpace(cfg.Redis.URI) == "" {
		return fmt.Errorf("REDIS_URI is required; set it in server/.env before starting the backend")
	}

	if err := checkPostgresReachable(ctx, cfg.Postgres.DatabaseURL, cfg.Postgres.ConnectTimeout); err != nil {
		return err
	}
	if err := checkRedisReachable(ctx, cfg.Redis.URI); err != nil {
		return err
	}

	return nil
}

func checkPostgresReachable(ctx context.Context, databaseURL string, timeout time.Duration) error {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return fmt.Errorf("invalid DATABASE_URL: %w", err)
	}

	address := net.JoinHostPort(cfg.ConnConfig.Host, strconv.Itoa(int(cfg.ConnConfig.Port)))
	if err := dialAddress(ctx, address, timeout); err != nil {
		return fmt.Errorf(
			"cannot reach PostgreSQL at %s: %w; ensure your local PostgreSQL container is running and DATABASE_URL points to it",
			address,
			err,
		)
	}

	return nil
}

func checkRedisReachable(ctx context.Context, redisURI string) error {
	address := strings.TrimSpace(redisURI)
	if address == "" {
		return fmt.Errorf("REDIS_URI is required")
	}
	if !strings.Contains(address, ":") {
		address = net.JoinHostPort(address, "6379")
	}
	if _, _, err := net.SplitHostPort(address); err != nil {
		return fmt.Errorf("invalid REDIS_URI %q: expected host:port", redisURI)
	}

	if err := dialAddress(ctx, address, defaultStartupPreflightTimeout); err != nil {
		return fmt.Errorf(
			"cannot reach Redis at %s: %w; start Redis locally with `make redis` or update REDIS_URI",
			address,
			err,
		)
	}

	return nil
}

func dialAddress(ctx context.Context, address string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = defaultStartupPreflightTimeout
	}

	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}
	return conn.Close()
}

package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Client wraps the PostgreSQL connection pool.
type Client struct {
	pool *pgxpool.Pool
}

// Config configures the PostgreSQL client.
type Config struct {
	DatabaseURL         string
	MaxConns            int32
	MinConns            int32
	MaxConnLifetime     time.Duration
	HealthcheckPeriod   time.Duration
	MaxConnIdleTime     time.Duration
	ConnectionTimeout   time.Duration
}

// NewClient creates and validates a PostgreSQL pool.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("database url is required")
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing database url: %w", err)
	}

	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	}
	if cfg.MaxConnLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	}
	if cfg.MaxConnIdleTime > 0 {
		poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	}
	if cfg.HealthcheckPeriod > 0 {
		poolCfg.HealthCheckPeriod = cfg.HealthcheckPeriod
	}
	if cfg.ConnectionTimeout > 0 {
		poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectionTimeout
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("connecting to PostgreSQL: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging PostgreSQL: %w", err)
	}

	slog.Info("connected to PostgreSQL")

	return &Client{pool: pool}, nil
}

// Close shuts down the PostgreSQL pool.
func (c *Client) Close() {
	if c == nil || c.pool == nil {
		return
	}

	slog.Info("closing PostgreSQL pool")
	c.pool.Close()
}

// Ping checks if PostgreSQL is reachable.
func (c *Client) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

func (c *Client) poolRef() *pgxpool.Pool {
	return c.pool
}

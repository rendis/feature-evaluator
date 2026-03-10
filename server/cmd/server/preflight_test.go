package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/rendis/feature-evaluator/internal/config"
)

func TestRunStartupPreflightRequiresDependencyEnv(t *testing.T) {
	t.Parallel()

	err := runStartupPreflight(context.Background(), &config.Config{})
	if err == nil {
		t.Fatal("expected missing env error")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunStartupPreflightSucceedsWhenDependenciesAreReachable(t *testing.T) {
	t.Parallel()

	pgListener := mustListen(t)
	defer pgListener.Close()
	redisListener := mustListen(t)
	defer redisListener.Close()

	cfg := &config.Config{
		Postgres: config.PostgresConfig{
			DatabaseURL:     fmt.Sprintf("postgres://postgres:postgres@%s/postgres?sslmode=disable", pgListener.Addr().String()),
			ConnectTimeout:  time.Second,
			MaxConnLifetime: time.Minute,
		},
		Redis: config.RedisConfig{
			URI: redisListener.Addr().String(),
		},
	}

	if err := runStartupPreflight(context.Background(), cfg); err != nil {
		t.Fatalf("preflight returned error: %v", err)
	}
}

func TestRunStartupPreflightReportsUnreachableRedis(t *testing.T) {
	t.Parallel()

	pgListener := mustListen(t)
	defer pgListener.Close()
	redisListener := mustListen(t)
	redisAddress := redisListener.Addr().String()
	redisListener.Close()

	cfg := &config.Config{
		Postgres: config.PostgresConfig{
			DatabaseURL:    fmt.Sprintf("postgres://postgres:postgres@%s/postgres?sslmode=disable", pgListener.Addr().String()),
			ConnectTimeout: time.Second,
		},
		Redis: config.RedisConfig{
			URI: redisAddress,
		},
	}

	err := runStartupPreflight(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected unreachable redis error")
	}
	if !strings.Contains(err.Error(), "cannot reach Redis") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func mustListen(t *testing.T) net.Listener {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	return listener
}

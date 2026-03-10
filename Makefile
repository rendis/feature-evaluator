SHELL := /bin/bash

PROJECT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
SERVER_DIR := $(PROJECT_DIR)/server
CONSOLE_DIR := $(PROJECT_DIR)/console
MCP_DIR := $(PROJECT_DIR)/apps/mcp

.PHONY: dev server console lint test quality format \
        lint-go lint-console test-go test-console typecheck \
        redis redis-stop \
        build-mcp mcp-login mcp-status mcp-logout

## Development

dev: ## Start backend + frontend (requires local PostgreSQL + Redis)
	@set -eu -o pipefail; \
	$(MAKE) --no-print-directory server & \
	server_pid=$$!; \
	$(MAKE) --no-print-directory console & \
	console_pid=$$!; \
	trap 'kill $$server_pid $$console_pid 2>/dev/null || true' EXIT INT TERM; \
	status=0; \
	while true; do \
		if ! kill -0 $$server_pid 2>/dev/null; then \
			wait $$server_pid || status=$$?; \
			break; \
		fi; \
		if ! kill -0 $$console_pid 2>/dev/null; then \
			wait $$console_pid || status=$$?; \
			break; \
		fi; \
		sleep 1; \
	done; \
	kill $$server_pid $$console_pid 2>/dev/null || true; \
	wait $$server_pid 2>/dev/null || true; \
	wait $$console_pid 2>/dev/null || true; \
	exit $$status

server: ## Run Go backend (port 8080, requires local PostgreSQL + Redis)
	@set -eu; \
	while IFS= read -r line || [ -n "$$line" ]; do \
		case "$$line" in \
			''|\#*) continue ;; \
		esac; \
		export "$$line"; \
	done < $(SERVER_DIR)/.env; \
	go run -C $(SERVER_DIR) ./cmd/server

console: ## Run React frontend (port 5173)
	pnpm -C $(CONSOLE_DIR) run dev

redis: ## Start local Redis via docker-compose
	docker compose -f $(PROJECT_DIR)/docker-compose.yml up -d redis

redis-stop: ## Stop Redis
	docker compose -f $(PROJECT_DIR)/docker-compose.yml down

## Quality

lint: lint-go lint-console ## Run all linters

lint-go: ## Run golangci-lint on server
	@cd $(SERVER_DIR) && golangci-lint run -c .golangci.yaml ./...

lint-console: ## Run ESLint on console
	pnpm -C $(CONSOLE_DIR) run lint

test: test-go test-console ## Run all tests

test-go: ## Run Go tests with race detection and coverage
	go test -C $(SERVER_DIR) -race -cover ./...

test-console: ## Run React tests
	pnpm -C $(CONSOLE_DIR) run test

typecheck: ## TypeScript type checking
	pnpm -C $(CONSOLE_DIR) run typecheck

quality: lint test typecheck ## Full quality gate (CI)

## Formatting

format: ## Format all code
	gofmt -w $(SERVER_DIR)
	goimports -w $(SERVER_DIR)
	pnpm -C $(CONSOLE_DIR) run format

## MCP

build-mcp: ## Build MCP server binary
	@mkdir -p "$(MCP_DIR)/bin"
	@pushd "$(MCP_DIR)" > /dev/null && go build -o "bin/feature-evaluator-mcp" ./cmd/server && popd > /dev/null

mcp-login: build-mcp ## Run MCP OIDC login
	$(MCP_DIR)/bin/feature-evaluator-mcp login

mcp-status: build-mcp ## Check MCP auth status
	$(MCP_DIR)/bin/feature-evaluator-mcp status

mcp-logout: build-mcp ## Clear MCP stored tokens
	$(MCP_DIR)/bin/feature-evaluator-mcp logout

## Help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := dev

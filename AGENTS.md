# AGENTS.md

This file provides guidance to AI agents when working with code in this repository.

## Overview

Feature Evaluator — a feature flag system with rule-based evaluation, segment targeting, and pack-based feature bundling. Monorepo: Go backend (`server/`) + React frontend (`console/`).

## Commands

```bash
make dev              # run everything (server + console + redis)
make server           # Go backend on :8080
make console          # React frontend on :5173
make redis            # Redis via docker-compose

make quality          # full CI gate: lint + test + typecheck
make lint             # lint-go + lint-console
make test             # test-go (with -race) + test-console
make typecheck        # pnpm -C console run typecheck

# individual
make lint-go          # golangci-lint (config: server/.golangci.yml)
make lint-console     # eslint
make test-go          # go test -race -count=1 ./...
make test-console     # vitest
make fmt              # gofmt + goimports + prettier

# Swagger
make swagger          # regenerate OpenAPI spec from handler annotations
```

Single Go test: `go test -C server -run TestName ./internal/domain/feature/...`
Single TS typecheck: `pnpm -C console run typecheck`

## Conventions

- **Never `cd`**: use `go run -C server`, `pnpm -C console`, `git -C /path`
- **Base API path**: `/features` (all routes under this prefix)
- **i18n**: Spanish default, English secondary. Namespace per route (`console/public/locales/{lang}/{namespace}.json`)
- **Config**: Viper reads env vars (underscore-separated). `AutomaticEnv()` does NOT load `.env` files — export them manually or use `make server`

## Architecture

### Backend (`server/`)

Go 1.25, Gin, PostgreSQL (`pgx/v5`), Redis, expr-lang, log/slog.

```
server/
├── cmd/server/          # entrypoint
├── internal/
│   ├── config/          # Viper config structs
│   ├── domain/          # business logic (ZERO external deps)
│   │   ├── evaluation/  # eval pipeline: feature→scheduling→environment→rules→external
│   │   ├── feature/     # feature + rules CRUD
│   │   ├── pack/        # feature bundles + activations (tenant/campus/program targets)
│   │   ├── segment/     # user segments + membership
│   │   ├── member/      # RBAC (owner/admin/editor/viewer)
│   │   ├── audit/       # eval error tracking
│   │   └── tag/         # feature tagging
│   ├── engine/          # expr-lang expression compiler, LRU cache (10K), security validation
│   ├── external/        # HTTP external validation with circuit breakers (gobreaker)
│   ├── handler/         # Gin handlers — only DTO transformation, no business logic
│   ├── dto/             # request/response types + mappers
│   ├── server/          # route registration + middleware stack
│   └── storage/postgres/ # repository implementations
└── pkg/apierror/        # structured API errors (code + messageKey for i18n)
```

**Eval pipeline** (`domain/evaluation/service.go`):

1. Fetch feature → check enabled → check scheduling (activeFrom/activeUntil) → check environment
2. Resolve pack activations (tenant/campus/program) → grant access if pack active
3. Pre-scan `inSegment()` calls → batch resolve segment memberships
4. Build expression env (context namespaces + `authenticated` + builtin functions)
5. Evaluate rules by priority → first match wins → return rule value
6. Optional external HTTP validation per rule (circuit breaker protected)
7. Fallback: return defaultValue

**Expression engine** (`engine/`): Uses `expr-lang/expr`. Expressions are written by admins in the console. Available in env: `user.*`, `tenant.*`, `campus.*`, `program.*`, `authenticated`, `inSegment(key)`, `now()`, `dateBefore()`, `dateAfter()`. Security: deny-list, max string length, max inSegment calls.

**Key gotchas**:

- `pkg/apierror`: always use `NewBadRequest(message, "error.messageKey")` pattern — messageKey is used for frontend i18n
- Redis fail-open: custom circuit breaker (3 errors → 30s cooldown → PING retry). Rate limiting also fail-open
- Gin `binding:"required"` on slices rejects empty `[]`, not just nil

### Frontend (`console/`)

React 19, Vite, TypeScript, TanStack Router (file-based), TanStack Query, shadcn/ui, Tailwind v4, react-hook-form + Zod, i18next, oidc-client-ts.

```
console/src/
├── api/           # typed API client (fetch wrapper) + per-resource modules
├── routes/        # TanStack file-based routing ($param for dynamic segments)
├── components/
│   ├── ui/        # shadcn/ui primitives
│   ├── shared/    # reusable (PageHeader, ConfirmDialog, PermissionButton, EmptyState)
│   ├── features/  # feature list, form, rule editor
│   ├── segments/  # segment list, import wizard
│   ├── packs/     # pack modals + features/activations tabs
│   └── dashboard/ # stats, charts, metrics
├── hooks/         # useUnsavedChanges, useDebounce, etc.
├── mutations/     # TanStack Query mutation hooks (invalidation logic)
├── queries/       # TanStack Query query factories
└── lib/           # permissions, auth, utils
```

**Patterns**:

- Packs and segments use modal dialogs (not separate routes)
- Features and rules use full-page forms with `useUnsavedChanges` hook for back navigation
- `PermissionButton` checks RBAC before rendering actions
- Query invalidation happens in mutation hooks, not in components

## API Routes

| Group  | Auth                | Routes                                                                                    |
| ------ | ------------------- | ----------------------------------------------------------------------------------------- |
| Health | none                | `GET /features/healthz`, `/features/readyz`                                               |
| Eval   | Bearer OR X-Api-Key | `POST /features/eval`, `/eval/bulk`, `/eval/active`                                       |
| Admin  | Bearer JWT          | `GET/POST/PUT/DELETE /features/admin/{features,packs,segments,tags,members,api-keys,...}` |

## MCP (OpenAPI Proxy)

Uses `mcp-openapi-proxy` — a Go binary that reads the Swagger spec and auto-generates MCP tools. Each API endpoint becomes a callable MCP tool (pattern: `fe_{method}_{path}`).

**Config**: `.mcp.json` at repo root
**Spec**: `server/docs/swagger.yaml` (generated by `make swagger`)
**Swagger UI**: Enable with `SERVER_SWAGGER_UI=true` → `GET /features/swagger/index.html`
**Skill**: `skills/feature-evaluator/SKILL.md`
**Setup docs**: `docs/mcp_setup.md`
**OIDC**: `mcp-openapi-proxy login` (when connecting to OIDC-enabled backend)

## Code Quality

- Go: golangci-lint with cyclop, funlen, gocognit, gosec, revive
- React: ESLint 9 flat config + Prettier
- Clean architecture: domain/ has zero external deps, handlers only map DTOs
- Errors: wrap with context in Go (`fmt.Errorf("doing X: %w", err)`), `pkg/apierror` for API responses
- Logging: `log/slog` (JSON in prod, text in dev)

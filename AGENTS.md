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

## Validation Policy (mandatory)

- **Always run all validations required by the affected surface before saying a change is done, asking for review, committing, or pushing.**
- **Do not rely on a single check** if the change can fail elsewhere (example: frontend route/search typing may require a full `build`, not only `typecheck`).
- If any validation fails, **fix it first**. Do not commit/push known-red code.
- **Before every commit/push, default to the strongest reasonable validation set**, not the weakest one that happens to pass.
- If there is any doubt about whether a check is needed, **run it**.
- In your final response, explicitly list:
  - which validations you ran
  - whether they passed/failed
  - any validation you intentionally did not run, and why

### Minimum validation matrix

- **Docs-only changes**: no code validation required.
- **Frontend code (`console/`)**:
  - Run `pnpm -C console run typecheck`
  - Run focused Vitest coverage for the changed behavior when tests exist or are added
- **Frontend routing / TanStack Router / search params / route contracts / Vite build-sensitive changes**:
  - Run `pnpm -C console run typecheck`
  - Run relevant focused tests
  - Run `pnpm -C console run build`
- **Backend Go changes (`server/`)**:
  - Run focused `go test -C server ...` for affected packages
  - If handlers, repositories, shared DTOs, or cross-package behavior changed, run `make test-go`
- **Cross-cutting or risky changes** (shared contracts, API shapes, repo-wide refactors, mixed frontend+backend work):
  - Prefer `make quality`
  - If `make quality` is too heavy for the iteration, explain why and run the nearest equivalent subset (`make lint`, `make test`, `make typecheck`, plus any focused checks)

### Pre-push default checklist

Use this checklist as the default behavior before commit/push:

- **Frontend-only change (`console/`)**
  - Run focused tests for the changed behavior
  - Run `pnpm -C console run typecheck`
  - Run `make lint-console` (or equivalent eslint check)
  - Run `pnpm -C console run build` for any UI/routing/type-sensitive change
- **Backend-only change (`server/`)**
  - Run focused `go test -C server ...`
  - Run `make lint-go`
  - Run `make test-go` if the change touches handlers, repositories, shared DTOs, or cross-package behavior
- **Mixed frontend + backend**
  - Run `make lint`
  - Run `make test`
  - Run `make typecheck`
  - Run `pnpm -C console run build`
- **High-risk / release-like / broad refactor**
  - Run `make quality`
  - If frontend is affected, also run `pnpm -C console run build`

### Validation reporting requirement

- Never say "validated" or "ready" without naming the exact commands that were executed.
- Never commit/push after only running focused tests if broader checks are warranted by the change.
- If a build emits warnings but succeeds, mention them explicitly in the final report.

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

# Implementation Plan — OFREP + 5 New Features

This document started as a forward-looking implementation plan. As of 2026-03-06 it is now the current implementation status for the same scope.

## Current Status

| Phase | Scope | Status | Notes |
| --- | --- | --- | --- |
| 0 | OFREP | Implemented | routes, mapping, reason translation, ETag/304 |
| 1 | Percentage Rollouts | Implemented | backend bucketing + console controls |
| 2 | Change History | Implemented | immutable changelog + console timeline |
| 3 | Multi-Workspace | Implemented with minor scope difference | `DELETE` archives, no hard "has data" block |
| 4 | Scheduled Rollouts | Implemented partially vs original scope | feature scheduling exists; rule rollout scheduling does not |
| 5 | A/B Testing | Implemented with minor scope difference | no hard max-4-variants guard |
| Docs | README + docs | Updated | aligned to code in this pass |
| Quality Gate | lint + tests + typecheck | Passed | verified on 2026-03-06 |

## Resolved Decisions

| # | Question | Current Decision |
| --- | --- | --- |
| 1 | Change history retention | No runtime config exposed today. History is implemented without a documented retention knob |
| 2 | A/B statistical threshold | Manual winner declaration stays the product decision |
| 3 | Multi-workspace limits | No configured limit |
| 4 | Scheduled rollout failure | One-shot execution. Failed schedules remain failed and are not retried automatically |
| 5 | Experiment vs rollout | Active experiment overrides rollout and rules |
| 6 | OpenFeature | OFREP implemented at `/ofrep/v1` |

## Verification Status

Verified in code and automated tests:

- [x] OFREP single and bulk evaluation
- [x] OFREP reason mapping (`TARGETING_MATCH`, `STATIC`, `SPLIT`, `DISABLED`)
- [x] OFREP bulk `ETag` and `If-None-Match` handling
- [x] rollout determinism and monotonicity
- [x] scheduled change execution through the worker
- [x] changelog entries for scheduler with `Actor: "system:scheduler"` and `ActorType: "system"`
- [x] workspace fallback to `default` when `X-Workspace` is missing
- [x] `make lint-go`
- [x] `make lint-console`
- [x] `make test`
- [x] `make typecheck`
- [x] `make quality`

Residual product gaps or scope differences:

- [ ] schedule rollout percentage for a rule
- [ ] explicit "cannot delete workspace with data" validation
- [ ] hard cap of 4 experiment variants if that product constraint is still required

## Phase 0: OFREP — OpenFeature Remote Evaluation Protocol

### Original intent

Expose a standards-based evaluation API so OpenFeature SDKs can talk to the evaluator without a custom provider.

### Current implementation

- Handler: `server/internal/handler/ofrep.go`
- DTOs: `server/internal/dto/ofrep.go`
- Routes:
  - `POST /ofrep/v1/evaluate/flags/:key`
  - `POST /ofrep/v1/evaluate/flags`
- Auth: same eval auth as native evaluation routes
- Bulk responses emit `ETag`; matching `If-None-Match` returns `304`

### Notes vs original plan

- The implemented file is `ofrep.go`, not `ofrep_handler.go`
- Context mapping accepts both flat OFREP payloads and already namespaced contexts

### Evidence

- `server/internal/handler/ofrep_test.go`
- `server/internal/server/server.go`

## Phase 1: Percentage Rollouts

### Original intent

Allow rule-level percentage rollouts that are deterministic, monotonic, and configurable in the console.

### Current implementation

- `RolloutPercentage` is stored on rules
- `RolloutSalt` is stored on features
- Evaluation uses FNV-1a over `featureKey + rolloutSalt + userID`
- Rollout is applied after expression match and before returning the rule result
- Console provides a rollout section and a badge in the rule list

### Notes vs original plan

- Scope matches the plan

### Evidence

- `server/internal/domain/evaluation/service.go`
- `server/internal/domain/evaluation/rollout_test.go`
- `console/src/components/rules/rule-form.tsx`
- `console/src/components/rules/rule-form-rollout.test.tsx`

## Phase 2: Change History

### Original intent

Record immutable admin change history with field-level diffs and console visibility.

### Current implementation

- Domain package: `server/internal/domain/changelog/`
- Storage: `server/internal/storage/postgres/changelog_repo.go`
- Handler: `server/internal/handler/changelog.go`
- Routes:
  - `GET /features/admin/changelog`
  - `GET /features/admin/changelog/:entityType/:entityKey`
- Console route: `/history`
- Feature detail page shows recent changes inline

### Notes vs original plan

- The implemented handler is `changelog.go`, not `changelog_handler.go`
- No documented retention environment variable exists today, so retention remains an operational decision rather than live product config

### Evidence

- `server/internal/domain/changelog/diff.go`
- `server/internal/handler/changelog.go`
- `console/src/routes/_authenticated/history/index.tsx`

## Phase 3: Multi-Workspace

### Original intent

Isolate data and RBAC per workspace, support repeated feature keys across workspaces, and keep backward compatibility through a `default` workspace.

### Current implementation

- Domain package: `server/internal/domain/workspace/`
- Middleware:
  - `WorkspaceResolver()`
  - `RequireActiveWorkspace()`
  - workspace-specific auth middleware for workspace CRUD
- Request header: `X-Workspace`
- Fallback workspace: `default`
- Existing data is migrated into `default`
- Console persists the active workspace and sends `X-Workspace` on every request

Routes:

- `GET /features/admin/workspaces`
- `GET /features/admin/workspaces/:key`
- `POST /features/admin/workspaces`
- `PUT /features/admin/workspaces/:key`
- `POST /features/admin/workspaces/:key/archive`
- `POST /features/admin/workspaces/:key/restore`
- `DELETE /features/admin/workspaces/:key` as archive alias

### Notes vs original plan

- The original "cannot delete workspace with data" rule is not implemented as such
- Current behavior is archival, not hard deletion
- This still satisfies backward compatibility and operational safety, but it is a scope difference

### Evidence

- `server/internal/server/middleware/workspace.go`
- `server/internal/server/middleware/workspace_test.go`
- `server/internal/storage/postgres/migrations.go`
- `console/src/stores/workspace-store.ts`
- `console/src/components/layout/workspace-selector.tsx`

## Phase 4: Scheduled Rollouts

### Original intent

Schedule future configuration changes and execute them once across multiple instances.

### Current implementation

- Domain package: `server/internal/domain/schedule/`
- Worker interval: 30 seconds
- Routes:
  - `POST /features/admin/features/:key/schedules`
  - `GET /features/admin/features/:key/schedules`
  - `DELETE /features/admin/schedules/:id`
- Supported `changeType` values today:
  - `toggle`
  - `update`
  - `default_value`
  - `environment`
- Executed schedules write changelog entries as `system:scheduler`

### Notes vs original plan

- There is no rule-level rollout percentage scheduling yet
- Collection name is `schedules`, not `scheduled_changes`
- The handler is `schedule.go`, not `schedule_handler.go`

### Evidence

- `server/internal/domain/schedule/worker.go`
- `server/internal/domain/schedule/worker_test.go`
- `server/internal/handler/schedule.go`

## Phase 5: A/B Testing

### Original intent

Run experiments with deterministic assignment, automatic exposures, client-reported conversions, and result reporting.

### Current implementation

- Domain package: `server/internal/domain/experiment/`
- Routes:
  - `GET /features/admin/experiments`
  - `GET /features/admin/experiments/:id`
  - `GET /features/admin/experiments/:id/results`
  - `POST /features/admin/experiments`
  - `PUT /features/admin/experiments/:id`
  - `POST /features/admin/experiments/:id/start`
  - `POST /features/admin/experiments/:id/pause`
  - `POST /features/admin/experiments/:id/complete`
  - `POST /features/admin/experiments/:id/declare-winner`
  - `POST /features/eval/conversions`
- Active experiments override normal rule and rollout evaluation
- Exposures are recorded automatically
- Results expose conversion rate and confidence intervals

### Notes vs original plan

- The implemented conversion route is `/features/eval/conversions`, not `/eval/conversions`
- Collections are `experiments`, `experiment_exposures`, and `experiment_conversions`
- The handler is `experiment.go`, not `experiment_handler.go`
- The original wording said "2 to 4 variants"; the current implementation enforces a minimum of 2 and weight sum 100, but not a hard maximum of 4

### Evidence

- `server/internal/domain/experiment/service.go`
- `server/internal/domain/experiment/service_test.go`
- `server/internal/domain/experiment/stats_test.go`
- `console/src/routes/_authenticated/experiments/`

## Documentation and Quality Gate

This update aligned the documentation with the implemented system:

- `docs/architecture.md`
- `docs/api-reference.md`
- `docs/configuration.md`
- `docs/requirements/roadmap-features.md`
- `README.md`
- `docs/UI-FLOW.md`
- `docs/flows/workspaces-flow.md`
- `docs/flows/schedules-flow.md`
- `docs/deployment.md`

Verified commands on 2026-03-06:

```bash
make lint-go
make lint-console
make test
make typecheck
make quality
```

All commands completed successfully in this update.

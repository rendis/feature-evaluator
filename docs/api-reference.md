# API Reference

Current public HTTP surface for the Feature Evaluator.

## Base Paths

| Base Path | Purpose |
| --- | --- |
| `/features` | native evaluation and admin API |
| `/ofrep/v1` | OpenFeature Remote Evaluation Protocol |

## Authentication

| Method | Header | Supported by |
| --- | --- | --- |
| Bearer JWT | `Authorization: Bearer <token>` | eval, OFREP, admin |
| API key | `X-Api-Key: ...` | eval, OFREP, admin |
| API key (OFREP alias) | `X-API-Key: ...` | OFREP-compatible clients |

Notes:

- eval and OFREP routes accept Bearer JWT or API key
- admin routes accept Bearer JWT or admin API key
- workspace routes use dedicated auth middleware because they must work before a workspace is selected

## Common Headers

| Header | Required | Meaning |
| --- | --- | --- |
| `X-Workspace` | No | workspace scope. Falls back to `default` when omitted |
| `X-Tenant-ID` | No | tenant context fallback |
| `X-Campus-ID` | No | campus context fallback |
| `X-Program-ID` | No | program context fallback |
| `X-Request-ID` | No | correlation ID. The server generates one if absent |

## Health

| Method | Route | Description |
| --- | --- | --- |
| `GET` | `/features/healthz` | liveness probe |
| `GET` | `/features/readyz` | readiness probe for PostgreSQL and Redis |

## Native Evaluation API

### POST /features/eval

Evaluates a single feature in the current workspace.

Example request:

```json
{
  "featureKey": "checkout-v2",
  "context": {
    "user": { "id": "u-123", "plan": "enterprise" },
    "tenant": { "id": "cl" }
  },
  "environment": "production"
}
```

Example response:

```json
{
  "featureKey": "checkout-v2",
  "value": true,
  "valueType": "boolean",
  "environment": "production",
  "matchedRule": { "id": "r-1", "name": "Enterprise users" },
  "reason": "matched_rule",
  "metadata": {},
  "evaluatedAt": "2026-03-06T12:00:00Z"
}
```

### POST /features/eval/bulk

Evaluates multiple features in one request.

Example request:

```json
{
  "features": [
    {
      "featureKey": "checkout-v2",
      "context": { "user": { "id": "u-123" } },
      "environment": "production"
    },
    {
      "featureKey": "new-dashboard",
      "context": { "user": { "id": "u-123" } },
      "environment": "production"
    }
  ]
}
```

### POST /features/eval/active

Evaluates all active features for the supplied context and environment.

Example request:

```json
{
  "context": { "user": { "id": "u-123" } },
  "environment": "production",
  "tags": ["mobile"]
}
```

### POST /features/eval/conversions

Records a conversion event for a running experiment.

Request body:

```json
{
  "experimentId": "exp_123",
  "userId": "u-123",
  "metricKey": "signup",
  "value": 1
}
```

Success response: `201 Created`

## OFREP

OFREP routes are outside `/features` and use `/ofrep/v1`.

### POST /ofrep/v1/evaluate/flags/:key

Single flag evaluation.

Request body:

```json
{
  "context": {
    "targetingKey": "u-123",
    "tenantId": "cl",
    "plan": "enterprise"
  }
}
```

Success response:

```json
{
  "key": "checkout-v2",
  "value": true,
  "variant": "enterprise-users",
  "reason": "TARGETING_MATCH",
  "metadata": {}
}
```

### POST /ofrep/v1/evaluate/flags

Bulk evaluation for all visible flags in the current workspace.

Request body:

```json
{
  "context": {
    "targetingKey": "u-123",
    "tenantId": "cl"
  }
}
```

Response shape:

```json
{
  "flags": [
    {
      "key": "checkout-v2",
      "value": true,
      "variant": "enterprise-users",
      "reason": "TARGETING_MATCH",
      "metadata": {}
    }
  ]
}
```

Caching behavior:

- responses include `ETag`
- send `If-None-Match` with the previous value
- server returns `304 Not Modified` when the payload hash is unchanged

### OFREP Context Mapping

| OFREP context | Internal evaluation context |
| --- | --- |
| `targetingKey` | `user.id` |
| `tenantId` | `tenant.id` |
| `campusId` | `campus.id` |
| `programId` | `program.id` |
| any other flat field | `user.<field>` |

Nested namespaced contexts are also accepted.

### OFREP Reason Mapping

| OFREP reason | Meaning in this server |
| --- | --- |
| `TARGETING_MATCH` | matched rule |
| `STATIC` | default value |
| `SPLIT` | experiment variant |
| `DISABLED` | feature disabled, inactive, expired, or environment-mismatched |
| `UNKNOWN` | any other non-mapped result |

### OFREP Error Codes

| HTTP | Error code | Meaning |
| --- | --- | --- |
| `400` | `PARSE_ERROR` | invalid JSON or malformed body |
| `400` | `TARGETING_KEY_MISSING` | `context.targetingKey` is required |
| `404` | `FLAG_NOT_FOUND` | requested flag does not exist |
| `500` | `GENERAL` | unexpected evaluation failure |

## Admin API

All routes below are relative to `/features/admin`.

### Features

| Method | Route | Permission |
| --- | --- | --- |
| `GET` | `/features` | `features.read` |
| `POST` | `/features` | `features.write` |
| `GET` | `/features/:key` | `features.read` |
| `PUT` | `/features/:key` | `features.write` |
| `DELETE` | `/features/:key` | `features.write` |
| `PATCH` | `/features/:key/toggle` | `features.write` |

### Rules

Rules are embedded under features.

| Method | Route | Permission |
| --- | --- | --- |
| `GET` | `/features/:key/rules` | `features.read` |
| `POST` | `/features/:key/rules` | `features.write` |
| `PUT` | `/features/:key/rules/:ruleId` | `features.write` |
| `DELETE` | `/features/:key/rules/:ruleId` | `features.write` |
| `PUT` | `/features/:key/rules/reorder` | `features.write` |

Rule payloads support `rolloutPercentage` and the existing expression/scope/external-validation fields.

### Schedules

| Method | Route | Permission |
| --- | --- | --- |
| `GET` | `/features/:key/schedules` | `features.read` |
| `POST` | `/features/:key/schedules` | `features.write` |
| `DELETE` | `/schedules/:id` | `features.write` |

Schedule request body:

```json
{
  "changeType": "toggle",
  "payload": { "enabled": true },
  "scheduledAt": "2026-03-10T14:00:00Z"
}
```

Supported `changeType` values in the backend:

- `toggle`
- `update`
- `default_value`
- `environment`

### Experiments

| Method | Route | Permission |
| --- | --- | --- |
| `GET` | `/experiments` | `experiments.read` |
| `GET` | `/experiments/:id` | `experiments.read` |
| `GET` | `/experiments/:id/results` | `experiments.read` |
| `POST` | `/experiments` | `experiments.write` |
| `PUT` | `/experiments/:id` | `experiments.write` |
| `POST` | `/experiments/:id/start` | `experiments.write` |
| `POST` | `/experiments/:id/pause` | `experiments.write` |
| `POST` | `/experiments/:id/complete` | `experiments.write` |
| `POST` | `/experiments/:id/declare-winner` | `experiments.write` |

Create/update request shape:

```json
{
  "featureKey": "checkout-v2",
  "name": "Checkout CTA test",
  "description": "Control vs variant",
  "variants": [
    { "key": "control", "value": false, "weight": 50 },
    { "key": "variant-a", "value": true, "weight": 50 }
  ],
  "metrics": [
    { "key": "signup", "name": "Signup" }
  ]
}
```

### Changelog

| Method | Route | Permission |
| --- | --- | --- |
| `GET` | `/changelog` | `audit.read` |
| `GET` | `/changelog/:entityType/:entityKey` | `audit.read` |

Supported query parameters on list endpoints:

- `entityType`
- `entityKey`
- `actor`
- `action`
- `from`
- `to`
- `page`
- `pageSize`

### Workspaces

Workspace routes are still under `/features/admin/workspaces`, but they use dedicated auth middleware instead of the normal workspace-scoped admin chain.

| Method | Route | Notes |
| --- | --- | --- |
| `GET` | `/workspaces` | supports `includeArchived=true` |
| `GET` | `/workspaces/:key` | get one workspace |
| `POST` | `/workspaces` | owner-only workspace creation |
| `PUT` | `/workspaces/:key` | owner-only update |
| `POST` | `/workspaces/:key/archive` | archive workspace |
| `POST` | `/workspaces/:key/restore` | restore archived workspace |
| `DELETE` | `/workspaces/:key` | backward-compatible alias for archive |

Important behavior:

- if the first workspace is being created, the current user is bootstrapped as owner
- the middleware does not require `X-Workspace` when listing or creating workspaces
- deleting a workspace does not hard-delete data; it archives the workspace

### Segments

| Method | Route | Permission |
| --- | --- | --- |
| `GET` | `/segments` | `segments.read` |
| `POST` | `/segments` | `segments.write` |
| `GET` | `/segments/:key` | `segments.read` |
| `PUT` | `/segments/:key` | `segments.write` |
| `DELETE` | `/segments/:key` | `segments.write` |
| `GET` | `/segments/:key/members` | `segments.read` |
| `POST` | `/segments/:key/members/import` | `segments.write` |
| `DELETE` | `/segments/:key/members/bulk` | `segments.write` |

### Packs

| Method | Route | Permission |
| --- | --- | --- |
| `GET` | `/packs` | `packs.read` |
| `GET` | `/packs/:key` | `packs.read` |
| `GET` | `/packs/:key/activations` | `packs.read` |
| `GET` | `/packs/by-target` | `packs.read` |
| `POST` | `/packs` | `packs.write` |
| `PUT` | `/packs/:key` | `packs.write` |
| `DELETE` | `/packs/:key` | `packs.write` |
| `PATCH` | `/packs/:key/toggle` | `packs.write` |
| `POST` | `/packs/:key/activate` | `packs.write` |
| `DELETE` | `/packs/:key/activate` | `packs.write` |

### Tags

| Method | Route | Permission |
| --- | --- | --- |
| `GET` | `/tags` | `features.read` |
| `POST` | `/tags` | `features.write` |
| `PUT` | `/tags/:key` | `features.write` |
| `DELETE` | `/tags/:key` | `features.write` |

### Members

| Method | Route | Permission |
| --- | --- | --- |
| `GET` | `/members/me` | authenticated member |
| `GET` | `/members` | `members.read` |
| `POST` | `/members` | `members.manage` |
| `PUT` | `/members/:id/role` | `members.manage` |
| `DELETE` | `/members/:id` | `members.manage` |
| `POST` | `/members/:id/transfer-ownership` | `ownership.transfer` |

### API Keys

| Method | Route | Permission |
| --- | --- | --- |
| `POST` | `/api-keys` | `members.manage` |
| `GET` | `/api-keys` | `members.manage` |
| `PUT` | `/api-keys/:id/rotate` | `members.manage` |
| `DELETE` | `/api-keys/:id` | `members.manage` |

### Expression Tools

| Method | Route | Permission |
| --- | --- | --- |
| `POST` | `/expression/validate` | `features.read` |
| `POST` | `/expression/test` | `features.read` |
| `GET` | `/expression/schema` | `features.read` |

### Dashboard and Audit

| Method | Route | Permission |
| --- | --- | --- |
| `GET` | `/dashboard/stats` | `features.read` |
| `GET` | `/dashboard/activity` | `features.read` |
| `GET` | `/dashboard/error-summary` | `features.read` |
| `GET` | `/dashboard/operations` | `features.read` |
| `GET` | `/dashboard/metrics/overview` | `audit.read` |
| `GET` | `/dashboard/metrics/features` | `audit.read` |
| `GET` | `/dashboard/metrics/reasons` | `audit.read` |
| `GET` | `/dashboard/metrics/tenants` | `audit.read` |
| `GET` | `/dashboard/metrics/environments` | `audit.read` |
| `GET` | `/dashboard/metrics/cache` | `audit.read` |
| `GET` | `/dashboard/metrics/external` | `audit.read` |
| `GET` | `/audit/errors` | `audit.read` |

## Role Matrix

| Permission | Owner | Admin | Editor | Viewer |
| --- | :---: | :---: | :---: | :---: |
| `features.read` | Y | Y | Y | Y |
| `features.write` | Y | Y | Y | - |
| `segments.read` | Y | Y | Y | Y |
| `segments.write` | Y | Y | Y | - |
| `packs.read` | Y | Y | Y | Y |
| `packs.write` | Y | Y | Y | - |
| `experiments.read` | Y | Y | Y | Y |
| `experiments.write` | Y | Y | Y | - |
| `members.read` | Y | Y | Y | Y |
| `members.manage` | Y | Y | - | - |
| `settings.manage` | Y | Y | - | - |
| `audit.read` | Y | Y | Y | Y |
| `workspace.delete` | Y | - | - | - |
| `ownership.transfer` | Y | - | - | - |

---
name: feature-evaluator
description: >-
  Manage the Feature Evaluator API — feature flags, targeting rules, packs,
  segments, experiments, and more. Each API endpoint is an MCP tool that
  executes real API calls via mcp-openapi-proxy.
allowed-tools:
  - mcp__feature-evaluator__*
---

# Feature Evaluator MCP

MCP integration for the Feature Evaluator — a feature flag system with rule-based evaluation, segment targeting, and pack-based feature bundling.

## MCP Proxy

This project uses [mcp-openapi-proxy](https://github.com/rendis/mcp-openapi-proxy) to auto-generate MCP tools from the OpenAPI spec. Each API endpoint becomes a callable tool that executes real HTTP requests.

**Repository**: https://github.com/rendis/mcp-openapi-proxy
**Install**: `go install github.com/rendis/mcp-openapi-proxy/cmd/mcp-openapi-proxy@latest`
**Docs**: See [mcp-openapi-proxy README](https://github.com/rendis/mcp-openapi-proxy#readme) for full configuration, authentication, and troubleshooting.

## Setup

### Claude Code

The project includes a `.mcp.json` — Claude Code auto-detects it. No manual setup needed.

```bash
make server       # Start API on port 8080
# MCP auto-configured via .mcp.json → mcp-openapi-proxy reads the Swagger spec
```

Verify: `claude mcp list` → should show `feature-evaluator` connected.

### OpenAI Codex

Edit `~/.codex/config.toml`:

```toml
[mcp_servers.feature-evaluator]
command = "mcp-openapi-proxy"
args = []

[mcp_servers.feature-evaluator.env]
MCP_SPEC = "https://raw.githubusercontent.com/rendis/feature-evaluator/main/server/docs/swagger.yaml"
MCP_BASE_URL = "<your-api-url>/features"
MCP_TOOL_PREFIX = "fe"
```

### Gemini CLI

Edit `~/.gemini/settings.json` (global) or `.gemini/settings.json` (project):

```json
{
  "mcpServers": {
    "feature-evaluator": {
      "command": "mcp-openapi-proxy",
      "args": [],
      "env": {
        "MCP_SPEC": "https://raw.githubusercontent.com/rendis/feature-evaluator/main/server/docs/swagger.yaml",
        "MCP_BASE_URL": "<your-api-url>/features",
        "MCP_TOOL_PREFIX": "fe"
      }
    }
  }
}
```

### Regenerate Swagger (after handler changes)

```bash
make swagger      # generates server/docs/swagger.yaml from handler annotations
```

### OIDC Authentication (production)

```bash
mcp-openapi-proxy login     # browser-based OIDC PKCE login
mcp-openapi-proxy status    # check auth status
mcp-openapi-proxy logout    # clear stored tokens
```

## Available MCP Tools

Each API endpoint becomes a callable MCP tool. Tool naming pattern: `fe_{operation}`.

### Key Tools

| Tool | Purpose |
|------|---------|
| `fe_list_features` | List all features |
| `fe_create_feature` | Create a feature |
| `fe_get_feature` | Get feature by key |
| `fe_update_feature` | Update a feature |
| `fe_delete_feature` | Delete a feature |
| `fe_toggle_feature` | Enable/disable a feature |
| `fe_list_rules` | List rules for a feature |
| `fe_create_rule` | Create a rule |
| `fe_evaluate` | Evaluate a single feature |
| `fe_bulk_evaluate` | Bulk evaluate features |
| `fe_evaluate_all` | Evaluate all active features |
| `fe_list_packs` | List packs |
| `fe_list_segments` | List segments |
| `fe_list_experiments` | List experiments |
| `fe_validate_expression` | Validate expression syntax |
| `fe_test_expression` | Test expression with context |
| `fe_expression_schema` | Available expression fields/functions |
| `fe_dashboard_stats` | Dashboard statistics |

All tools follow the `fe_*` prefix convention. Run any `fe_list_*` tool to discover available resources.

## Quick Start Workflow

```
1. fe_list_features                → discover existing features
2. fe_get_feature { key: "..." }   → inspect a specific feature
3. fe_list_rules { key: "..." }    → see targeting rules
4. fe_evaluate { featureKey: "...", context: {...} }  → evaluate a feature
5. fe_expression_schema            → see available expression fields/functions
6. fe_validate_expression { expression: "..." }       → validate rule syntax
```

## API Groups

All routes under base path `/features`.

### Health (no auth)
- `GET /healthz`, `GET /readyz`

### Evaluation (Bearer OR X-Api-Key + rate limited)
- `POST /eval` — evaluate single feature
- `POST /eval/bulk` — evaluate multiple features
- `POST /eval/active` — evaluate all active features
- `POST /eval/conversions` — record experiment conversion

### OFREP (OpenFeature Remote Evaluation Protocol)
- `POST /ofrep/v1/evaluate/flags/:key` — single flag (separate base path)
- `POST /ofrep/v1/evaluate/flags` — bulk flags

### Features (Bearer JWT)
- CRUD: `GET/POST/PUT/DELETE /admin/features`, `/admin/features/:key`
- `PATCH /admin/features/:key/toggle` — enable/disable
- `GET /admin/environments` — list environments

### Rules (Bearer JWT)
- CRUD: `GET/POST/PUT/DELETE /admin/features/:key/rules`
- `PUT /admin/features/:key/rules/reorder`
- `POST/GET /admin/features/:key/expression/test`, `/expression-schema`

### Expression Tools (Bearer JWT)
- `POST /admin/expression/validate` — validate syntax
- `POST /admin/expression/test` — test with context
- `GET /admin/expression/schema` — available fields/functions

### Schedules (Bearer JWT)
- `POST/GET /admin/features/:key/schedules`
- `DELETE /admin/schedules/:id`

### Packs (Bearer JWT)
- CRUD: `GET/POST/PUT/DELETE /admin/packs`, `/admin/packs/:key`
- `PATCH /admin/packs/:key/toggle`
- `POST/DELETE /admin/packs/:key/activate` — activate/deactivate on target
- `GET /admin/packs/:key/activations`, `/admin/packs/by-target`

### Segments (Bearer JWT)
- CRUD: `GET/POST/PUT/DELETE /admin/segments`, `/admin/segments/:key`
- `GET /admin/segments/:key/schema`, `/admin/segments/:key/records`
- `POST /admin/segments/:key/data/import`

### Experiments (Bearer JWT)
- CRUD: `GET/POST/PUT /admin/experiments`, `/admin/experiments/:id`
- Lifecycle: `POST /admin/experiments/:id/{start,pause,complete,declare-winner}`
- `GET /admin/experiments/:id/results`

### Tags, Members, API Keys, Auth Profiles, External APIs
- CRUD at `/admin/tags`, `/admin/members`, `/admin/api-keys`, `/admin/auth-profiles`, `/admin/external-apis`

### Dashboard & Metrics (Bearer JWT)
- `GET /admin/dashboard/{stats,activity,error-summary,operations}`
- `GET /admin/dashboard/metrics/{overview,features,reasons,tenants,environments,cache,external}`

### Audit & Changelog (Bearer JWT)
- `GET /admin/audit/errors`
- `GET /admin/changelog`, `/admin/changelog/:entityType/:entityKey`

### Workspaces (Bearer JWT)
- CRUD: `GET/POST/PUT/DELETE /admin/workspaces`, `/admin/workspaces/:key`
- `POST /admin/workspaces/:key/{archive,restore}`

### Tiers (Bearer JWT, read-only)
- `GET /admin/tiers`

## Expression Language (expr-lang)

Rules use `expr-lang/expr`. Full reference: [references/expressions.md](references/expressions.md)

### Eval Request

```json
{
  "featureKey": "my-feature",
  "context": {
    "user": { "id": "u-1", "email": "alice@empresa.com", "role": "admin" },
    "tenant": { "id": "t-1" },
    "campus": { "id": "c-1" },
    "program": { "id": "p-1" },
    "custom": { "country": "US" },
    "billing": { "plan": "pro" }
  },
  "environment": "production"
}
```

### Evaluation Inputs

- `environment` can be sent in the eval request body.
- `X-Environment` overrides body `environment` on eval endpoints.
- `environment` is used as a feature-level eligibility gate before rule evaluation. It is not documented as an expression variable.
- Each `context` namespace becomes a top-level variable in expressions (`user.*`, `tenant.*`, `custom.*`, `billing.*`, etc.).
- `custom` is valid and already supported.
- Any additional namespace under `context` is also exposed as a top-level variable.

See [references/expressions.md](references/expressions.md) for the full precedence and evaluation behavior.

### System Headers vs Expression Headers

System headers affect request handling, auth, workspace resolution, or context fallbacks. They are not exposed to `headers.*` automatically just because they were sent.

| Header | Backend role | Context fallback | Derived/auth input | Workspace/input scope | Auto in expressions? |
|--------|--------------|------------------|--------------------|-----------------------|----------------------|
| `Authorization` | Bearer auth for eval/admin | No | Yes | No | No |
| `X-Api-Key` | API key auth for eval/admin | No | Yes | No | No |
| `X-Environment` | Overrides body `environment` | No | No | No | No |
| `X-Tenant-ID` | Tenant extraction middleware | `context.tenant.id` only if `context.tenant` is absent | No | No | No |
| `X-Campus-ID` | Campus extraction middleware | `context.campus.id` only if `context.campus` is absent | No | No | No |
| `X-Program-ID` | Program extraction middleware | `context.program.id` only if `context.program` is absent | No | No | No |
| `X-Workspace` | Workspace resolution | No | No | Yes | No |
| `X-Request-ID` | Request correlation | No | No | No | No |

`headers.*` only contains headers explicitly declared in the feature `InputContract`. If a feature maps a header like `X-Region` to expression key `region`, the rule can read `headers.region`.

### Expression Variables

| Variable | Source |
|----------|--------|
| `user.*`, `tenant.*`, `campus.*`, `program.*`, `custom.*` | `context` in request body |
| any additional `context.<namespace>.*` | additional custom namespaces in request body |
| `headers.*` | HTTP headers (via feature InputContract) |
| `requestBody.*` | Request body fields |
| `authenticated` | True if auth profile validation passed |
| `derived.email` | JWT `email` claim, fallback `context.user.email` |
| `derived.userId` | JWT `sub` claim, fallback `context.user.id` |
| `derived.subject` | JWT `sub` claim (no fallback) |
| `derived.name` | JWT `name` claim (no fallback) |
| `derived.bearerTokenPresent` | True if Bearer token in request |
| `derived.apiKeyPresent` | True if API key in request |

**JWT auto-extraction:** Send `Authorization: Bearer <jwt>` → claims `sub`, `email`, `name` auto-populate `derived.*`. JWT claims take priority over context body.

### Builtin Functions

| Function | Description |
|----------|-------------|
| `now()` | Current UTC time |
| `dateBefore(date, ref)` | True if date < ref (RFC3339, `YYYY-MM-DD`) |
| `dateAfter(date, ref)` | True if date > ref |
| `contains(val, needle)` | String containment |
| `startsWith(val, prefix)` | String prefix |
| `endsWith(val, suffix)` | String suffix |
| `inSegment(key)` | Segment membership check |
| `externalApi(key)` | External API validation |

### Common Patterns

```
# Email targeting (from JWT)
derived.email == "admin@empresa.com"
endsWith(derived.email, "@empresa.com")

# Email targeting (from context body)
user.email == "admin@empresa.com"

# Custom namespaces from context
custom.country == "CL"
billing.plan == "pro"

# Header declared in feature InputContract
headers.region == "us-east-1"

# Auth + role
authenticated && user.role == "admin"
derived.apiKeyPresent

# Segments
inSegment("beta-users")

# Date range
dateAfter(now(), "2025-06-01") && dateBefore(now(), "2025-12-31")

# Combined
tenant.id == "acme" && inSegment("early-access") && authenticated
```

### Security Limits

Max AST depth: 10, max nodes: 100, max `inSegment()` calls: 5, max string length: 1000.
Denied: exec, system, import, require, __proto__, constructor, prototype, eval, Function, process.

## Known API Behaviors

- List responses: `{"data": [...], "pagination": {"page": 1, "pageSize": 20, "total": N, "totalPages": M}}`
- Toggle/Delete: return 204 No Content
- Errors: `{"error": {"code": "...", "message": "...", "messageKey": "..."}}`
- Pack targets: `tenant`, `campus`, `program`
- Value types: `boolean`, `string`, `number`, `json`
- Access policies: `public` (default), `optional`, `required`

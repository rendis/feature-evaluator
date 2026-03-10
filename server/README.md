# Feature Evaluator -- Backend

Go 1.25 REST API implementing feature flag evaluation with expression-based rules, segment membership, feature packs, and external endpoint validation.

## Architecture

Clean architecture with strict dependency direction: handlers depend on services, services depend on repository interfaces, storage implements those interfaces.

```
cmd/server/main.go              # Entry point, DI wiring, graceful shutdown
internal/
├── config/                     # Viper-based env configuration
├── server/
│   ├── server.go               # Gin router setup, route registration
│   └── middleware/              # Request pipeline (see below)
├── domain/                     # Business logic -- zero external dependencies
│   ├── feature/                # Feature model, repository interface, service
│   ├── segment/                # Segment + member models, repository, service
│   ├── member/                 # Team member model, RBAC permissions
│   ├── evaluation/             # Evaluation orchestration service
│   ├── pack/                   # Feature packs + activations
│   ├── apikey/                 # API key management (eval + admin types)
│   ├── tag/                    # Feature tags with colors
│   ├── metrics/                # Runtime metrics collector
│   └── audit/                  # Eval error model, repository, service
├── engine/                     # Rule engine (expr-lang)
│   ├── engine.go               # Compile + evaluate expressions
│   ├── environment.go          # Build expr env (user, tenant, functions)
│   ├── functions.go            # inSegment(), now(), dateBefore(), dateAfter()
│   ├── security.go             # Deny-list, complexity limits
│   └── compiler_cache.go       # LRU cache (max 10K compiled programs)
├── external/                   # External HTTP validation
│   ├── caller.go               # HTTP client with timeout
│   ├── circuit_breaker.go      # Per-endpoint circuit breaker (gobreaker)
│   ├── request_builder.go      # Declarative request mapping
│   └── response_evaluator.go   # Evaluate external response via expr
├── storage/
│   ├── postgres/               # All PostgreSQL repository implementations
│   └── redis/                  # Cache client, key management
├── handler/                    # HTTP handlers (Gin)
│   ├── eval.go                 # POST /eval, /eval/bulk, /eval/active
│   ├── feature.go              # Feature CRUD
│   ├── rule.go                 # Rule CRUD + reorder + expression tools
│   ├── segment.go              # Segment CRUD + member import/delete
│   ├── pack.go                 # Pack CRUD + activations
│   ├── apikey.go               # API key CRUD + rotate
│   ├── tag.go                  # Tag CRUD
│   ├── member.go               # Team member management
│   ├── dashboard.go            # Dashboard stats + activity
│   ├── metrics.go              # Metrics endpoints
│   ├── audit.go                # Error log queries
│   └── health.go               # /healthz, /readyz
├── dto/                        # Request/response types + mappers
└── pkg/apierror/               # Structured error types + codes
```

## Rule Engine

Expressions use [expr-lang](https://github.com/expr-lang/expr) syntax. Compiled to bytecode and cached in an LRU (max 10K entries, keyed by SHA-256 of expression text).

See [Rule Engine Guide](../docs/rule-engine.md) for full documentation.

### Quick Reference

| Field           | Type             | Description                   |
| --------------- | ---------------- | ----------------------------- |
| `user`          | `map[string]any` | User attributes from request  |
| `tenant`        | `string`         | From `X-Tenant-ID` header     |
| `campus`        | `string`         | From `X-Campus-ID` header     |
| `program`       | `string`         | From `X-Program-ID` header    |
| `authenticated` | `bool`           | Whether request has valid JWT |

### Custom Functions

| Function     | Signature                           | Description                                                     |
| ------------ | ----------------------------------- | --------------------------------------------------------------- |
| `inSegment`  | `inSegment(key string) bool`        | Check segment membership (tenant-scoped first, global fallback) |
| `now`        | `now() time.Time`                   | Current time                                                    |
| `dateBefore` | `dateBefore(date, ref string) bool` | Date comparison                                                 |
| `dateAfter`  | `dateAfter(date, ref string) bool`  | Date comparison                                                 |

### Expression Examples

```
user.plan == "enterprise" && user.companySize > 100
tenant in ["cl", "co"] || user.email endsWith "@example.com"
inSegment("beta-testers") && user.active == true
dateBefore(user.trialEnd, now())
```

### Security Limits

- **Deny-list**: Blocks `exec`, `system`, `import`, `require`, `__proto__`, `constructor`
- **AST depth**: Max 10
- **AST nodes**: Max 100
- **inSegment() calls**: Max 5 per expression
- **String literals**: Max 1000 chars

## Middleware Stack

Applied in order for every request:

| Middleware     | Description                                               |
| ------------- | --------------------------------------------------------- |
| `recovery`     | Panic recovery, logs stack trace                          |
| `requestid`    | Generates UUID, sets `X-Request-ID` header                |
| `bodysize`     | Limits request body to 1MB                                |
| `logging`      | Structured request/response logging via `log/slog`        |
| `cors`         | Origin validation, credentials support                    |
| `tenant`       | Extracts `X-Tenant-ID`, `X-Campus-ID`, `X-Program-ID`    |
| `auth`         | JWT validation (OIDC JWKS, cached 1hr) or dev-mode bypass |
| `authz`        | Role-based permission check                               |
| `ratelimit`    | GCRA algorithm via `go-redis/redis_rate`, fail-open with 30s local cooldown |

## Configuration

All configuration via environment variables. See [Configuration Guide](../docs/configuration.md) for full reference.

Key settings for local development:

```bash
AUTH_DISABLED=true
DATABASE_URL=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable
REDIS_URI=localhost:6379
AUTH_SECRETS_MASTER_KEY=...        # Required: base64-encoded 32-byte key
LOG_LEVEL=debug
LOG_FORMAT=text
```

Startup now performs an explicit dependency preflight:

- `DATABASE_URL` must be set and reachable over TCP
- `REDIS_URI` must be set and reachable over TCP
- missing values no longer fall back to silent defaults

### `AUTH_SECRETS_MASTER_KEY`

This variable is required at startup. The backend fails fast if it is missing or does not decode to 32 bytes.

It is used to encrypt:

- `AuthProfile` secret payloads stored in PostgreSQL
- inline API keys stored for rule-level external validation

Generate a valid key with:

```bash
openssl rand -base64 32
```

Then add it to `server/.env`:

```bash
AUTH_SECRETS_MASTER_KEY=your-generated-base64-key
```

`make server` and `make dev` read `server/.env` directly, so the variable must exist there before starting the backend.

## Inbound Auth and External Validation

The evaluation flow is split in two layers:

1. `AuthProfile`
   Reusable inbound authentication for `/eval` and OFREP. A feature binds to one profile through `authProfileKey`.
2. `externalValidation`
   Optional rule-level HTTP validation executed only after the feature has already passed its access policy and inbound auth checks.

### Available Request Inputs

During external validation, templates can reference both the evaluation context and the incoming request:

- `{{user.id}}`
- `{{context.user.id}}`
- `{{tenant.id}}`
- `{{input.headers.authorization}}`
- `{{input.headers.x-student-id}}`
- `{{input.auth.bearerToken}}`
- `{{input.auth.apiKey}}`
- `{{input.request.ip}}`
- `{{input.request.method}}`

`requestMapping` reads from those paths and builds the JSON body sent to the external validator. `target` accepts nested paths such as `payload.userId`.

For builder-generated fixed values, `requestMapping.source` also supports:

- `literal:<text>`
- `number:<value>`
- `boolean:true|false`
- `json:<raw-json>`

Example:

```json
{
  "requestMapping": [
    { "source": "user.id", "target": "payload.userId" },
    { "source": "input.auth.bearerToken", "target": "payload.token" }
  ],
  "headers": {
    "Authorization": "Bearer {{input.auth.bearerToken}}",
    "X-Student-Id": "{{input.headers.x-student-id}}"
  }
}
```

### Success Contract

Default behavior:

- if `responseCondition` is empty, any `2xx` response means the validation passed;
- any non-`2xx` response means the validation did not pass;
- transport errors, timeouts, DNS failures, and open circuit breakers still go through `failMode`.

When you need explicit logic, `responseCondition` can inspect:

- `response`
  Parsed JSON body when the response is JSON. If the body is not JSON, it falls back to the raw text.
- `responseText`
  Raw response body as text.
- `http.status`
  HTTP status code.
- `http.headers["x-decision"]`
  Response headers, using bracket notation for dashed names.

Examples:

```text
http.status == 200 && response.authorized == true
http.status in [200, 204]
http.status == 403 && response.reason == "plan_blocked"
responseText == "ok"
responseText contains "ok"
```

### AuthProfile Type by Type

#### `api_key`

Use when the incoming eval request must carry a fixed secret known by the evaluator.

- no external call is made
- `secretPayload`: `{ "apiKey": "..." }`
- `config.location`: `header | query`
- `config.name`: header name or query param name
- `config.prefix`: optional prefix stripped before comparison
- `cacheTTLSeconds` is always forced to `0`

#### `oidc_standard`

Use when the incoming bearer token should be validated against an external OIDC provider using the standard discovery and JWKS flow.

- reads `Authorization: Bearer <token>` from the incoming request
- discovers `/.well-known/openid-configuration` from the configured issuer
- fetches and caches `jwks_uri` keys internally
- validates signature, `iss`, `aud`, `exp`, and `nbf`
- `config.issuer`
- `config.audience`
- `cacheTTLSeconds` is always forced to `0` because JWKS/discovery already use internal cache

#### `custom`

Use when validation requires a custom HTTP request built from the incoming eval request.

- `config.url`
- `config.method`: `GET | POST`
- `config.timeout`
- `config.headers`: array of mappings from `request.header(name)` to external headers
- `config.body`: array of mappings from `request.header(name)` or `request.body(jsonPath)` to external JSON body fields
- `config.successRule`: `any_2xx | status | json_field | response_header | text_contains`
- optional `config.outboundAuthHeaderName` + `secretPayload.outboundApiKey`
- successful validations may be cached with `cacheTTLSeconds`

## Console Builder Model

The admin console exposes guided builders instead of raw `type + JSON` as the primary UX:

- `AuthProfile`
  - API key or fixed secret
  - automatic OAuth2 passthrough
  - custom validator
- `externalValidation`
  - headers and body mapped from the incoming request or evaluation context
  - optional inline API key for the destination validator
  - success rule builder

## Caching Strategy

| Data                 | Redis Key                                 | TTL             | Invalidation                        |
| -------------------- | ----------------------------------------- | --------------- | ----------------------------------- |
| Feature config       | `fe:feature:{key}`                        | 60s             | Immediate on write + TTL safety net |
| Segment membership   | `fe:seg:{segmentKey}:{userId}:{tenantId}` | 300s            | Pattern delete on import/delete     |
| External call result | `fe:ext:{sha256}`                         | Per-rule config | TTL only                            |
| Member by email      | `fe:member:{email}`                       | 300s            | Delete on member update             |
| Pack activations     | `fe:pack:{packKey}`                       | 300s            | Delete on activation change         |
| JWKS                 | In-memory                                 | 3600s           | Timer refresh                       |
| Compiled expressions | In-memory LRU (10K)                       | None            | New key on expression change        |

> **Important:** Redis failure triggers fallback to direct PostgreSQL queries. The circuit breaker on Redis opens after 3 errors and retries with PING after 30s.

## Testing

```bash
# All tests with race detection and coverage
go test -C server -race -cover ./...

# Specific package
go test -C server -race ./internal/engine/...

# With verbose output
go test -C server -race -v ./internal/domain/...
```

Tests follow table-driven patterns with `testify/assert`. Mocks are generated from domain interfaces.

## Code Quality

Enforced by `golangci-lint` (config: `.golangci.yml`):

| Category    | Linters                                 | Limits                            |
| ----------- | --------------------------------------- | --------------------------------- |
| Correctness | govet, errcheck, staticcheck, typecheck | --                                |
| Complexity  | cyclop, gocognit, funlen, nestif        | CC 15, CogC 20, 80 lines, depth 4 |
| Style       | revive, gofmt, goimports, misspell      | --                                |
| Security    | gosec                                   | --                                |
| Performance | prealloc, bodyclose                     | --                                |

```bash
# Run linter
golangci-lint run -c server/.golangci.yml server/...
```

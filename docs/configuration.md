# Configuration

Backend configuration is loaded from environment variables through Viper. Frontend configuration is provided through Vite `VITE_*` variables.

## Important Loading Note

The backend uses `viper.AutomaticEnv()`. That means environment variables must be exported in the shell or provided by the process manager. `.env` files are not loaded automatically by the backend unless your own wrapper does it.

## Backend (`server/`)

### Server

| Variable                  | Description               | Default |
| ------------------------- | ------------------------- | ------- |
| `SERVER_PORT`             | HTTP server port          | `8080`  |
| `SERVER_SHUTDOWN_TIMEOUT` | graceful shutdown timeout | `10s`   |

The HTTP server also sets `ReadHeaderTimeout=5s` in code.

### PostgreSQL

| Variable                      | Description                  | Default                                                                |
| ----------------------------- | ---------------------------- | ---------------------------------------------------------------------- |
| `DATABASE_URL`                | PostgreSQL connection string | `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable` |
| `POSTGRES_MAX_CONNS`          | maximum pool size            | `20`                                                                   |
| `POSTGRES_MIN_CONNS`          | minimum pool size            | `2`                                                                    |
| `POSTGRES_MAX_CONN_LIFETIME`  | max connection lifetime      | `30m`                                                                  |
| `POSTGRES_MAX_CONN_IDLE_TIME` | max idle time per connection | `5m`                                                                   |
| `POSTGRES_HEALTHCHECK_PERIOD` | pool healthcheck interval    | `1m`                                                                   |
| `POSTGRES_CONNECT_TIMEOUT`    | connection timeout           | `10s`                                                                  |

Startup behavior:

- PostgreSQL migrations run automatically on boot
- the `default` workspace is created automatically if missing

### Redis

| Variable         | Description    | Default          |
| ---------------- | -------------- | ---------------- |
| `REDIS_URI`      | Redis address  | `localhost:6379` |
| `REDIS_PASSWORD` | Redis password | empty            |

Redis is used for:

- rate limiting
- segment membership cache
- pack activation cache
- running experiment cache
- metrics aggregation

Redis is fail-open for rate limiting and caching. If Redis is unavailable, the API continues using direct storage fallbacks where applicable.

### Authentication

| Variable         | Description                               | Default         |
| ---------------- | ----------------------------------------- | --------------- |
| `AUTH_DISABLED`  | skip JWT validation for development       | `false`         |
| `DEV_USER_EMAIL` | injected user email in auth-disabled mode | `dev@local.dev` |
| `DEV_USER_ROLE`  | injected role in auth-disabled mode       | `owner`         |

When `AUTH_DISABLED=true`:

- bearer validation is skipped
- a mock authenticated user is injected
- the system can auto-bootstrap the first owner in the current workspace

### OIDC

| Variable        | Description           | Default |
| --------------- | --------------------- | ------- |
| `OIDC_ISSUER`   | OIDC issuer URL       | empty   |
| `OIDC_AUDIENCE` | expected JWT audience | empty   |

Production deployments should set both values.

### CORS

| Variable             | Description                       | Default                 |
| -------------------- | --------------------------------- | ----------------------- |
| `CORS_ALLOW_ORIGINS` | comma-separated inherited origins | `http://localhost:5173` |

Each value must be a full origin (`scheme://host[:port]`) without path, query, or fragment. Requests with an `Origin` header that is not on the effective list are rejected with `403`.

### Security policy management

The effective runtime policy is the union of:

- inherited entries from `CORS_ALLOW_ORIGINS`
- app-managed entries stored in PostgreSQL and editable from the console at `/settings/security`

Inherited env entries remain active and read-only in the UI. The console only replaces the app-managed portion of the policy.
This screen controls browser CORS access only. Server-to-server callers that authenticate with Bearer tokens or API keys are not gated by origin/domain.

### Rate Limiting

| Variable           | Description                            | Default |
| ------------------ | -------------------------------------- | ------- |
| `RATE_LIMIT_EVAL`  | requests/sec for eval and OFREP routes | `500`   |
| `RATE_LIMIT_ADMIN` | requests/sec for admin routes          | `60`    |

### Logging

| Variable     | Description                      | Default |
| ------------ | -------------------------------- | ------- |
| `LOG_LEVEL`  | `debug`, `info`, `warn`, `error` | `info`  |
| `LOG_FORMAT` | `text` or `json`                 | `text`  |

When `LOG_FORMAT=json`, Gin is switched to release mode.

## PostgreSQL Schema Notes

The current schema is workspace-aware. The exact index set is created by startup migrations, but the important constraints are:

- feature keys are unique per workspace
- segment keys are unique per workspace
- pack keys are unique per workspace
- tag keys are unique per workspace
- member email is unique per workspace
- changelog entries are indexed by entity, actor, and `createdAt`
- schedules are indexed by `status + scheduledAt` and `featureKey`
- only one running experiment is allowed per `featureKey` and workspace
- conversions are unique per `experimentId + userId + metricKey`

Retention and cleanup rules:

- `evaluation_errors` retention is managed explicitly
- expired `pack_activations` can be cleaned asynchronously

## Frontend (`console/`)

### Vite Variables

| Variable                 | Description              | Default / Example                |
| ------------------------ | ------------------------ | -------------------------------- |
| `VITE_API_URL`           | backend base URL         | `http://localhost:8080/features` |
| `VITE_AUTH_DISABLED`     | bypass OIDC in local dev | unset                            |
| `VITE_OIDC_AUTHORITY`    | OIDC authority URL       | unset                            |
| `VITE_OIDC_CLIENT_ID`    | OIDC client ID           | unset                            |
| `VITE_OIDC_REDIRECT_URI` | OAuth callback URI       | `{origin}/auth/callback`         |

The frontend persists the selected workspace client-side and sends it as `X-Workspace` on every API call.

## Local Development Example

### Backend env

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable
POSTGRES_MAX_CONNS=20
POSTGRES_MIN_CONNS=2
REDIS_URI=localhost:6379
REDIS_PASSWORD=
AUTH_DISABLED=true
DEV_USER_EMAIL=admin@local.dev
DEV_USER_ROLE=owner
CORS_ALLOW_ORIGINS=http://localhost:5173,http://localhost:5174
LOG_LEVEL=debug
LOG_FORMAT=text
```

### Frontend env

```env
VITE_API_URL=http://localhost:8080/features
VITE_AUTH_DISABLED=true
```

### Start Commands

```bash
make redis
make server
make console
```

Or:

```bash
make dev
```

## Production Notes

- set `AUTH_DISABLED=false`
- set `OIDC_ISSUER` and `OIDC_AUDIENCE`
- keep `VITE_AUTH_DISABLED` unset
- point `VITE_API_URL` to the deployed `/features` path
- if Redis requires authentication, set `REDIS_PASSWORD`

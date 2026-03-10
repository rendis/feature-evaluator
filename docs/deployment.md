# Deployment

## Docker Images

Both the backend and frontend have independent Dockerfiles optimized for production.

### Backend (`server/Dockerfile`)

Multi-stage build:

1. **Builder stage:** Compiles Go binary with `CGO_ENABLED=0` and stripped debug info (`-ldflags="-s -w"`)
2. **Runtime stage:** Alpine 3.21 with ca-certificates and tzdata, runs as non-root `appuser`

```bash
docker build -t feature-evaluator-server server/
docker run -p 8080:8080 \
  -e DATABASE_URL=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable \
  -e REDIS_URI=redis:6379 \
  -e OIDC_ISSUER=https://auth.example.com \
  -e OIDC_AUDIENCE=feature-evaluator \
  -e LOG_FORMAT=json \
  feature-evaluator-server
```

### Frontend (`console/Dockerfile`)

Multi-stage build:

1. **Base stage:** Node 22 Alpine with pnpm enabled
2. **Deps stage:** Installs dependencies with `--frozen-lockfile`
3. **Build stage:** Runs `pnpm build` (TypeScript compile + Vite build)
4. **Runtime stage:** Nginx 1.27 Alpine serving the static `dist/` directory

```bash
docker build -t feature-evaluator-console console/ \
  --build-arg VITE_API_URL=https://api.example.com/features \
  --build-arg VITE_OIDC_AUTHORITY=https://auth.example.com \
  --build-arg VITE_OIDC_CLIENT_ID=feature-evaluator-console

docker run -p 80:80 feature-evaluator-console
```

> **Note:** Frontend environment variables are baked into the build at compile time since Vite replaces `import.meta.env.*` during the build process.

## Nginx Configuration

The frontend nginx config handles:

- **SPA routing:** All paths fall back to `index.html` for client-side routing
- **API proxy:** `/features` requests are proxied to `http://server:8080`
- **Static assets:** JS, CSS, images, and fonts get 1-year cache with `immutable`
- **Compression:** Gzip enabled for text, JSON, and JavaScript (min 256 bytes)

## Docker Compose (Development)

The root `docker-compose.yml` provides Redis for local development:

```yaml
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 3
```

Start with:

```bash
docker compose up -d redis
# or
make redis
```

PostgreSQL is expected to be provided separately. In local development this repo reuses the existing `postgres-local` container on `localhost:5432`.

## Production Checklist

### Backend

- [ ] Set `DATABASE_URL` to the production PostgreSQL connection string
- [ ] Set `REDIS_URI` to production Redis endpoint
- [ ] Set `OIDC_ISSUER` and `OIDC_AUDIENCE` for JWT validation
- [ ] Set `AUTH_DISABLED=false` (or remove -- defaults to false)
- [ ] Set `LOG_FORMAT=json` for structured logging
- [ ] Set `LOG_LEVEL=info` (or `warn` for reduced verbosity)
- [ ] Set `CORS_ALLOW_ORIGINS` to production frontend URL
- [ ] Review rate limits: `RATE_LIMIT_EVAL`, `RATE_LIMIT_ADMIN`
- [ ] Ensure PostgreSQL indexes are created (auto-migrated on startup)

### Frontend

- [ ] Build with production environment variables
- [ ] Set `VITE_API_URL` to production backend URL
- [ ] Set `VITE_OIDC_AUTHORITY`, `VITE_OIDC_CLIENT_ID`, `VITE_OIDC_REDIRECT_URI`
- [ ] Remove `VITE_AUTH_DISABLED` (must not be set in production)

### Infrastructure

- [ ] PostgreSQL: configure network access, database user, connection string, and `pg_trgm`
- [ ] Redis: configure persistence, memory limits, and auth if needed
- [ ] TLS termination at load balancer or API gateway level
- [ ] Health checks: `/features/healthz` (liveness), `/features/readyz` (readiness)

## Health Checks

| Endpoint                | Type      | Checks                |
| ----------------------- | --------- | --------------------- |
| `GET /features/healthz` | Liveness  | Server is running     |
| `GET /features/readyz`  | Readiness | PostgreSQL + Redis alive |

Configure your container orchestrator to use these endpoints:

```yaml
# Kubernetes example
livenessProbe:
  httpGet:
    path: /features/healthz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
readinessProbe:
  httpGet:
    path: /features/readyz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

## Graceful Shutdown

The backend handles `SIGINT` and `SIGTERM` signals:

1. Stops accepting new connections
2. Stops the metrics collector
3. Waits for in-flight requests to complete (configurable timeout, default 10s)
4. Closes PostgreSQL connection
5. Closes Redis connection

## Code Quality Gates

Run all checks before deploying:

```bash
make quality
```

This runs:

- `golangci-lint` on the Go backend (30+ linters)
- ESLint on the React frontend
- TypeScript type checking (`tsc --noEmit`)
- All tests with race detection and coverage

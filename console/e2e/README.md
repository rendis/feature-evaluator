# Playwright E2E

This folder contains the browser automation stack for the console.

## Local run

```bash
pnpm -C console exec playwright install chromium
pnpm -C console run e2e
```

The default Playwright config starts the Vite dev server with:

- `VITE_AUTH_DISABLED=true`
- `VITE_API_URL=http://127.0.0.1:8080/features`

The tests use mocked network responses for the new cache-config and observability
flows so they can run deterministically even when the backend surface is not yet
available.

## Environment assumptions

- Console runs locally on `http://127.0.0.1:4173`
- Backend is optional for the harnessed flows, but if you point Playwright at a
  real stack, it should be backed by Postgres + Redis in the usual repo setup
- The browser specs seed `fe-workspace` in localStorage for auth-disabled runs

## Coverage

- Feature cache config create/edit
- Rule external binding cache create/edit
- Auth profile cache create/edit against the real console UI
- Segment cache config
- Experiment cache config
- Feature observability summary/rules/traces smoke flow

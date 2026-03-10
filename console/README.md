# Feature Evaluator -- Admin Console

React 19 SPA for managing features, rules, segments, packs, and team members. Built with Vite 7, TypeScript 5.9, TanStack Router, and shadcn/ui.

## Architecture

```
src/
├── main.tsx                    # App entry point
├── app.tsx                     # Root provider stack (Query, Router, i18n, Auth)
├── globals.css                 # Tailwind + OKLCH theme variables
├── config/
│   ├── env.ts                  # Runtime environment config
│   ├── query-client.ts         # TanStack Query defaults
│   └── oidc.ts                 # OIDC provider config
├── api/
│   ├── client.ts               # Axios/fetch wrapper with auth headers
│   ├── types.ts                # Shared API types
│   ├── features.ts             # Feature API functions
│   ├── rules.ts                # Rule API functions
│   ├── segments.ts             # Segment API functions
│   ├── packs.ts                # Pack API functions
│   ├── members.ts              # Member API functions
│   ├── apikeys.ts              # API key API functions
│   ├── tags.ts                 # Tag API functions
│   ├── dashboard.ts            # Dashboard + metrics API
│   ├── audit.ts                # Audit API functions
│   └── expression.ts           # Expression validation/test
├── auth/
│   ├── auth-provider.tsx       # OIDC context provider (oidc-client-ts)
│   ├── auth-guard.tsx          # Route-level auth check
│   ├── callback-handler.tsx    # OAuth callback processing
│   ├── permission-guard.tsx    # Permission-based UI gating
│   └── roles.ts                # Role -> permission mapping
├── queries/                    # TanStack Query definitions
├── mutations/                  # TanStack Query mutations
├── hooks/
│   ├── use-auth.ts             # Auth state + token access
│   ├── use-current-user.ts     # Current member data
│   ├── use-permissions.ts      # Permission checks
│   ├── use-debounce.ts         # Debounced value
│   ├── use-media-query.ts      # CSS media query hook
│   └── use-mobile.ts           # Mobile breakpoint detection
├── stores/                     # Zustand stores
│   ├── theme-store.ts          # Light / dark / system preference
│   ├── sidebar-store.ts        # Sidebar open/collapsed state
│   └── locale-store.ts         # Language preference (es/en)
├── i18n/
│   └── index.ts                # i18next config (lazy namespaces)
├── components/
│   ├── ui/                     # shadcn/ui primitives
│   ├── layout/                 # App shell, sidebar, header, toggles
│   ├── features/               # Feature list, card, form, detail, toggle
│   ├── rules/                  # Rule list, form, expression editor
│   ├── segments/               # Segment list, form, member import
│   ├── packs/                  # Pack list, form, features tab, activations
│   ├── dashboard/              # Stats, activity, metrics charts
│   ├── settings/               # Member management, API keys
│   ├── audit/                  # Error log table and filters
│   └── shared/                 # Reusable: empty-state, error-state, loading, confirm-dialog
├── workers/                    # Web Workers (CSV parsing)
└── routes/                     # TanStack Router (file-based)
    ├── __root.tsx
    ├── _authenticated.tsx       # Layout wrapper + auth guard
    ├── _authenticated/
    │   ├── index.tsx            # Dashboard
    │   ├── features/            # Feature list + detail
    │   ├── segments/            # Segment list + detail
    │   ├── settings/            # Members, packs
    │   └── audit.tsx            # Error audit log
    ├── login.tsx
    └── auth/
        ├── callback.tsx         # OIDC callback
        └── access-denied.tsx    # Unauthorized page
```

## Routing

File-based routing via [TanStack Router](https://tanstack.com/router) + Vite plugin. Route tree is auto-generated in `routeTree.gen.ts`.

| Route                 | Page            | Auth |
| --------------------- | --------------- | :--: |
| `/`                   | Dashboard       |  Y   |
| `/features`           | Feature list    |  Y   |
| `/features/:key`      | Feature detail  |  Y   |
| `/segments`           | Segment list    |  Y   |
| `/segments/:key`      | Segment detail  |  Y   |
| `/settings/packs`     | Pack list       |  Y   |
| `/settings/packs/:key`| Pack detail     |  Y   |
| `/settings/members`   | Team management |  Y   |
| `/audit`              | Error audit log |  Y   |
| `/login`              | OIDC redirect   |  -   |
| `/auth/callback`      | OAuth callback  |  -   |
| `/auth/access-denied` | Access denied   |  -   |

The `_authenticated` layout wraps all protected routes with `AuthGuard` + `AppShell` (sidebar, header).

## State Management

| Concern      | Solution              | Details                                                |
| ------------ | --------------------- | ------------------------------------------------------ |
| Server state | TanStack Query v5     | staleTime 30s, gcTime 5min, retry 1                    |
| Auth state   | oidc-client-ts        | Token in sessionStorage, auto silent renew             |
| Theme        | Zustand               | `light` / `dark` / `system`, persisted to localStorage |
| Sidebar      | Zustand               | Open/collapsed state, persisted                        |
| Locale       | Zustand               | `es` / `en`, persisted to localStorage                 |
| Forms        | React Hook Form + Zod | Validation schemas, resolver integration               |

## Theme System

CSS variables using OKLCH color space for perceptually uniform light/dark modes. No pure black or pure white -- warm, soft tones throughout.

- Toggle: light / dark / system (respects OS preference)
- Contrast: WCAG AA verified
- Switch: `.dark` class on `<html>`

### Breakpoints

| Name    | Range        | Behavior                                          |
| ------- | ------------ | ------------------------------------------------- |
| Mobile  | 0 - 1023px   | Card layouts, stacked forms, overlay sidebar      |
| Desktop | 1024px+      | Full tables, persistent sidebar, text mode editor |

## i18n

Spanish is the default language. English available via toggle. Uses `react-i18next` with lazy-loaded namespaces per route.

- Detection: `localStorage` only (no navigator detection)
- Fallback: `es`
- Namespaces: `common`, `auth`, `features`, `rules`, `segments`, `packs`, `settings`, `audit`, `validation`
- Locale files: `public/locales/{es,en}/*.json`

## Development

```bash
# Start dev server (port 5173)
pnpm -C console run dev

# Type checking
pnpm -C console run typecheck

# Lint
pnpm -C console run lint

# Format
pnpm -C console run format
```

Or via root Makefile:

```bash
make console       # Start dev server
make lint-console  # ESLint
make typecheck     # tsc --noEmit
```

### Environment

Create `console/.env` for local development:

```env
VITE_API_URL=http://localhost:8080/features
VITE_AUTH_DISABLED=true
```

> **Tip:** With `VITE_AUTH_DISABLED=true`, the console bypasses OIDC and uses the backend's mock user. No OIDC provider needed for local dev.

## Code Quality

### ESLint 9 (flat config)

Config: `eslint.config.ts`

Key rules:

- `@typescript-eslint/no-explicit-any: error`
- `@typescript-eslint/no-unused-vars: error`
- `complexity: [warn, 15]`
- `max-depth: [warn, 4]`
- `import-x/no-cycle: error`
- `react-hooks/exhaustive-deps`

### Prettier

Config: `.prettierrc`

```json
{
  "singleQuote": true,
  "semi": true,
  "trailingComma": "all",
  "printWidth": 100,
  "tabWidth": 2
}
```

## Key Dependencies

| Package               | Purpose                          |
| --------------------- | -------------------------------- |
| `react` 19            | UI framework                     |
| `@tanstack/react-router` | File-based routing            |
| `@tanstack/react-query`  | Server state management       |
| `zustand`             | Client state management          |
| `oidc-client-ts`      | OIDC authentication              |
| `react-hook-form`     | Form management                  |
| `zod`                 | Schema validation                |
| `react-i18next`       | Internationalization             |
| `tailwindcss` v4      | Utility-first CSS                |
| `recharts`            | Dashboard charts                 |
| `codemirror` 6        | Expression text editor           |
| `papaparse`           | CSV parsing for segment import   |
| `sonner`              | Toast notifications              |
| `lucide-react`        | Icon library                     |
| `cmdk`                | Command palette                  |

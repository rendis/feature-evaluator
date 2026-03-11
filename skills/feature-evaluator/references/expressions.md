# Expression System Reference

## Table of Contents

- [Eval API](#eval-api)
- [Expression Environment](#expression-environment)
- [Derived Fields](#derived-fields)
- [Builtin Functions](#builtin-functions)
- [Security Constraints](#security-constraints)
- [Auth Flow into Expressions](#auth-flow-into-expressions)
- [Evaluation Pipeline](#evaluation-pipeline)
- [Expression Examples](#expression-examples)
- [Rollout](#rollout)
- [Eval Response](#eval-response)

---

## Eval API

Three endpoints, all under `/features`:

| Endpoint                | Method | Purpose                       |
| ----------------------- | ------ | ----------------------------- |
| `/features/eval`        | POST   | Evaluate single feature       |
| `/features/eval/bulk`   | POST   | Evaluate multiple features    |
| `/features/eval/active` | POST   | Evaluate all enabled features |

**Auth**: `Authorization: Bearer <jwt>` or `X-Api-Key: <key>`

### Request Format

**Single eval:**

```json
{
  "featureKey": "premium-dashboard",
  "context": {
    "user": { "id": "user-123", "email": "alice@empresa.com", "role": "admin" },
    "tenant": { "id": "tenant-456" },
    "campus": { "id": "campus-789" },
    "program": { "id": "program-000" },
    "custom": { "country": "US", "accountAge": 180 }
  },
  "environment": "production"
}
```

**Bulk eval:**

```json
{
  "features": [
    { "featureKey": "feat-a", "context": { ... } },
    { "featureKey": "feat-b", "context": { ... } }
  ]
}
```

**All active:**

```json
{
  "context": { "user": { ... }, "tenant": { ... } },
  "environment": "production",
  "tags": ["billing"]
}
```

### Context Namespaces

All keys under `context` become top-level variables in expressions. Standard namespaces:

- `user` — user attributes (any fields: id, email, role, tier, etc.)
- `tenant` — tenant attributes
- `campus` — campus attributes
- `program` — program attributes
- `custom` — arbitrary custom data

Any additional keys also become top-level variables.

---

## Expression Environment

All variables available to rule expressions:

| Variable         | Type | Source                         | Description                               |
| ---------------- | ---- | ------------------------------ | ----------------------------------------- |
| `user.*`         | map  | `context.user` in body         | User attributes (any fields caller sends) |
| `tenant.*`       | map  | `context.tenant` in body       | Tenant attributes                         |
| `campus.*`       | map  | `context.campus` in body       | Campus attributes                         |
| `program.*`      | map  | `context.program` in body      | Program attributes                        |
| Any custom ns    | map  | `context.<key>` in body        | Additional namespaces                     |
| `headers.*`      | map  | HTTP headers via InputContract | Normalized request headers                |
| `requestBody.*`  | map  | Request body                   | Raw request body fields                   |
| `authenticated`  | bool | Auth profile validation        | Top-level boolean                         |
| `derived.*`      | map  | Auto-extracted (see below)     | Derived auth/identity fields              |
| `<segmentKey>.*` | map  | Segment source binding         | Resolved segment record attributes        |

`user` namespace always exists (empty map if not provided).

---

## Derived Fields

Auto-extracted from JWT and/or context. Located in `derived.*`:

| Field                        | Type   | Source                                           | Description                       |
| ---------------------------- | ------ | ------------------------------------------------ | --------------------------------- |
| `derived.authenticated`      | bool   | Auth profile result                              | Same as top-level `authenticated` |
| `derived.bearerTokenPresent` | bool   | Auto                                             | True if Bearer token in request   |
| `derived.apiKeyPresent`      | bool   | Auto                                             | True if API key in request        |
| `derived.email`              | string | JWT `email` claim, fallback `context.user.email` | User email                        |
| `derived.userId`             | string | JWT `sub` claim, fallback `context.user.id`      | User identifier                   |
| `derived.subject`            | string | JWT `sub` claim only                             | JWT subject (no fallback)         |
| `derived.name`               | string | JWT `name` claim only                            | User display name (no fallback)   |

### Priority Rules

JWT claims take precedence over context body:

| Scenario                                 | `derived.email`           | `user.email`              |
| ---------------------------------------- | ------------------------- | ------------------------- |
| JWT only (`Authorization: Bearer <jwt>`) | From JWT `email` claim    | `undefined`               |
| Context only (`context.user.email`)      | From `context.user.email` | From `context.user.email` |
| Both JWT + context                       | From JWT (wins)           | From context body         |

**Key insight:** `derived.email` is always populated if email is available (JWT or fallback). `user.email` only exists if the caller sends it in the request body.

---

## Builtin Functions

| Function                    | Signature           | Returns     | Description                            |
| --------------------------- | ------------------- | ----------- | -------------------------------------- |
| `now()`                     | `() → time`         | `time.Time` | Current UTC time                       |
| `dateBefore(date, ref)`     | `(any, any) → bool` | `bool`      | True if `date` is before `ref`         |
| `dateAfter(date, ref)`      | `(any, any) → bool` | `bool`      | True if `date` is after `ref`          |
| `contains(value, needle)`   | `(any, any) → bool` | `bool`      | True if string contains substring      |
| `startsWith(value, prefix)` | `(any, any) → bool` | `bool`      | True if string starts with prefix      |
| `endsWith(value, suffix)`   | `(any, any) → bool` | `bool`      | True if string ends with suffix        |
| `inSegment(key)`            | `(string) → bool`   | `bool`      | True if user belongs to segment        |
| `externalApi(key)`          | `(string) → bool`   | `bool`      | True if external API validation passes |

### Date Formats

`dateBefore` and `dateAfter` accept `time.Time` or strings in these formats:

- RFC3339: `2025-06-01T15:04:05Z`
- DateTime: `2025-06-01T15:04:05`
- Date only: `2025-06-01`

Both arguments are required (exactly 2).

---

## Security Constraints

Expressions are validated before compilation:

| Constraint                | Limit            |
| ------------------------- | ---------------- |
| Max AST depth             | 10               |
| Max AST nodes             | 100              |
| Max `inSegment()` calls   | 5 per expression |
| Max string literal length | 1000 characters  |

**Denied patterns** (case-insensitive): `exec`, `system`, `import`, `require`, `__proto__`, `constructor`, `prototype`, `eval`, `Function`, `process`

---

## Auth Flow into Expressions

1. Caller sends `Authorization: Bearer <jwt>` header
2. JWT payload decoded **without signature verification** at eval level
3. Claims `sub`, `email`, `name` extracted → `derived.*` fields
4. If feature has an auth profile configured → signature **verified** → `authenticated = true`
5. If no JWT present, fallback: `context.user.email` → `derived.email`, `context.user.id` → `derived.userId`

### Access Policies

Features define how auth is enforced:

| Policy     | Behavior                                               |
| ---------- | ------------------------------------------------------ |
| `public`   | No auth required, eval always proceeds                 |
| `optional` | Auth attempted if configured, eval proceeds regardless |
| `required` | Auth must succeed, returns 401 if unauthorized         |

### Auth Profile Types

| Type          | Description                                                       |
| ------------- | ----------------------------------------------------------------- |
| OIDC Standard | Validates JWT signature via JWKS discovery (issuer + audience)    |
| API Key       | Fixed key validation (header or query param)                      |
| Custom HTTP   | HTTP request to external validator with configurable success rule |

---

## Evaluation Pipeline

Complete flow for `POST /features/eval`:

1. **Fetch feature** → return 404 if not found
2. **Auth validation** → if feature has `authProfileKey`, validate request; enforce `accessPolicy`
3. **Feature checks** → enabled? scheduling (activeFrom/activeUntil)? environment matches?
4. **Trial override** → if trial active, return `trialValue`
5. **Experiment check** → if running experiment, assign variant
6. **Segment resolution** → pre-scan all rules for `inSegment()` calls → batch resolve memberships
7. **Pack activation** → check if feature granted by active pack (tenant/campus/program)
8. **Rule evaluation** (by priority, ascending):
   - Prepare expression input (headers, body, derived, segment sources)
   - Build expression environment
   - Evaluate expression → must return `bool`
   - If true: check rollout percentage → return rule `value`
   - First match wins
9. **Fallback** → return `defaultValue`

---

## Expression Examples

### Email-based Targeting

```
# From JWT (automatic extraction)
derived.email == "admin@empresa.com"
endsWith(derived.email, "@empresa.com")

# From context body
user.email == "admin@empresa.com"
endsWith(user.email, "@empresa.com")
```

### Auth & Role Targeting

```
authenticated
authenticated && user.role == "admin"
derived.bearerTokenPresent && user.tier == "enterprise"
```

### Segment Membership

```
inSegment("beta-users")
inSegment("vip-customers") && tenant.plan == "enterprise"
```

### Date-based Scheduling

```
dateAfter(now(), "2025-06-01")
dateAfter(now(), "2025-06-01") && dateBefore(now(), "2025-12-31")
```

### Combined Targeting

```
tenant.id == "acme" && user.role == "admin" && inSegment("early-access")
contains(user.email, "+beta@") && authenticated
```

### Header & Request Body

```
headers.region == "us-east-1"
requestBody.action == "purchase" && requestBody.amount > 100
```

### External API Validation

```
externalApi("fraud-check")
externalApi("kyc-verified") && user.purchaseAmount > 1000
```

---

## Rollout

Rules can have `rolloutPercentage` (0-100):

- **Deterministic**: FNV-1a hash of `featureKey + rolloutSalt + userId`
- **Monotonic**: increasing percentage always includes previously included users
- `userId` comes from `derived.userId`
- If no userId available, rollout check is skipped (rule matches all)

---

## Eval Response

```json
{
  "featureKey": "premium-dashboard",
  "value": true,
  "valueType": "boolean",
  "environment": "production",
  "matchedRule": { "id": "uuid", "name": "Admin Access" },
  "packGrant": "enterprise-pack",
  "inTrial": false,
  "trialEndsAt": null,
  "tierKeys": ["tier-enterprise"],
  "experiment": { "experimentId": "uuid", "variantKey": "variant-a" },
  "segments": [{ "key": "beta-users", "member": true }],
  "metadata": {},
  "reason": "matched_rule",
  "evaluatedAt": "2025-06-01T12:00:00Z",
  "error": null
}
```

**Reason values**: `matched_rule`, `default_value`, `pack_grant`, `trial_active`, `feature_disabled`, `environment_mismatch`, `scheduling_inactive`, `unauthorized`, `error`

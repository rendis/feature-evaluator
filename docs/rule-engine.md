# Rule Engine

The Feature Evaluator uses [expr-lang](https://github.com/expr-lang/expr) as its expression language. Expressions are compiled to bytecode and cached in an LRU cache (max 10,000 entries, keyed by SHA-256 of expression text).

## Context Variables

Every expression has access to these variables:

| Variable        | Type             | Description                         |
| --------------- | ---------------- | ----------------------------------- |
| `user`          | `map[string]any` | User attributes from `context.user` |
| `tenant`        | `string`         | From `X-Tenant-ID` header           |
| `campus`        | `string`         | From `X-Campus-ID` header           |
| `program`       | `string`         | From `X-Program-ID` header          |
| `authenticated` | `bool`           | Whether request has valid JWT       |

The `user` map is fully dynamic -- any attributes sent in the evaluation request are available.

## Custom Functions

| Function     | Signature                           | Description                                                     |
| ------------ | ----------------------------------- | --------------------------------------------------------------- |
| `inSegment`  | `inSegment(key string) bool`        | Check segment membership (tenant-scoped first, global fallback) |
| `now`        | `now() time.Time`                   | Current UTC time                                                |
| `dateBefore` | `dateBefore(date, ref string) bool` | Returns true if `date` is before `ref`                          |
| `dateAfter`  | `dateAfter(date, ref string) bool`  | Returns true if `date` is after `ref`                           |

### inSegment()

Checks if the user is a member of the named segment. Lookup order:

1. Tenant-scoped: matches `segmentKey` + `userId` + `tenantId`
2. Global fallback: matches `segmentKey` + `userId` with empty `tenantId`

Results are cached in Redis for 300 seconds.

**Batch optimization:** Before evaluating rules, the engine pre-scans all expressions for `inSegment()` calls and resolves all segment memberships in a single batch query.

### Date Functions

Date strings are parsed with these formats (tried in order):

- `2006-01-02T15:04:05Z07:00` (RFC 3339)
- `2006-01-02`

## Expression Examples

### Basic Comparisons

```
user.plan == "enterprise"
user.companySize > 100
user.age >= 18 && user.age <= 65
```

### String Operations

```
user.email endsWith "@example.com"
user.name contains "admin"
user.country in ["CL", "CO", "MX"]
```

### Logical Combinations

```
user.plan == "enterprise" && user.companySize > 100
tenant in ["cl", "co"] || user.email endsWith "@example.com"
!(user.role == "guest")
```

### Segment Membership

```
inSegment("beta-testers")
inSegment("beta-testers") && user.active == true
inSegment("enterprise-users") || user.plan == "enterprise"
```

### Date/Time Conditions

```
dateBefore(user.trialEnd, now())
dateAfter(user.subscriptionStart, "2025-01-01")
```

### Multi-Tenant Scoping

```
tenant == "cl" && user.plan == "premium"
tenant in ["cl", "co"] && inSegment("early-adopters")
```

### Complex Rules

```
(user.plan == "enterprise" || inSegment("beta-testers")) && user.active == true && tenant in ["cl", "co"]
```

## Rule Evaluation Order

Rules within a feature are evaluated in priority order (lowest number = highest priority). The first matching rule wins.

```
Rule 1 (priority: 0): Enterprise users --> value: true
Rule 2 (priority: 1): Beta testers    --> value: true
Rule 3 (priority: 2): Chile only      --> value: "variant-b"
Default                                --> value: false
```

## Rule Scope

Each rule can be scoped to specific tenants, campuses, and programs. A rule is only evaluated if the request's tenant/campus/program headers match the scope. Empty scope means "all".

```json
{
  "scope": {
    "tenantIds": ["cl", "co"],
    "campusIds": [],
    "programIds": []
  }
}
```

This rule only applies to requests with `X-Tenant-ID: cl` or `X-Tenant-ID: co`.

## External Validation

Rules can optionally include an external HTTP call. When the expression matches, the external call is made before returning the value:

```json
{
  "expression": "user.plan == \"enterprise\"",
  "value": true,
  "externalValidation": {
    "url": "https://crm.example.com/api/eligibility",
    "method": "POST",
    "authType": "bearer",
    "authSecretRef": "CRM_API_TOKEN",
    "timeout": 3,
    "cacheTTL": 300,
    "failMode": "open",
    "requestMapping": [
      { "source": "user.id", "target": "userId" },
      { "source": "tenant", "target": "tenantCode" }
    ],
    "responseCondition": "response.eligible == true"
  }
}
```

### External Call Behavior

- **Circuit breaker:** Per-endpoint circuit breaker (gobreaker). Opens after consecutive failures.
- **Caching:** Responses are cached in Redis by SHA-256 of request parameters.
- **Fail modes:**
  - `open` -- If the call fails, the rule still matches (optimistic).
  - `closed` -- If the call fails, the rule does not match (pessimistic).

## Security Limits

Expressions are validated before compilation:

| Check           | Limit                                                                    |
| --------------- | ------------------------------------------------------------------------ |
| Deny-list       | Blocks `exec`, `system`, `import`, `require`, `__proto__`, `constructor` |
| AST depth       | Max 10 levels                                                            |
| AST nodes       | Max 100 nodes                                                            |
| inSegment calls | Max 5 per expression                                                     |
| String literals | Max 1,000 characters each                                                |

Expressions that violate these limits are rejected at creation time with a descriptive error.

## Expression Validation Endpoint

Use `POST /features/admin/expression/validate` to check an expression before saving:

```bash
curl -X POST http://localhost:8080/features/admin/expression/validate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"expression": "user.plan == \"enterprise\" && inSegment(\"beta\")"}'
```

## Expression Testing Endpoint

Use `POST /features/admin/expression/test` to evaluate an expression against sample data:

```bash
curl -X POST http://localhost:8080/features/admin/expression/test \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "expression": "user.plan == \"enterprise\"",
    "context": { "user": { "plan": "enterprise" } }
  }'
```

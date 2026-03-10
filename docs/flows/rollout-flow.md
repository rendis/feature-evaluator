# Percentage Rollouts Flow

[< Back to Overview](../UI-FLOW.md)

---

## Overview

Percentage rollouts allow admins to limit a rule to a fraction of users. This feature is integrated directly into the rule create/edit form as an additional section, and rollout percentages are displayed in the rule list on the feature detail page.

---

## Integration Points

Rollouts are not a standalone route. They appear in two places:

1. **Rule Form** (`/features/:key/rules/new` and `/features/:key/rules/:ruleId/edit`) -- Section 4 "Gradual Rollout"
2. **Rule List** on the feature detail page (`/features/:key`) -- Percentage badge per rule row

---

## Rule Form -- Rollout Section

Located as the 4th `FormSection` card inside `RuleForm` (between Scope and External Validation).

**Component**: `console/src/components/rules/rule-form-rollout.tsx` (`RolloutSection`)

| Element | Type | Action |
|---|---|---|
| "Limit rollout" checkbox | Checkbox | Enables/disables rollout. When unchecked, `rolloutPercentage` is `null` (no limit). When checked, defaults to 100% |
| Description text | Display | "Apply only to a percentage of users" |
| Slider | Slider | 0-100 range, step 1. Visible only when rollout enabled |
| Number input | Input | 0-100, synced with slider. `w-20`, centered text |
| "%" suffix | Display | Next to number input |
| Helper text | Display | "All users will receive this rule" when 100%, or "X% of users will receive this rule" for partial rollout |

### State Management

- `rolloutPercentage` state in `RuleForm`: `number | null`
  - `null` = rollout disabled (all users)
  - `number` (0-100) = rollout enabled
- Checkbox toggle: `null` <-> `100`
- Slider and input are synchronized bidirectionally
- Input validation: integer, 0-100 range (invalid input ignored)
- Dirty detection: compares current `rolloutPercentage` against initial value from `rule.rolloutPercentage`

### API Payload

The `rolloutPercentage` field is included in the create/update rule payload:
- `null` when rollout is disabled
- `number` (0-100) when enabled

---

## Rule List -- Percentage Badge

**Component**: `console/src/components/rules/rule-list.tsx` (`RuleRow`)

| Element | Type | Condition |
|---|---|---|
| Percentage badge | `Badge variant="outline"` | Shown only when `rule.rolloutPercentage != null && rule.rolloutPercentage < 100` |

The badge displays the rollout percentage (e.g., "75%") in monospace font. If the rule has no rollout or is at 100%, no badge is shown.

---

## User Flow

1. Admin navigates to create or edit a rule
2. In the "Gradual Rollout" section, the checkbox is unchecked by default (no rollout limit)
3. Admin checks the "Limit rollout" checkbox -- slider and input appear, defaulting to 100%
4. Admin adjusts the slider or types a percentage value
5. The helper text updates dynamically to reflect the chosen percentage
6. On save, the `rolloutPercentage` value is included in the API payload
7. On the feature detail page, the rule list shows a percentage badge next to any rule with a partial rollout (<100%)

---

## i18n

**Namespace**: `rules` (existing, no new namespace needed)

Keys used (with Spanish defaults inline via `defaultValue`):
- `rollout.enableLabel` -- "Limitar despliegue"
- `rollout.enableDescription` -- "Aplicar solo a un porcentaje de usuarios"
- `rollout.allUsers` -- "Todos los usuarios recibiran esta regla"
- `rollout.partial` -- "{{percentage}}% de los usuarios recibiran esta regla"

---

## Component Files

| File | Purpose |
|---|---|
| `console/src/components/rules/rule-form-rollout.tsx` | `RolloutSection` -- checkbox + slider + number input |
| `console/src/components/rules/rule-form.tsx` | `RuleForm` -- integrates `RolloutSection` as section 4 |
| `console/src/components/rules/rule-list.tsx` | `RuleRow` -- displays percentage badge |
| `console/src/api/types.ts` | `Rule.rolloutPercentage?: number | null` |

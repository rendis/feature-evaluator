import type { CreateRuleRequest } from '@/api/rules';
import type { ExternalApiBinding, Feature, Rule, SourceBindings, ValueType } from '@/api/types';
import type { TFunction } from 'i18next';

export interface RuleDraft {
  name: string;
  priority: number;
  enabled: boolean;
  value: string;
  expression: string;
  metadata: Record<string, unknown>;
  sourceBindings: SourceBindings;
  externalApiBindings: ExternalApiBinding[];
  rolloutEnabled: boolean;
  rolloutLimit: number;
}

// --- Draft lifecycle ---

export function createDraft(rule: Rule | undefined, feature: Feature, nextPriority: number): RuleDraft {
  return {
    name: rule?.name ?? '',
    priority: rule?.priority ?? nextPriority,
    enabled: rule?.enabled ?? true,
    value: defaultRuleValue(rule, feature.valueType),
    expression: rule?.expression ?? '',
    metadata: deepClone(rule?.metadata ?? {}),
    sourceBindings: deepClone(rule?.sourceBindings ?? { segments: [] }),
    externalApiBindings: deepClone(rule?.externalApiBindings ?? []),
    rolloutEnabled: rule?.rolloutPercentage != null,
    rolloutLimit: rule?.rolloutPercentage ?? 100,
  };
}

export function createClonedDraft(
  rule: Rule,
  feature: Feature,
  nextPriority: number,
  clonedName: string,
): RuleDraft {
  const draft = createDraft(rule, feature, nextPriority);

  return {
    ...draft,
    name: clonedName,
    priority: nextPriority,
  };
}

export function cloneRuleDraft(draft: RuleDraft): RuleDraft {
  return {
    ...draft,
    metadata: deepClone(draft.metadata),
    sourceBindings: deepClone(draft.sourceBindings),
    externalApiBindings: deepClone(draft.externalApiBindings),
  };
}

export function serializeSnapshot(draft: RuleDraft): string {
  return JSON.stringify({
    name: draft.name,
    priority: draft.priority,
    enabled: draft.enabled,
    value: draft.value,
    expression: draft.expression,
    metadata: draft.metadata,
    sourceBindings: draft.sourceBindings,
    externalApiBindings: draft.externalApiBindings,
    rolloutEnabled: draft.rolloutEnabled,
    rolloutLimit: draft.rolloutLimit,
  });
}

// --- Validation ---

export function validateDraft(draft: RuleDraft, t: TFunction<'rules'>): string | null {
  if (!draft.name.trim()) return t('form.nameRequired');
  if (!draft.value.trim()) return t('form.nameRequired');
  if (!draft.expression.trim()) {
    return t('form.invalidExpression', {
      defaultValue: 'Completa la condicion antes de guardar la regla.',
    });
  }
  return null;
}

// --- Payload builder ---

export function buildPayload(draft: RuleDraft, valueType: ValueType): CreateRuleRequest {
  return {
    name: draft.name,
    priority: draft.priority,
    enabled: draft.enabled,
    expression: draft.expression,
    value: parseRuleValue(draft.value, valueType),
    sourceBindings: draft.sourceBindings,
    externalApiBindings: draft.externalApiBindings,
    metadata: draft.metadata,
    rolloutPercentage: draft.rolloutEnabled ? draft.rolloutLimit : null,
  };
}

export function summarizeDraft(draft: RuleDraft, t: TFunction<'rules'>): string {
  const parts: string[] = [];
  parts.push(`${t('fields.name')}: ${draft.name || '—'}`);
  parts.push(`${t('fields.priority')}: ${draft.priority}`);
  parts.push(`${t('fields.enabled')}: ${draft.enabled ? '✓' : '✗'}`);
  if (draft.expression) {
    parts.push(`${t('fields.expression')}: ${draft.expression.slice(0, 40)}${draft.expression.length > 40 ? '…' : ''}`);
  }
  if (draft.rolloutEnabled) {
    parts.push(`${t('form.rolloutSection')}: ${draft.rolloutLimit}%`);
  }
  if (draft.externalApiBindings.length > 0) {
    parts.push(`APIs: ${draft.externalApiBindings.map((b) => b.externalApiKey).join(', ')}`);
  }
  return parts.join(' · ');
}

// --- Helpers ---

function parseRuleValue(raw: string, vt: ValueType): unknown {
  if (vt === 'boolean') return raw === 'true';
  if (vt === 'number') return Number(raw);
  if (vt === 'json') return JSON.parse(raw) as unknown;
  return raw;
}

function stringifyRuleValue(val: unknown, vt: ValueType): string {
  if (vt === 'json') return JSON.stringify(val, null, 2);
  return String(val ?? '');
}

function defaultRuleValue(rule: Rule | undefined, vt: ValueType): string {
  if (rule) return stringifyRuleValue(rule.value, vt);
  return vt === 'boolean' ? 'true' : '';
}

function deepClone<T>(value: T): T {
  if (typeof globalThis.structuredClone === 'function') {
    return globalThis.structuredClone(value);
  }

  return JSON.parse(JSON.stringify(value)) as T;
}

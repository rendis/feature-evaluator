import type { CreateFeatureRequest, UpdateFeatureRequest } from '@/api/features';
import type { Feature, FeatureAccessPolicy, InputContract, InputHeader, Tag, ValueType } from '@/api/types';
import type { TFunction } from 'i18next';

import { RESOURCE_KEY_PATTERN, slugifyResourceKey } from '@/lib/resource-key';

export interface FeatureDraft {
  name: string;
  key: string;
  description: string;
  valueType: ValueType;
  defaultValue: string;
  evalCacheEnabled: boolean;
  evalCacheTTLSeconds: string;
  accessPolicy: FeatureAccessPolicy;
  authProfileKey: string;
  headers: InputHeader[];
  requestBodyExampleText: string;
  tags: Tag[];
  activeFrom: string;
  activeUntil: string;
  environments: string[];
  trialUntil: string;
  trialValue: string;
}

// --- Draft lifecycle ---

export function createDraft(feature?: Feature): FeatureDraft {
  const ic = buildInitialInputContract(feature);
  return {
    name: feature?.name ?? '',
    key: feature?.key ?? '',
    description: feature?.description ?? '',
    valueType: feature?.valueType ?? 'boolean',
    defaultValue: feature ? stringifyDefault(feature.defaultValue, feature.valueType) : 'false',
    evalCacheEnabled: feature?.evalCacheEnabled ?? (feature?.evalCacheTTLSeconds ?? 0) > 0,
    evalCacheTTLSeconds: feature?.evalCacheTTLSeconds ? String(feature.evalCacheTTLSeconds) : '300',
    accessPolicy: feature?.accessPolicy ?? 'required',
    authProfileKey: feature?.authProfileKey ?? '',
    headers: ic.headers,
    requestBodyExampleText: stringifyRequestBodyExample(ic),
    tags: feature?.tags ?? [],
    activeFrom: utcToLocal(feature?.activeFrom),
    activeUntil: utcToLocal(feature?.activeUntil),
    environments: feature?.environments ?? [],
    trialUntil: utcToLocal(feature?.trialUntil),
    trialValue:
      feature?.trialValue !== undefined && feature?.trialValue !== null
        ? stringifyDefault(feature.trialValue, feature.valueType)
        : '',
  };
}

export function serializeSnapshot(draft: FeatureDraft): string {
  return JSON.stringify({
    name: draft.name,
    description: draft.description,
    valueType: draft.valueType,
    defaultValue: draft.defaultValue,
    evalCacheEnabled: draft.evalCacheEnabled,
    evalCacheTTLSeconds: draft.evalCacheTTLSeconds,
    accessPolicy: draft.accessPolicy,
    authProfileKey: draft.authProfileKey,
    headers: draft.headers,
    requestBodyExampleText: draft.requestBodyExampleText,
    tags: draft.tags.map((t) => t.key).sort(),
    activeFrom: draft.activeFrom,
    activeUntil: draft.activeUntil,
    environments: [...draft.environments].sort(),
    trialUntil: draft.trialUntil,
    trialValue: draft.trialValue,
  });
}

// --- Validation ---

export function validateDraft(draft: FeatureDraft, t: TFunction<'features'>): string | null {
  if (!draft.name.trim()) return t('form.nameRequired');

  const key = slugifyResourceKey(draft.name);
  if (!draft.key && !key) return t('form.keyInvalid');
  if (draft.key && !RESOURCE_KEY_PATTERN.test(draft.key)) return t('form.keyInvalid');

  if (!draft.defaultValue.trim()) return t('form.defaultValueRequired');

  if (draft.accessPolicy === 'required' && !draft.authProfileKey.trim()) {
    return t('form.authProfileRequired');
  }

  const activeFrom = localToUtc(draft.activeFrom);
  const activeUntil = localToUtc(draft.activeUntil);
  if (activeFrom && activeUntil && new Date(activeFrom) >= new Date(activeUntil)) {
    return t('form.dateRangeError');
  }

  const bodyResult = parseRequestBodyExample(draft.requestBodyExampleText);
  if (bodyResult.error) return bodyResult.error;

  return null;
}

// --- Payload builders ---

export function buildCreatePayload(draft: FeatureDraft): CreateFeatureRequest {
  const normalizedIC = buildNormalizedInputContract(draft);
  const activeFrom = localToUtc(draft.activeFrom);
  const activeUntil = localToUtc(draft.activeUntil);
  const tagKeys = draft.tags.map((t) => t.key);
  const environments = draft.environments.length > 0 ? draft.environments : undefined;

  return {
    key: draft.key || slugifyResourceKey(draft.name),
    name: draft.name,
    description: draft.description,
    valueType: draft.valueType,
    defaultValue: parseDefaultValue(draft.defaultValue, draft.valueType),
    evalCacheEnabled: draft.evalCacheEnabled,
    evalCacheTTLSeconds: draft.evalCacheEnabled ? normalizePositiveInt(draft.evalCacheTTLSeconds, 300) : 0,
    accessPolicy: draft.accessPolicy,
    authProfileKey: draft.accessPolicy === 'public' ? '' : draft.authProfileKey,
    inputContract: normalizedIC,
    tags: tagKeys,
    activeFrom,
    activeUntil,
    environments,
    trialUntil: localToUtc(draft.trialUntil),
    trialValue: draft.trialValue ? parseDefaultValue(draft.trialValue, draft.valueType) : undefined,
  };
}

export function buildUpdatePayload(draft: FeatureDraft): UpdateFeatureRequest {
  const normalizedIC = buildNormalizedInputContract(draft);
  const activeFrom = localToUtc(draft.activeFrom);
  const activeUntil = localToUtc(draft.activeUntil);
  const tagKeys = draft.tags.map((t) => t.key);
  const environments = draft.environments.length > 0 ? draft.environments : undefined;

  return {
    name: draft.name,
    description: draft.description,
    defaultValue: parseDefaultValue(draft.defaultValue, draft.valueType),
    evalCacheEnabled: draft.evalCacheEnabled,
    evalCacheTTLSeconds: draft.evalCacheEnabled ? normalizePositiveInt(draft.evalCacheTTLSeconds, 300) : 0,
    accessPolicy: draft.accessPolicy,
    authProfileKey: draft.accessPolicy === 'public' ? '' : draft.authProfileKey,
    inputContract: normalizedIC,
    tags: tagKeys,
    activeFrom,
    activeUntil,
    environments,
    trialUntil: localToUtc(draft.trialUntil),
    trialValue: draft.trialValue ? parseDefaultValue(draft.trialValue, draft.valueType) : undefined,
  };
}

export function summarizeDraft(draft: FeatureDraft, t: TFunction<'features'>): string {
  const parts: string[] = [];
  parts.push(`${t('fields.name')}: ${draft.name || '—'}`);
  parts.push(`${t('fields.valueType')}: ${t(`valueTypes.${draft.valueType}`)}`);
  parts.push(`${t('fields.accessPolicy')}: ${t(`accessPolicy.${draft.accessPolicy}`)}`);
  if (draft.headers.length > 0) {
    parts.push(`Headers: ${draft.headers.length}`);
  }
  if (draft.tags.length > 0) {
    parts.push(`Tags: ${draft.tags.map((tg) => tg.name).join(', ')}`);
  }
  if (draft.environments.length > 0) {
    parts.push(`${t('fields.environments')}: ${draft.environments.join(', ')}`);
  }
  parts.push(
    `${t('cache.title', { defaultValue: 'Cache' })}: ${draft.evalCacheEnabled ? `${draft.evalCacheTTLSeconds || '300'}s` : t('cache.disabled', { defaultValue: 'disabled' })}`,
  );
  return parts.join(' · ');
}

// --- Helpers ---

export function parseDefaultValue(raw: string, vt: ValueType): unknown {
  if (vt === 'boolean') return raw === 'true';
  if (vt === 'number') return Number(raw);
  if (vt === 'json') return JSON.parse(raw) as unknown;
  return raw;
}

function normalizePositiveInt(raw: string, fallback: number): number {
  const parsed = Number.parseInt(raw, 10);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return fallback;
  }
  return parsed;
}

export function stringifyDefault(val: unknown, vt: ValueType): string {
  if (vt === 'json') return JSON.stringify(val, null, 2);
  return String(val ?? '');
}

export function utcToLocal(utc: string | null | undefined): string {
  if (!utc) return '';
  const d = new Date(utc);
  if (isNaN(d.getTime())) return '';
  const offset = d.getTimezoneOffset();
  const local = new Date(d.getTime() - offset * 60000);
  return local.toISOString().slice(0, 16);
}

export function localToUtc(local: string): string | null {
  if (!local) return null;
  return new Date(local).toISOString();
}

function buildInitialInputContract(feature: Feature | undefined): InputContract {
  return {
    headers: feature?.inputContract?.headers ?? [],
    requestBodyExample: feature?.inputContract?.requestBodyExample ?? {},
    requestBodySchema: feature?.inputContract?.requestBodySchema,
  };
}

function stringifyRequestBodyExample(inputContract: InputContract): string {
  if (
    !inputContract.requestBodyExample ||
    Object.keys(inputContract.requestBodyExample).length === 0
  ) {
    return '{\n  \n}';
  }
  return JSON.stringify(inputContract.requestBodyExample, null, 2);
}

export function parseRequestBodyExample(text: string): {
  value: Record<string, unknown> | undefined;
  error: string | null;
} {
  const trimmed = text.trim();
  if (!trimmed) return { value: undefined, error: null };
  try {
    const parsed = JSON.parse(trimmed) as unknown;
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return { value: undefined, error: 'El ejemplo de request body debe ser un objeto JSON.' };
    }
    return { value: parsed as Record<string, unknown>, error: null };
  } catch {
    return { value: undefined, error: 'El ejemplo de request body no es JSON valido.' };
  }
}

function buildNormalizedInputContract(draft: FeatureDraft): InputContract {
  const bodyResult = parseRequestBodyExample(draft.requestBodyExampleText);
  return {
    headers: draft.headers
      .map((h) => ({
        ...h,
        headerName: h.headerName.trim(),
        expressionKey: h.expressionKey.trim(),
        label: h.label.trim(),
        description: h.description?.trim() ?? '',
      }))
      .filter((h) => h.headerName.length > 0),
    requestBodyExample: bodyResult.value,
  };
}

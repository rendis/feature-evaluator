import type { AuthProfile, AuthProfileType } from '@/api/types';
import type { TFunction } from 'i18next';

import { KeyRound, ShieldCheck, Sparkles } from 'lucide-react';

export type UseCase = AuthProfileType;
export type MappingSourceType = 'request_header' | 'request_body';
export type MappingTargetType = 'header' | 'body';

export interface StaticHeader {
  id: string;
  key: string;
  value: string;
}
export type SuccessRuleType = 'any_2xx' | 'status' | 'json_field' | 'response_header' | 'text_contains';

export interface MappingRow {
  id: string;
  sourceType: MappingSourceType;
  sourceValue: string;
  sourceStripPrefix: string;
  targetType: MappingTargetType;
  targetValue: string;
}

export interface SuccessRuleDraft {
  type: SuccessRuleType;
  status: string;
  path: string;
  value: string;
  header: string;
}

export interface DraftState {
  name: string;
  key: string;
  active: boolean;
  type: UseCase;
  cacheEnabled: boolean;
  cacheTTLSeconds: string;
  apiKey: {
    location: 'header' | 'query';
    name: string;
    prefix: string;
    secret: string;
  };
  oidc: {
    issuer: string;
    audience: string;
  };
  custom: {
    url: string;
    method: 'GET' | 'POST';
    timeout: string;
    outboundAuthHeaderName: string;
    outboundApiKey: string;
    requestHeaders: StaticHeader[];
    headerMappings: MappingRow[];
    bodyMappings: MappingRow[];
    successRule: SuccessRuleDraft;
  };
  test: {
    bearerToken: string;
    headers: string;
    query: string;
    body: string;
  };
}

export function serializeSnapshot(draft: DraftState) {
  const stripIds = <T extends { id: string }>(rows: T[]) =>
    rows.map(({ id: _, ...rest }) => rest);
  return JSON.stringify({
    name: draft.name,
    active: draft.active,
    type: draft.type,
    cacheEnabled: draft.cacheEnabled,
    cacheTTLSeconds: draft.cacheTTLSeconds,
    apiKey: draft.apiKey,
    oidc: draft.oidc,
    custom: {
      ...draft.custom,
      outboundApiKey: draft.custom.outboundApiKey ? '***' : '',
      requestHeaders: stripIds(draft.custom.requestHeaders),
      headerMappings: stripIds(draft.custom.headerMappings),
      bodyMappings: stripIds(draft.custom.bodyMappings),
    },
  });
}

export function createDraft(profile?: AuthProfile | null): DraftState {
  const type = profile?.type ?? 'api_key';
  const config = profile?.config ?? {};
  const headers = parseMappingRows((config.headers as unknown[]) ?? [], 'header');
  const body = parseMappingRows((config.body as unknown[]) ?? [], 'body');
  const requestHeaders = parseStaticHeaders((config.requestHeaders as unknown[]) ?? []);

  return {
    name: profile?.name ?? '',
    key: profile?.key ?? '',
    active: profile?.active ?? true,
    type,
    cacheEnabled: (profile?.cacheTTLSeconds ?? 0) > 0,
    cacheTTLSeconds: profile?.cacheTTLSeconds ? String(profile.cacheTTLSeconds) : '300',
    apiKey: {
      location: (config.location as 'header' | 'query') ?? 'header',
      name: (config.name as string) ?? 'X-Api-Key',
      prefix: (config.prefix as string) ?? '',
      secret: '',
    },
    oidc: {
      issuer: (config.issuer as string) ?? '',
      audience: (config.audience as string) ?? '',
    },
    custom: {
      url: (config.url as string) ?? '',
      method: (config.method as 'GET' | 'POST') ?? 'POST',
      timeout: config.timeout ? String(config.timeout) : '5000',
      outboundAuthHeaderName: (config.outboundAuthHeaderName as string) ?? '',
      outboundApiKey: '',
      requestHeaders,
      headerMappings: headers,
      bodyMappings: body,
      successRule: parseSuccessRule(config.successRule as Record<string, unknown> | undefined),
    },
    test: {
      bearerToken: defaultTestBearerToken(type),
      headers: JSON.stringify(defaultTestHeaders(type, config), null, 2),
      query: JSON.stringify(defaultTestQuery(type, config), null, 2),
      body: JSON.stringify(defaultTestBody(type), null, 2),
    },
  };
}

export function slugifyName(value: string) {
  return value
    .normalize('NFKD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
    .replace(/_+/g, '_');
}

export function buildPayload(draft: DraftState, key: string) {
  const cacheTTLSeconds =
    draft.cacheEnabled && draft.type !== 'api_key' && draft.type !== 'oidc_standard'
      ? normalizePositiveInt(draft.cacheTTLSeconds, 300)
      : 0;

  if (draft.type === 'api_key') {
    return {
      key,
      name: draft.name.trim(),
      active: draft.active,
      type: draft.type,
      cacheTTLSeconds,
      config: {
        location: draft.apiKey.location,
        name: draft.apiKey.name.trim(),
        prefix: draft.apiKey.prefix.trim(),
      },
      secretPayload: draft.apiKey.secret.trim()
        ? { apiKey: draft.apiKey.secret.trim() }
        : undefined,
    };
  }

  if (draft.type === 'oidc_standard') {
    return {
      key,
      name: draft.name.trim(),
      active: draft.active,
      type: draft.type,
      cacheTTLSeconds: 0,
      config: {
        issuer: draft.oidc.issuer.trim(),
        audience: draft.oidc.audience.trim(),
      },
    };
  }

  return {
    key,
    name: draft.name.trim(),
    active: draft.active,
    type: draft.type,
    cacheTTLSeconds,
    config: {
      url: draft.custom.url.trim(),
      method: draft.custom.method,
      timeout: normalizePositiveInt(draft.custom.timeout, 5000),
      outboundAuthHeaderName: draft.custom.outboundAuthHeaderName.trim(),
      requestHeaders: draft.custom.requestHeaders
        .filter((h) => h.key.trim() !== '')
        .map((h) => ({ key: h.key.trim(), value: h.value.trim() })),
      headers: draft.custom.headerMappings.filter(isMappingComplete).map(toMappingConfig),
      body: draft.custom.bodyMappings.filter(isMappingComplete).map(toMappingConfig),
      successRule: buildSuccessRulePayload(draft.custom.successRule),
    },
    secretPayload: draft.custom.outboundApiKey.trim()
      ? { outboundApiKey: draft.custom.outboundApiKey.trim() }
      : undefined,
  };
}

export function validateDraft(draft: DraftState, t: TFunction<'settings'>) {
  if (!draft.name.trim()) {
    return t('authProfiles.nameRequired');
  }
  if (!slugifyName(draft.name)) {
    return t('authProfiles.validation.invalidGeneratedKey');
  }
  if (draft.type === 'api_key') {
    if (!draft.apiKey.name.trim()) return t('authProfiles.validation.apiKeyNameRequired');
    if (!draft.apiKey.secret.trim() && draft.active) {
      return t('authProfiles.validation.apiKeySecretRequired');
    }
  }
  if (draft.type === 'oidc_standard' && !draft.oidc.issuer.trim()) {
    return t('authProfiles.validation.oidcIssuerRequired');
  }
  if (draft.type === 'oidc_standard' && !draft.oidc.audience.trim()) {
    return t('authProfiles.validation.oidcAudienceRequired');
  }
  if (draft.type === 'custom') {
    if (!draft.custom.url.trim()) return t('authProfiles.validation.customUrlRequired');
    if (
      !draft.custom.headerMappings.some(isMappingComplete) &&
      !draft.custom.bodyMappings.some(isMappingComplete)
    ) {
      return t('authProfiles.validation.mappingRequired');
    }
  }
  return null;
}

export function shouldReplaceSecrets(draft: DraftState) {
  if (draft.type === 'api_key') return draft.apiKey.secret.trim().length > 0;
  if (draft.type === 'oidc_standard') return false;
  return draft.custom.outboundApiKey.trim().length > 0;
}

export function summarizeDraft(draft: DraftState, t: TFunction<'settings'>) {
  switch (draft.type) {
    case 'api_key':
      return t('authProfiles.summary.apiKey', {
        location: t(`authProfiles.summary.location.${draft.apiKey.location}`),
        name: draft.apiKey.name || t('authProfiles.summary.unnamed'),
      });
    case 'oidc_standard':
      return t('authProfiles.summary.oidc', {
        issuer: draft.oidc.issuer || t('authProfiles.summary.configuredIssuer'),
        audience: draft.oidc.audience || t('authProfiles.summary.configuredAudience'),
      });
    case 'custom':
      return t('authProfiles.summary.custom', {
        url: draft.custom.url || t('authProfiles.summary.configuredUrl'),
        staticHeaderCount: draft.custom.requestHeaders.filter((h) => h.key.trim() !== '').length,
        headerCount: draft.custom.headerMappings.filter(isMappingComplete).length,
        bodyCount: draft.custom.bodyMappings.filter(isMappingComplete).length,
      });
  }
}

export function parseObjectJson<T>(value: string): Record<string, T> {
  const trimmed = value.trim();
  if (!trimmed) return {};
  const parsed = JSON.parse(trimmed) as unknown;
  if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
    throw new Error('authProfiles.validation.jsonObjectRequired');
  }
  return parsed as Record<string, T>;
}

export function createStaticHeader(): StaticHeader {
  return { id: crypto.randomUUID(), key: '', value: '' };
}

export function createMappingRow(targetType: MappingTargetType): MappingRow {
  return {
    id: crypto.randomUUID(),
    sourceType: 'request_header',
    sourceValue: '',
    sourceStripPrefix: '',
    targetType,
    targetValue: '',
  };
}

export function combineMappingRows(headerMappings: MappingRow[], bodyMappings: MappingRow[]) {
  return [...headerMappings, ...bodyMappings];
}

export function splitMappingRows(rows: MappingRow[]) {
  return {
    headerMappings: rows.filter((row) => row.targetType === 'header'),
    bodyMappings: rows.filter((row) => row.targetType === 'body'),
  };
}

export function applyUseCase(draft: DraftState, type: UseCase, hasSecret: boolean) {
  const next = { ...draft, type };
  if (type === 'api_key' || type === 'oidc_standard') {
    next.cacheEnabled = false;
  }
  if (type === 'api_key') {
    next.test = {
      bearerToken: '',
      headers: JSON.stringify(
        draft.apiKey.location === 'header'
          ? { [draft.apiKey.name || 'X-Api-Key']: 'sample-key' }
          : {},
        null,
        2,
      ),
      query: JSON.stringify(
        draft.apiKey.location === 'query' ? { [draft.apiKey.name || 'api_key']: 'sample-key' } : {},
        null,
        2,
      ),
      body: '{}',
    };
  }
  if (type === 'oidc_standard') {
    next.test = {
      bearerToken: 'sample-token',
      headers: '{}',
      query: '{}',
      body: '{}',
    };
  }
  if (type === 'custom') {
    next.custom = {
      ...next.custom,
      headerMappings: next.custom.headerMappings,
      bodyMappings: next.custom.bodyMappings,
      outboundApiKey: hasSecret ? '' : next.custom.outboundApiKey,
    };
  }
  return next;
}

export function applyTestPreset(draft: DraftState, preset: 'bearer' | 'api_key' | 'custom_header') {
  if (preset === 'bearer') {
    return {
      ...draft,
      test: {
        bearerToken: draft.type === 'oidc_standard' ? 'sample-token' : '',
        headers: JSON.stringify({ Authorization: 'Bearer sample-token' }, null, 2),
        query: '{}',
        body: '{}',
      },
    };
  }
  if (preset === 'api_key') {
    const headers =
      draft.type === 'api_key' && draft.apiKey.location === 'header'
        ? { [draft.apiKey.name || 'X-Api-Key']: 'sample-key' }
        : {};
    const query =
      draft.type === 'api_key' && draft.apiKey.location === 'query'
        ? { [draft.apiKey.name || 'api_key']: 'sample-key' }
        : {};
    return {
      ...draft,
      test: {
        bearerToken: '',
        headers: JSON.stringify(headers, null, 2),
        query: JSON.stringify(query, null, 2),
        body: '{}',
      },
    };
  }
  return {
    ...draft,
    test: {
      bearerToken: '',
      headers: JSON.stringify({ 'X-Session-Token': 'sample-token' }, null, 2),
      query: '{}',
      body: JSON.stringify({ session: { token: 'sample-token' } }, null, 2),
    },
  };
}

const useCaseCardsBase = [
  { type: 'api_key', icon: KeyRound },
  { type: 'oidc_standard', icon: ShieldCheck },
  { type: 'custom', icon: Sparkles },
] as const;

export type UseCaseCard = ReturnType<typeof getUseCaseCards>[number];

export function getUseCaseCards(t: TFunction<'settings'>) {
  return useCaseCardsBase.map((option) => ({
    ...option,
    title: t(`authProfiles.typeOptions.${option.type}`),
    description: t(`authProfiles.useCaseDescriptions.${option.type}`),
  }));
}

// --- internal helpers ---

function normalizePositiveInt(value: string, fallback: number) {
  const parsed = Number.parseInt(value, 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

function isMappingComplete(row: MappingRow) {
  return row.sourceValue.trim() !== '' && row.targetValue.trim() !== '';
}

function toMappingConfig(row: MappingRow) {
  return {
    source:
      row.sourceType === 'request_header'
        ? {
            type: 'request_header',
            name: row.sourceValue.trim(),
            ...(row.sourceStripPrefix.trim() ? { stripPrefix: row.sourceStripPrefix } : {}),
          }
        : { type: 'request_body', path: row.sourceValue.trim() },
    target:
      row.targetType === 'header'
        ? { type: 'header', name: row.targetValue.trim() }
        : { type: 'body', path: row.targetValue.trim() },
  };
}

function parseMappingRows(raw: unknown[], fallbackTarget: MappingTargetType) {
  return raw
    .flatMap((item) => {
      if (!item || typeof item !== 'object') return [];
      const mapping = item as Record<string, unknown>;
      const source =
        typeof mapping.source === 'object' && mapping.source !== null
          ? (mapping.source as Record<string, unknown>)
          : {};
      const target =
        typeof mapping.target === 'object' && mapping.target !== null
          ? (mapping.target as Record<string, unknown>)
          : {};
      return [
        {
          id: crypto.randomUUID(),
          sourceType: source.type === 'request_body' ? 'request_body' : 'request_header',
          sourceValue: String(source.name ?? source.path ?? ''),
          sourceStripPrefix: String(source.stripPrefix ?? ''),
          targetType: target.type === 'body' ? 'body' : 'header',
          targetValue: String(target.name ?? target.path ?? ''),
        } satisfies MappingRow,
      ];
    })
    .map((row) => ({ ...row, targetType: row.targetType || fallbackTarget }));
}

function parseSuccessRule(raw?: Record<string, unknown>): SuccessRuleDraft {
  if (!raw || !raw.type) {
    return { type: 'any_2xx', status: '200', path: 'valid', value: 'true', header: 'x-decision' };
  }
  return {
    type: raw.type as SuccessRuleType,
    status: String(raw.status ?? '200'),
    path: String(raw.path ?? 'valid'),
    value: String(raw.value ?? 'true'),
    header: String(raw.header ?? 'x-decision'),
  };
}

function buildSuccessRulePayload(rule: SuccessRuleDraft) {
  switch (rule.type) {
    case 'status':
      return { type: 'status', status: normalizePositiveInt(rule.status, 200) };
    case 'json_field':
      return {
        type: 'json_field',
        path: rule.path.trim(),
        operator: 'equals',
        value: rule.value.trim(),
      };
    case 'response_header':
      return { type: 'response_header', header: rule.header.trim(), value: rule.value.trim() };
    case 'text_contains':
      return { type: 'text_contains', value: rule.value.trim() };
    default:
      return { type: 'any_2xx' };
  }
}

function parseStaticHeaders(raw: unknown[]): StaticHeader[] {
  return raw.flatMap((item) => {
    if (!item || typeof item !== 'object') return [];
    const entry = item as Record<string, unknown>;
    const key = String(entry.key ?? '').trim();
    if (!key) return [];
    return [{ id: crypto.randomUUID(), key, value: String(entry.value ?? '').trim() }];
  });
}

function defaultTestHeaders(type: UseCase, config: Record<string, unknown>) {
  if (type === 'api_key' && config.location === 'header') {
    return { [(config.name as string) || 'X-Api-Key']: 'sample-key' };
  }
  return { Authorization: 'Bearer sample-token' };
}

function defaultTestBearerToken(type: UseCase) {
  if (type === 'oidc_standard') return 'sample-token';
  return '';
}

function defaultTestQuery(type: UseCase, config: Record<string, unknown>) {
  if (type === 'api_key' && config.location === 'query') {
    return { [(config.name as string) || 'api_key']: 'sample-key' };
  }
  return {};
}

function defaultTestBody(type: UseCase) {
  if (type === 'custom') return { session: { token: 'sample-token' } };
  return {};
}

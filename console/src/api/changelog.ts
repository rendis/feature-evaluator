import { api } from './client';

import type { PaginatedResponse } from './types';

export interface FieldChange {
  field: string;
  oldValue: unknown;
  newValue: unknown;
}

export interface ChangeEntry {
  id: string;
  entityType: string;
  entityKey: string;
  parentKey?: string;
  action: string;
  actor: string;
  actorType: string;
  fieldChanges?: FieldChange[];
  metadata?: Record<string, unknown>;
  createdAt: string;
}

export interface ListChangelogParams {
  page?: number;
  pageSize?: number;
  entityType?: string;
  entityKey?: string;
  actor?: string;
  action?: string;
  from?: string;
  to?: string;
}

const BASE = '/admin/changelog';

function buildQuery(params: ListChangelogParams): string {
  const p = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') p.set(k, String(v));
  }
  const qs = p.toString();
  return qs ? `?${qs}` : '';
}

export const changelogApi = {
  list: (params: ListChangelogParams = {}) =>
    api.get<PaginatedResponse<ChangeEntry>>(`${BASE}${buildQuery(params)}`),
  listByEntity: (entityType: string, entityKey: string, params: Omit<ListChangelogParams, 'entityType' | 'entityKey'> = {}) =>
    api.get<PaginatedResponse<ChangeEntry>>(`${BASE}/${entityType}/${entityKey}${buildQuery(params)}`),
};

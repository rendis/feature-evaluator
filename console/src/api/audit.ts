import { api } from './client';

import type { AuditError, PaginatedResponse } from './types';

const BASE = '/admin/audit/errors';

export interface ListAuditErrorsParams {
  page?: number;
  pageSize?: number;
  featureKey?: string;
  tenantId?: string;
  errorType?: string;
  from?: string;
  to?: string;
}

function buildQuery(params: object): string {
  const p = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') p.set(k, String(v));
  }
  const qs = p.toString();
  return qs ? `?${qs}` : '';
}

export const auditApi = {
  errors: (params: ListAuditErrorsParams = {}) =>
    api.get<PaginatedResponse<AuditError>>(`${BASE}${buildQuery(params)}`),
};

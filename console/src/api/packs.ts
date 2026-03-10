import { api } from './client';

import type {
  ActivatePackRequest,
  CreatePackRequest,
  DeactivatePackRequest,
  Pack,
  PackActivation,
  PaginatedResponse,
  UpdatePackRequest,
} from './types';

const BASE = '/admin/packs';

export interface ListPacksParams {
  page?: number;
  pageSize?: number;
  search?: string;
}

function buildQuery(params: ListPacksParams): string {
  const p = new URLSearchParams();
  if (params.page) p.set('page', String(params.page));
  if (params.pageSize) p.set('pageSize', String(params.pageSize));
  if (params.search) p.set('search', params.search);
  const qs = p.toString();
  return qs ? `?${qs}` : '';
}

export const packsApi = {
  list: (params: ListPacksParams = {}) =>
    api.get<{ data: Pack[] }>(`${BASE}${buildQuery(params)}`).then((res) => {
      const data = res.data;
      const page = params.page ?? 1;
      const pageSize = params.pageSize ?? (data.length || 20);
      const total = data.length;
      const totalPages = pageSize > 0 ? Math.ceil(total / pageSize) : 1;
      return {
        data,
        pagination: { page, pageSize, total, totalPages },
      } as PaginatedResponse<Pack>;
    }),
  getByKey: (key: string) => api.get<Pack>(`${BASE}/${key}`),
  create: (data: CreatePackRequest) => api.post<Pack>(BASE, data),
  update: (key: string, data: UpdatePackRequest) => api.put<Pack>(`${BASE}/${key}`, data),
  remove: (key: string) => api.delete<undefined>(`${BASE}/${key}`),
  toggle: (key: string, enabled: boolean) => api.patch<Pack>(`${BASE}/${key}/toggle`, { enabled }),
  activate: (key: string, data: ActivatePackRequest) =>
    api.post<PackActivation>(`${BASE}/${key}/activate`, data),
  deactivate: (key: string, data: DeactivatePackRequest) =>
    api.delete<undefined>(`${BASE}/${key}/activate`, data),
  listActivations: (key: string) =>
    api.get<{ data: PackActivation[] }>(`${BASE}/${key}/activations`).then((r) => r.data),
  byTarget: (type: string, id: string) =>
    api.get<{ data: Pack[] }>(`${BASE}/by-target?type=${encodeURIComponent(type)}&id=${encodeURIComponent(id)}`).then((r) => r.data),
};

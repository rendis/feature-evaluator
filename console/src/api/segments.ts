import { api } from './client';

import type { PaginatedResponse, Segment, SegmentRecord, SegmentSchema } from './types';

const BASE = '/admin/segments';

export interface ListSegmentsParams {
  page?: number;
  pageSize?: number;
  search?: string;
}

export interface CreateSegmentRequest {
  key: string;
  name: string;
  description?: string;
  metadata?: Record<string, unknown>;
}

export interface UpdateSegmentRequest {
  name?: string;
  description?: string;
  metadata?: Record<string, unknown>;
}

export interface ListSegmentRecordsParams {
  page?: number;
  pageSize?: number;
  q?: string;
}

export interface ImportSegmentDataRequest {
  mode: 'replace';
  sourceType: 'csv' | 'json';
  recordKeyPath: string;
  schema: Record<string, unknown>;
  records: Record<string, unknown>[];
}

export interface ImportResult {
  inserted: number;
  datasetVersion?: string;
  previewFields?: string[];
}

function buildQuery(params: object): string {
  const p = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v !== undefined && v !== '') p.set(k, String(v));
  }
  const qs = p.toString();
  return qs ? `?${qs}` : '';
}

export const segmentsApi = {
  list: (params: ListSegmentsParams = {}) =>
    api.get<PaginatedResponse<Segment>>(`${BASE}${buildQuery(params)}`),
  get: (key: string) => api.get<Segment>(`${BASE}/${key}`),
  getSchema: (key: string) => api.get<SegmentSchema>(`${BASE}/${key}/schema`),
  create: (data: CreateSegmentRequest) => api.post<Segment>(BASE, data),
  update: (key: string, data: UpdateSegmentRequest) => api.put<Segment>(`${BASE}/${key}`, data),
  remove: (key: string) => api.delete<undefined>(`${BASE}/${key}`),
  records: {
    list: (segmentKey: string, params: ListSegmentRecordsParams = {}) =>
      api.get<PaginatedResponse<SegmentRecord>>(`${BASE}/${segmentKey}/records${buildQuery(params)}`),
  },
  data: {
    import: (segmentKey: string, data: ImportSegmentDataRequest) =>
      api.post<ImportResult>(`${BASE}/${segmentKey}/data/import`, data),
  },
};

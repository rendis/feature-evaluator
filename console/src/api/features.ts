import { api } from './client';

import type {
  Feature,
  FeatureAccessPolicy,
  FeatureSummary,
  InputContract,
  PaginatedResponse,
  ToggleFeatureResponse,
} from './types';

const BASE = '/admin/features';

export interface ListFeaturesParams {
  page?: number;
  pageSize?: number;
  search?: string;
  enabled?: boolean;
  valueType?: string;
  tags?: string[];
  sort?: string;
  environment?: string;
  view?: 'summary';
}

export interface CreateFeatureRequest {
  key: string;
  name: string;
  description?: string;
  valueType: string;
  defaultValue: unknown;
  metadata?: Record<string, unknown>;
  tags?: string[];
  activeFrom?: string | null;
  activeUntil?: string | null;
  environments?: string[];
  accessPolicy?: FeatureAccessPolicy;
  authProfileKey?: string;
  inputContract?: InputContract;
  trialUntil?: string | null;
  trialValue?: unknown;
}

export interface UpdateFeatureRequest {
  name?: string;
  description?: string;
  defaultValue?: unknown;
  metadata?: Record<string, unknown>;
  tags?: string[];
  activeFrom?: string | null;
  activeUntil?: string | null;
  environments?: string[];
  accessPolicy?: FeatureAccessPolicy;
  authProfileKey?: string;
  inputContract?: InputContract;
  trialUntil?: string | null;
  trialValue?: unknown;
}

export type FeatureListResponse<T extends Feature | FeatureSummary = Feature> =
  PaginatedResponse<T>;

function buildQuery(params: ListFeaturesParams): string {
  const p = new URLSearchParams();
  if (params.page) p.set('page', String(params.page));
  if (params.pageSize) p.set('pageSize', String(params.pageSize));
  if (params.search) p.set('search', params.search);
  if (params.enabled !== undefined) p.set('enabled', String(params.enabled));
  if (params.valueType) p.set('valueType', params.valueType);
  if (params.tags && params.tags.length > 0) {
    for (const tag of params.tags) {
      p.append('tags', tag);
    }
  }
  if (params.sort) p.set('sort', params.sort);
  if (params.environment) p.set('environment', params.environment);
  if (params.view) p.set('view', params.view);
  const qs = p.toString();
  return qs ? `?${qs}` : '';
}

export const featuresApi = {
  list: <T extends Feature | FeatureSummary = Feature>(params: ListFeaturesParams = {}) =>
    api.get<FeatureListResponse<T>>(`${BASE}${buildQuery(params)}`),
  get: (key: string) => api.get<Feature>(`${BASE}/${key}`),
  create: (data: CreateFeatureRequest) => api.post<Feature>(BASE, data),
  update: (key: string, data: UpdateFeatureRequest) => api.put<Feature>(`${BASE}/${key}`, data),
  remove: (key: string) => api.delete<undefined>(`${BASE}/${key}`),
  toggle: (key: string, enabled: boolean) =>
    api.patch<ToggleFeatureResponse>(`${BASE}/${key}/toggle`, { enabled }),
  environments: () => api.get<string[]>('/admin/environments'),
};

import { api } from './client';

import type { Tier, TierIcon } from './types';

const BASE = '/admin/tiers';

export interface CreateTierRequest {
  name: string;
  level: number;
  color: string;
  icon: string;
}

export interface UpdateTierRequest {
  name: string;
  level: number;
  color: string;
  icon: string;
}

export const tiersApi = {
  list: (search?: string) => {
    const qs = search ? `?search=${encodeURIComponent(search)}` : '';
    return api.get<{ data: Tier[] }>(`${BASE}${qs}`).then((res) => res.data);
  },
  getByKey: (key: string) => api.get<Tier>(`${BASE}/${key}`),
  create: (data: CreateTierRequest) => api.post<Tier>(BASE, data),
  update: (key: string, data: UpdateTierRequest) => api.put<Tier>(`${BASE}/${key}`, data),
  remove: (key: string) => api.delete<undefined>(`${BASE}/${key}`),
  listIcons: () => api.get<{ data: TierIcon[]; builtinIcons: string[] }>(`${BASE}-icons`),
  uploadIcon: (name: string, file: File): Promise<TierIcon> => {
    const formData = new FormData();
    formData.append('name', name);
    formData.append('file', file);
    return api.postForm<TierIcon>(`${BASE}-icons`, formData);
  },
  deleteIcon: (id: string) => api.delete<undefined>(`${BASE}-icons/${id}`),
};

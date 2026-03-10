import { api } from './client';

import type { Tag } from './types';

const BASE = '/admin/tags';

export interface CreateTagRequest {
  name: string;
  color: string;
}

export interface UpdateTagRequest {
  name?: string;
  color?: string;
}

export const tagsApi = {
  list: (search?: string) => {
    const qs = search ? `?search=${encodeURIComponent(search)}` : '';
    return api.get<{ data: Tag[] }>(`${BASE}${qs}`).then((res) => res.data);
  },
  create: (data: CreateTagRequest) => api.post<Tag>(BASE, data),
  update: (key: string, data: UpdateTagRequest) => api.put<Tag>(`${BASE}/${key}`, data),
  remove: (key: string) => api.delete<undefined>(`${BASE}/${key}`),
};

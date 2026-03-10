import { api } from './client';

import type {
  ApiKey,
  CreateApiKeyRequest,
  CreateApiKeyResponse,
  RotateApiKeyResponse,
} from './types';

const BASE = '/admin/api-keys';

export const apiKeysApi = {
  list: () => api.get<{ data: ApiKey[] }>(BASE).then((r) => r.data),
  create: (data: CreateApiKeyRequest) => api.post<CreateApiKeyResponse>(BASE, data),
  revoke: (id: string) => api.delete<undefined>(`${BASE}/${id}`),
  rotate: (id: string) => api.put<RotateApiKeyResponse>(`${BASE}/${id}/rotate`),
};

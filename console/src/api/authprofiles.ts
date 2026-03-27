import { api } from './client';

import type { AuthProfile, AuthProfileType } from './types';

const BASE = '/admin/auth-profiles';

export interface CreateAuthProfileRequest {
  key: string;
  name: string;
  active: boolean;
  type: AuthProfileType;
  config?: Record<string, unknown>;
  cacheEnabled?: boolean;
  cacheTTLSeconds?: number;
  secretPayload?: Record<string, string>;
}

export interface UpdateAuthProfileRequest {
  key: string;
  name: string;
  active: boolean;
  type: AuthProfileType;
  config?: Record<string, unknown>;
  cacheEnabled?: boolean;
  cacheTTLSeconds?: number;
  secretPayload?: Record<string, string>;
  replaceSecret?: boolean;
}

export interface TestAuthProfileRequest {
  name: string;
  active: boolean;
  type: AuthProfileType;
  config?: Record<string, unknown>;
  cacheEnabled?: boolean;
  cacheTTLSeconds?: number;
  secretPayload?: Record<string, string>;
  testRequest: {
    headers?: Record<string, string>;
    query?: Record<string, string>;
    body?: Record<string, unknown>;
  };
}

export interface AuthProfileTestResponse {
  ok: boolean;
  attempted: boolean;
  cached?: boolean;
  httpStatus?: number;
  details?: Record<string, unknown>;
}

export const authProfilesApi = {
  list: () => api.get<{ data: AuthProfile[] }>(BASE).then((r) => r.data),
  get: (key: string) => api.get<AuthProfile>(`${BASE}/${key}`),
  create: (data: CreateAuthProfileRequest) => api.post<AuthProfile>(BASE, data),
  update: (key: string, data: UpdateAuthProfileRequest) =>
    api.put<AuthProfile>(`${BASE}/${key}`, data),
  remove: (key: string) => api.delete<undefined>(`${BASE}/${key}`),
  test: (data: TestAuthProfileRequest) => api.post<AuthProfileTestResponse>(`${BASE}/test`, data),
};

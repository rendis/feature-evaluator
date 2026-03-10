import { api } from './client';

export interface Workspace {
  id: string;
  key: string;
  name: string;
  description: string;
  metadata?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  archivedAt?: string | null;
  archivedBy?: string;
}

export interface CreateWorkspaceRequest {
  key: string;
  name: string;
  description?: string;
}

export interface UpdateWorkspaceRequest {
  name: string;
  description?: string;
}

const BASE = '/admin/workspaces';

function buildQuery(params?: { includeArchived?: boolean }) {
  if (!params?.includeArchived) return '';
  return '?includeArchived=true';
}

export const workspacesApi = {
  list: (params?: { includeArchived?: boolean }) =>
    api.get<{ data: Workspace[] }>(`${BASE}${buildQuery(params)}`).then((r) => r.data),
  get: (key: string) => api.get<Workspace>(`${BASE}/${key}`),
  create: (data: CreateWorkspaceRequest) => api.post<Workspace>(BASE, data),
  update: (key: string, data: UpdateWorkspaceRequest) => api.put<Workspace>(`${BASE}/${key}`, data),
  archive: (key: string) => api.post<undefined>(`${BASE}/${key}/archive`),
  restore: (key: string) => api.post<undefined>(`${BASE}/${key}/restore`),
  remove: (key: string) => api.delete<undefined>(`${BASE}/${key}`),
};

import { useMutation, useQueryClient } from '@tanstack/react-query';

import type { CreateWorkspaceRequest, UpdateWorkspaceRequest } from '@/api/workspaces';

import { workspacesApi } from '@/api/workspaces';
import { workspaceQueries } from '@/queries/workspace-queries';

export function useCreateWorkspace() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateWorkspaceRequest) => workspacesApi.create(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: workspaceQueries.all() }),
  });
}

export function useUpdateWorkspace(key: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateWorkspaceRequest) => workspacesApi.update(key, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: workspaceQueries.all() }),
  });
}

export function useDeleteWorkspace() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (key: string) => workspacesApi.remove(key),
    onSuccess: () => qc.invalidateQueries({ queryKey: workspaceQueries.all() }),
  });
}

export function useArchiveWorkspace() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (key: string) => workspacesApi.archive(key),
    onSuccess: () => qc.invalidateQueries({ queryKey: workspaceQueries.all() }),
  });
}

export function useRestoreWorkspace() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (key: string) => workspacesApi.restore(key),
    onSuccess: () => qc.invalidateQueries({ queryKey: workspaceQueries.all() }),
  });
}

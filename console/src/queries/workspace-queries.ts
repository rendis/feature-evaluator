import { queryOptions } from '@tanstack/react-query';

import { workspacesApi } from '@/api/workspaces';

export const workspaceQueries = {
  all: () => ['workspaces'] as const,
  list: (params: { includeArchived?: boolean } = {}) =>
    queryOptions({
      queryKey: [...workspaceQueries.all(), 'list', params],
      queryFn: () => workspacesApi.list(params),
      staleTime: 60_000,
    }),
  detail: (key: string) =>
    queryOptions({
      queryKey: [...workspaceQueries.all(), 'detail', key],
      queryFn: () => workspacesApi.get(key),
      enabled: !!key,
    }),
};

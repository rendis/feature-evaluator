import { queryOptions } from '@tanstack/react-query';

import type { ListChangelogParams } from '@/api/changelog';

import { changelogApi } from '@/api/changelog';

export const changelogQueries = {
  all: () => ['changelog'] as const,
  list: (params: ListChangelogParams = {}) =>
    queryOptions({
      queryKey: [...changelogQueries.all(), 'list', params],
      queryFn: () => changelogApi.list(params),
    }),
  byEntity: (entityType: string, entityKey: string, params: Omit<ListChangelogParams, 'entityType' | 'entityKey'> = {}) =>
    queryOptions({
      queryKey: [...changelogQueries.all(), 'entity', entityType, entityKey, params],
      queryFn: () => changelogApi.listByEntity(entityType, entityKey, params),
    }),
};

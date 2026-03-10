import { queryOptions } from '@tanstack/react-query';

import type { ListPacksParams } from '@/api/packs';

import { packsApi } from '@/api/packs';

export const packQueries = {
  all: () => ['packs'] as const,
  list: (params: ListPacksParams = {}) =>
    queryOptions({
      queryKey: [...packQueries.all(), 'list', params],
      queryFn: () => packsApi.list(params),
    }),
  detail: (key: string) =>
    queryOptions({
      queryKey: [...packQueries.all(), 'detail', key],
      queryFn: () => packsApi.getByKey(key),
      enabled: !!key,
    }),
  activations: (key: string) =>
    queryOptions({
      queryKey: [...packQueries.all(), 'activations', key],
      queryFn: () => packsApi.listActivations(key),
      enabled: !!key,
    }),
};

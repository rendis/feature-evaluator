import { queryOptions } from '@tanstack/react-query';

import { tiersApi } from '@/api/tiers';

export const tierQueries = {
  all: () => ['tiers'] as const,
  list: (search?: string) =>
    queryOptions({
      queryKey: [...tierQueries.all(), 'list', search ?? ''],
      queryFn: () => tiersApi.list(search),
      staleTime: 30 * 1000,
    }),
  detail: (key: string) =>
    queryOptions({
      queryKey: [...tierQueries.all(), 'detail', key],
      queryFn: () => tiersApi.getByKey(key),
      enabled: !!key,
    }),
  icons: () =>
    queryOptions({
      queryKey: [...tierQueries.all(), 'icons'],
      queryFn: () => tiersApi.listIcons(),
      staleTime: 60 * 1000,
    }),
};

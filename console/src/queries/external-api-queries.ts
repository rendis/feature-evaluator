import { queryOptions } from '@tanstack/react-query';

import { externalApisApi } from '@/api/external-apis';

export const externalApiQueries = {
  all: () => ['external-apis'] as const,
  expressionProfile: () =>
    queryOptions({
      queryKey: [...externalApiQueries.all(), 'expression-profile'],
      queryFn: externalApisApi.expressionProfile,
      staleTime: 5 * 60 * 1000,
    }),
  list: () =>
    queryOptions({
      queryKey: [...externalApiQueries.all(), 'list'],
      queryFn: externalApisApi.list,
    }),
  detail: (key: string) =>
    queryOptions({
      queryKey: [...externalApiQueries.all(), 'detail', key],
      queryFn: () => externalApisApi.get(key),
      enabled: !!key,
    }),
};

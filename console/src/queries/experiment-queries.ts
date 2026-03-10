import { queryOptions } from '@tanstack/react-query';

import { experimentsApi } from '@/api/experiments';

export const experimentQueries = {
  all: () => ['experiments'] as const,
  list: () =>
    queryOptions({
      queryKey: [...experimentQueries.all(), 'list'],
      queryFn: () => experimentsApi.list(),
    }),
  detail: (id: string) =>
    queryOptions({
      queryKey: [...experimentQueries.all(), 'detail', id],
      queryFn: () => experimentsApi.get(id),
      enabled: !!id,
    }),
  results: (id: string) =>
    queryOptions({
      queryKey: [...experimentQueries.all(), 'results', id],
      queryFn: () => experimentsApi.results(id),
      enabled: !!id,
      refetchInterval: 30_000,
    }),
};

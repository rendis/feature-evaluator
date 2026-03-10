import { queryOptions } from '@tanstack/react-query';

import { schedulesApi } from '@/api/schedules';

export const scheduleQueries = {
  all: () => ['schedules'] as const,
  list: (featureKey: string) =>
    queryOptions({
      queryKey: [...scheduleQueries.all(), 'list', featureKey],
      queryFn: () => schedulesApi.list(featureKey),
      staleTime: 15_000,
    }),
};

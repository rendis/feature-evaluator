import { queryOptions } from '@tanstack/react-query';

import { tagsApi } from '@/api/tags';

export const tagQueries = {
  all: () => ['tags'] as const,
  list: (search?: string) =>
    queryOptions({
      queryKey: [...tagQueries.all(), 'list', search ?? ''],
      queryFn: () => tagsApi.list(search),
      staleTime: 30 * 1000,
    }),
};

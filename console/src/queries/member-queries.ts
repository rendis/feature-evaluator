import { queryOptions } from '@tanstack/react-query';

import { membersApi } from '@/api/members';

export const memberQueries = {
  all: () => ['members'] as const,
  list: () =>
    queryOptions({
      queryKey: [...memberQueries.all(), 'list'],
      queryFn: membersApi.list,
    }),
  me: () =>
    queryOptions({
      queryKey: [...memberQueries.all(), 'me'],
      queryFn: membersApi.me,
      staleTime: 5 * 60_000,
    }),
};

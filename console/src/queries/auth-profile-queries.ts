import { queryOptions } from '@tanstack/react-query';

import { authProfilesApi } from '@/api/authprofiles';

export const authProfileQueries = {
  all: () => ['auth-profiles'] as const,
  list: () =>
    queryOptions({
      queryKey: [...authProfileQueries.all(), 'list'],
      queryFn: authProfilesApi.list,
    }),
  detail: (key: string) =>
    queryOptions({
      queryKey: [...authProfileQueries.all(), 'detail', key],
      queryFn: () => authProfilesApi.get(key),
      enabled: !!key,
    }),
};

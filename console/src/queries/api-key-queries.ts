import { queryOptions } from '@tanstack/react-query';

import { apiKeysApi } from '@/api/apikeys';

export const apiKeyQueries = {
  all: () => ['api-keys'] as const,
  list: () =>
    queryOptions({
      queryKey: [...apiKeyQueries.all(), 'list'],
      queryFn: apiKeysApi.list,
    }),
};

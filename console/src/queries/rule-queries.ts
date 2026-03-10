import { queryOptions } from '@tanstack/react-query';

import { rulesApi } from '@/api/rules';

export const ruleQueries = {
  all: (featureKey: string) => ['features', 'detail', featureKey, 'rules'] as const,
  list: (featureKey: string) =>
    queryOptions({
      queryKey: [...ruleQueries.all(featureKey)],
      queryFn: () => rulesApi.list(featureKey),
      enabled: !!featureKey,
    }),
};

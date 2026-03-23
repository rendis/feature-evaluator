import { queryOptions } from '@tanstack/react-query';

import { securityPoliciesApi } from '@/api/security-policies';

export const securityPolicyQueries = {
  all: () => ['security-policy'] as const,
  detail: () =>
    queryOptions({
      queryKey: [...securityPolicyQueries.all(), 'detail'],
      queryFn: () => securityPoliciesApi.get(),
      staleTime: 30_000,
    }),
};

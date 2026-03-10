import { queryOptions } from '@tanstack/react-query';

import { expressionApi } from '@/api/expression';

export const expressionQueries = {
  schema: () =>
    queryOptions({
      queryKey: ['field-schema'],
      queryFn: expressionApi.schema,
      staleTime: 10 * 60_000,
    }),
  featureSchema: (featureKey: string) =>
    queryOptions({
      queryKey: ['feature-expression-schema', featureKey],
      queryFn: () => expressionApi.featureSchema(featureKey),
      enabled: !!featureKey,
      staleTime: 10 * 60_000,
    }),
};

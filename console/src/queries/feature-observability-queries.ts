import { queryOptions } from '@tanstack/react-query';

import type { FeatureObservabilityTraceParams } from '@/api/feature-observability';

import { featureObservabilityApi } from '@/api/feature-observability';

export const featureObservabilityQueries = {
  all: (featureKey: string) => ['features', 'detail', featureKey, 'observability'] as const,
  overview: (featureKey: string) =>
    queryOptions({
      queryKey: [...featureObservabilityQueries.all(featureKey), 'overview'],
      queryFn: () => featureObservabilityApi.overview(featureKey),
      enabled: !!featureKey,
    }),
  rules: (featureKey: string) =>
    queryOptions({
      queryKey: [...featureObservabilityQueries.all(featureKey), 'rules'],
      queryFn: () => featureObservabilityApi.rules(featureKey),
      enabled: !!featureKey,
    }),
  rule: (featureKey: string, ruleId: string) =>
    queryOptions({
      queryKey: [...featureObservabilityQueries.all(featureKey), 'rule', ruleId],
      queryFn: () => featureObservabilityApi.rule(featureKey, ruleId),
      enabled: !!featureKey && !!ruleId,
    }),
  traces: (featureKey: string, params: FeatureObservabilityTraceParams = {}) =>
    queryOptions({
      queryKey: [...featureObservabilityQueries.all(featureKey), 'traces', params],
      queryFn: () => featureObservabilityApi.traces(featureKey, params),
      enabled: !!featureKey,
    }),
};

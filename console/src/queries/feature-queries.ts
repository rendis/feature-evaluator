import { queryOptions } from '@tanstack/react-query';

import type { ListFeaturesParams } from '@/api/features';
import type { FeatureSummary } from '@/api/types';

import { featuresApi } from '@/api/features';

export const featureQueries = {
  all: () => ['features'] as const,
  lists: () => [...featureQueries.all(), 'list'] as const,
  details: () => [...featureQueries.all(), 'detail'] as const,
  list: (params: ListFeaturesParams = {}) =>
    queryOptions({
      queryKey: [...featureQueries.lists(), params],
      queryFn: () => featuresApi.list(params),
    }),
  summaryList: (params: Omit<ListFeaturesParams, 'view'> = {}) =>
    queryOptions({
      queryKey: [...featureQueries.lists(), { ...params, view: 'summary' as const }],
      queryFn: () => featuresApi.list<FeatureSummary>({ ...params, view: 'summary' }),
    }),
  detail: (key: string) =>
    queryOptions({
      queryKey: [...featureQueries.details(), key],
      queryFn: () => featuresApi.get(key),
      enabled: !!key,
    }),
  environments: () =>
    queryOptions({
      queryKey: [...featureQueries.all(), 'environments'],
      queryFn: () => featuresApi.environments(),
      staleTime: 5 * 60 * 1000,
    }),
};

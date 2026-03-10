import { queryOptions } from '@tanstack/react-query';

import { dashboardApi } from '@/api/dashboard';

export const dashboardQueries = {
  all: () => ['dashboard'] as const,
  stats: () =>
    queryOptions({
      queryKey: [...dashboardQueries.all(), 'stats'],
      queryFn: () => dashboardApi.stats(),
      staleTime: 60_000,
    }),
  activity: () =>
    queryOptions({
      queryKey: [...dashboardQueries.all(), 'activity'],
      queryFn: () => dashboardApi.activity(),
      staleTime: 30_000,
    }),
  operations: () =>
    queryOptions({
      queryKey: [...dashboardQueries.all(), 'operations'],
      queryFn: () => dashboardApi.operations(),
      staleTime: 30_000,
      refetchInterval: 30_000,
    }),
  errorSummary: () =>
    queryOptions({
      queryKey: [...dashboardQueries.all(), 'error-summary'],
      queryFn: () => dashboardApi.errorSummary(),
      staleTime: 60_000,
    }),
  metricsOverview: (days?: number) =>
    queryOptions({
      queryKey: [...dashboardQueries.all(), 'metrics-overview', days ?? 7],
      queryFn: () => dashboardApi.metricsOverview(days),
      staleTime: 60_000,
    }),
  metricsFeatures: (limit?: number) =>
    queryOptions({
      queryKey: [...dashboardQueries.all(), 'metrics-features', limit ?? 10],
      queryFn: () => dashboardApi.metricsFeatures(limit),
      staleTime: 60_000,
    }),
  metricsReasons: () =>
    queryOptions({
      queryKey: [...dashboardQueries.all(), 'metrics-reasons'],
      queryFn: () => dashboardApi.metricsReasons(),
      staleTime: 60_000,
    }),
  metricsEnvironments: () =>
    queryOptions({
      queryKey: [...dashboardQueries.all(), 'metrics-environments'],
      queryFn: () => dashboardApi.metricsEnvironments(),
      staleTime: 60_000,
    }),
};

import { api } from './client';

export interface DashboardStats {
  totalFeatures: number;
  activeFeatures: number;
  totalSegments: number;
  totalSegmentMembers: number;
}

export interface RecentActivity {
  id: string;
  type: 'feature_created' | 'feature_updated' | 'feature_toggled' | 'rule_created' | 'rule_updated';
  featureKey: string;
  description: string;
  createdBy: string;
  createdAt: string;
}

export interface ErrorSummary {
  total: number;
  byType: { type: string; count: number }[];
}

export interface MetricsOverview {
  today: { total: number; errors: number; cacheHitRatio: number };
  trend: { date: string; total: number; errors: number }[];
}

export interface DashboardOperations {
  checkedAt: string;
  overallStatus: 'healthy' | 'degraded' | 'unhealthy';
  services: {
    postgresql: {
      status: 'healthy' | 'unhealthy';
      latencyMs: number | null;
    };
    redis: {
      status: 'healthy' | 'degraded' | 'unhealthy';
      latencyMs: number | null;
      circuitOpen: boolean;
      openUntil: string | null;
    };
  };
  metrics: {
    evaluationsToday: number;
    errorsToday: number;
    cacheHitRatio: number;
    rateLimitRejectsToday: number;
    externalP50Ms: number;
    externalP95Ms: number;
    circuitBreakerOpenEvents: number;
    circuitBreakerCloseEvents: number;
  };
}

export interface TopFeature {
  featureKey: string;
  count: number;
}

export type ReasonBreakdown = Record<string, number>;

export type EnvironmentMetric = Record<string, number>;

export const dashboardApi = {
  stats: () => api.get<DashboardStats>('/admin/dashboard/stats'),
  activity: () => api.get<RecentActivity[]>('/admin/dashboard/activity'),
  errorSummary: () => api.get<ErrorSummary>('/admin/dashboard/error-summary'),
  operations: () => api.get<DashboardOperations>('/admin/dashboard/operations'),
  metricsOverview: (days?: number) =>
    api.get<MetricsOverview>(`/admin/dashboard/metrics/overview?days=${days ?? 7}`),
  metricsFeatures: (limit?: number) =>
    api.get<TopFeature[]>(`/admin/dashboard/metrics/features?limit=${limit ?? 10}`),
  metricsReasons: () => api.get<ReasonBreakdown>('/admin/dashboard/metrics/reasons'),
  metricsEnvironments: () => api.get<EnvironmentMetric>('/admin/dashboard/metrics/environments'),
};

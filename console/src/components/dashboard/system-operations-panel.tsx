import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Activity,
  AlertTriangle,
  Database,
  Gauge,
  RefreshCw,
  ServerCog,
  ShieldAlert,
  Timer,
  type LucideIcon,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { DashboardOperations } from '@/api/dashboard';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { useLocaleFormatters } from '@/hooks/use-locale-formatters';
import { getVisibleErrorMessage } from '@/lib/display-error';
import { dashboardQueries } from '@/queries/dashboard-queries';

type Status = DashboardOperations['overallStatus'];

const STATUS_VARIANT: Record<Status, 'success' | 'warning' | 'destructive'> = {
  healthy: 'success',
  degraded: 'warning',
  unhealthy: 'destructive',
};

function formatLatency(value: number | null, fallback: string) {
  if (value === null) {
    return fallback;
  }

  return `${value} ms`;
}

function OperationsMetric({
  icon: Icon,
  label,
  value,
}: {
  icon: LucideIcon;
  label: string;
  value: string;
}) {
  return (
    <div className="rounded-lg border bg-card/60 p-4">
      <div className="flex items-center gap-2 text-sm font-medium">
        <Icon className="text-muted-foreground h-4 w-4" />
        <span>{label}</span>
      </div>
      <p className="mt-3 text-2xl font-semibold tracking-tight">{value}</p>
    </div>
  );
}

function DependencyCard({
  icon: Icon,
  title,
  status,
  latencyMs,
  detail,
}: {
  icon: LucideIcon;
  title: string;
  status: Status;
  latencyMs: number | null;
  detail?: string;
}) {
  const { t } = useTranslation('dashboard');

  return (
    <div className="rounded-lg border bg-card/60 p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="bg-muted rounded-lg p-2">
            <Icon className="h-4 w-4" />
          </div>
          <div>
            <p className="text-sm font-medium">{title}</p>
            <p className="text-muted-foreground text-xs">
              {t('operations.latency')}: {formatLatency(latencyMs, t('operations.unavailable'))}
            </p>
          </div>
        </div>
        <Badge variant={STATUS_VARIANT[status]}>{t(`operations.status.${status}`)}</Badge>
      </div>
      {detail ? <p className="text-muted-foreground mt-3 text-xs">{detail}</p> : null}
    </div>
  );
}

function PanelSkeleton() {
  return (
    <div className="rounded-lg border p-4" data-testid="system-operations-panel">
      <div className="flex items-center justify-between gap-3">
        <div className="space-y-2">
          <Skeleton className="h-5 w-44" />
          <Skeleton className="h-4 w-60" />
        </div>
        <Skeleton className="h-9 w-9" />
      </div>
      <div className="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Skeleton className="h-28 w-full rounded-lg" />
        <Skeleton className="h-28 w-full rounded-lg" />
      </div>
      <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 6 }).map((_, index) => (
          <Skeleton key={index} className="h-24 w-full rounded-lg" />
        ))}
      </div>
    </div>
  );
}

function RefreshButton({
  disabled,
  spinning,
  onRefresh,
}: {
  disabled?: boolean;
  spinning?: boolean;
  onRefresh: () => Promise<void>;
}) {
  const { t } = useTranslation('dashboard');

  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={onRefresh}
      disabled={disabled}
      aria-label={t('operations.refresh')}
    >
      <RefreshCw className={`h-4 w-4 ${spinning ? 'animate-spin' : ''}`} />
    </Button>
  );
}

function buildMetrics(
  t: ReturnType<typeof useTranslation>['t'],
  data: DashboardOperations,
  formatNumber: (value: number, options?: Intl.NumberFormatOptions) => string,
) {
  return [
    {
      label: t('operations.metrics.evaluationsToday'),
      value: formatNumber(data.metrics.evaluationsToday),
      icon: Activity,
    },
    {
      label: t('operations.metrics.errorsToday'),
      value: formatNumber(data.metrics.errorsToday),
      icon: AlertTriangle,
    },
    {
      label: t('operations.metrics.cacheHitRatio'),
      value: formatNumber(data.metrics.cacheHitRatio, {
        style: 'percent',
        maximumFractionDigits: 1,
      }),
      icon: Gauge,
    },
    {
      label: t('operations.metrics.rateLimitRejectsToday'),
      value: formatNumber(data.metrics.rateLimitRejectsToday),
      icon: ShieldAlert,
    },
    {
      label: t('operations.metrics.externalLatency'),
      value: `p50 ${formatNumber(data.metrics.externalP50Ms)} / p95 ${formatNumber(data.metrics.externalP95Ms)} ms`,
      icon: Timer,
    },
    {
      label: t('operations.metrics.circuitBreakerEvents'),
      value: `${formatNumber(data.metrics.circuitBreakerOpenEvents)} / ${formatNumber(data.metrics.circuitBreakerCloseEvents)}`,
      icon: ServerCog,
    },
  ];
}

function OperationsErrorState({
  error,
  onRefresh,
}: {
  error: unknown;
  onRefresh: () => Promise<void>;
}) {
  const { t } = useTranslation('dashboard');

  return (
    <div className="rounded-lg border p-4" data-testid="system-operations-panel">
      <div className="flex items-center justify-between gap-3">
        <div>
          <h3 className="text-sm font-semibold">{t('operations.title')}</h3>
          <p className="text-muted-foreground mt-1 text-sm">
            {getVisibleErrorMessage(error, t('operations.error'))}
          </p>
        </div>
        <RefreshButton onRefresh={onRefresh} />
      </div>
    </div>
  );
}

function OperationsContent({
  data,
  isFetching,
  onRefresh,
}: {
  data: DashboardOperations;
  isFetching: boolean;
  onRefresh: () => Promise<void>;
}) {
  const { t } = useTranslation('dashboard');
  const { formatNumber, formatTime } = useLocaleFormatters();

  const lastChecked = formatTime(data.checkedAt);
  const redisDetail =
    data.services.redis.circuitOpen && data.services.redis.openUntil
      ? t('operations.circuitOpenUntil', {
          time: formatTime(data.services.redis.openUntil),
        })
      : undefined;
  const metrics = buildMetrics(t, data, formatNumber);

  return (
    <section className="rounded-lg border p-4" data-testid="system-operations-panel">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-sm font-semibold">{t('operations.title')}</h3>
            <Badge variant={STATUS_VARIANT[data.overallStatus]}>
              {t(`operations.status.${data.overallStatus}`)}
            </Badge>
            <Badge variant="outline">{t('operations.globalLabel')}</Badge>
          </div>
          <p className="text-muted-foreground mt-1 text-sm">{t('operations.globalScope')}</p>
          <p className="text-muted-foreground mt-2 text-xs">
            {t('operations.lastChecked', { time: lastChecked })}
          </p>
        </div>
        <RefreshButton disabled={isFetching} spinning={isFetching} onRefresh={onRefresh} />
      </div>

      <div className="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-2">
        <DependencyCard
          icon={Database}
          title={t('operations.services.postgresql')}
          status={data.services.postgresql.status}
          latencyMs={data.services.postgresql.latencyMs}
        />
        <DependencyCard
          icon={ServerCog}
          title={t('operations.services.redis')}
          status={data.services.redis.status}
          latencyMs={data.services.redis.latencyMs}
          detail={redisDetail}
        />
      </div>

      <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {metrics.map((metric) => (
          <OperationsMetric
            key={metric.label}
            icon={metric.icon}
            label={metric.label}
            value={metric.value}
          />
        ))}
      </div>
    </section>
  );
}

export function SystemOperationsPanel() {
  const queryClient = useQueryClient();
  const operationsQuery = dashboardQueries.operations();
  const { data, error, isLoading, isFetching } = useQuery(operationsQuery);

  const handleRefresh = async () => {
    await queryClient.invalidateQueries({
      queryKey: operationsQuery.queryKey,
      exact: true,
    });
  };

  if (isLoading) {
    return <PanelSkeleton />;
  }

  if (!data) {
    return <OperationsErrorState error={error} onRefresh={handleRefresh} />;
  }

  return <OperationsContent data={data} isFetching={isFetching} onRefresh={handleRefresh} />;
}

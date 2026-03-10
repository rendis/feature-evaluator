import { useSuspenseQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Area, AreaChart, ResponsiveContainer } from 'recharts';

import { useLocaleFormatters } from '@/hooks/use-locale-formatters';
import { Skeleton } from '@/components/ui/skeleton';
import { dashboardQueries } from '@/queries/dashboard-queries';

function SparklineChart({ data, color }: { data: { value: number }[]; color: string }) {
  return (
    <ResponsiveContainer width="100%" height={40}>
      <AreaChart data={data} margin={{ top: 0, right: 0, left: 0, bottom: 0 }}>
        <defs>
          <linearGradient id={`grad-${color}`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity={0.3} />
            <stop offset="100%" stopColor={color} stopOpacity={0} />
          </linearGradient>
        </defs>
        <Area
          type="monotone"
          dataKey="value"
          stroke={color}
          strokeWidth={1.5}
          fill={`url(#grad-${color})`}
          isAnimationActive={false}
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}

function MetricCard({
  title,
  value,
  sparkData,
  sparkColor,
  valueColor,
}: {
  title: string;
  value: string;
  sparkData: { value: number }[];
  sparkColor: string;
  valueColor?: string;
}) {
  return (
    <div className="rounded-lg border bg-card p-6 shadow-sm">
      <p className="text-muted-foreground text-sm font-medium">{title}</p>
      <p className={`mt-1 text-2xl font-bold tracking-tight ${valueColor ?? ''}`}>{value}</p>
      <div className="mt-3">
        <SparklineChart data={sparkData} color={sparkColor} />
      </div>
    </div>
  );
}

export function MetricsOverviewSkeleton() {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
      {Array.from({ length: 3 }).map((_, i) => (
        <Skeleton key={i} className="h-32 w-full rounded-lg" />
      ))}
    </div>
  );
}

export function MetricsOverview() {
  const { t } = useTranslation('dashboard');
  const { formatNumber } = useLocaleFormatters();
  const { data } = useSuspenseQuery(dashboardQueries.metricsOverview());

  const totalSparkData = data.trend.map((d) => ({ value: d.total }));
  const errorSparkData = data.trend.map((d) => ({ value: d.total > 0 ? (d.errors / d.total) * 100 : 0 }));
  const cacheSparkData = data.trend.map(() => ({ value: data.today.cacheHitRatio * 100 }));

  const errorRate = data.today.total > 0 ? (data.today.errors / data.today.total) * 100 : 0;
  const cacheRatio = data.today.cacheHitRatio * 100;

  const primaryColor = 'oklch(0.55 0.15 250)';
  const destructiveColor = 'oklch(0.58 0.22 25)';
  const successColor = 'oklch(0.6 0.18 145)';

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
      <MetricCard
        title={t('metrics.totalEvals')}
        value={formatNumber(data.today.total)}
        sparkData={totalSparkData}
        sparkColor={primaryColor}
      />
      <MetricCard
        title={t('metrics.errorRate')}
        value={formatNumber(errorRate / 100, {
          style: 'percent',
          minimumFractionDigits: 1,
          maximumFractionDigits: 1,
        })}
        sparkData={errorSparkData}
        sparkColor={destructiveColor}
        valueColor={errorRate > 5 ? 'text-destructive' : undefined}
      />
      <MetricCard
        title={t('metrics.cacheHitRatio')}
        value={formatNumber(cacheRatio / 100, {
          style: 'percent',
          minimumFractionDigits: 1,
          maximumFractionDigits: 1,
        })}
        sparkData={cacheSparkData}
        sparkColor={successColor}
        valueColor={cacheRatio > 80 ? 'text-success' : undefined}
      />
    </div>
  );
}

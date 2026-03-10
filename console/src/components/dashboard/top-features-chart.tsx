import { useSuspenseQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Bar, BarChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';

import { Skeleton } from '@/components/ui/skeleton';
import { dashboardQueries } from '@/queries/dashboard-queries';

export function TopFeaturesChartSkeleton() {
  return <Skeleton className="h-72 w-full rounded-lg" />;
}

export function TopFeaturesChart() {
  const { t } = useTranslation('dashboard');
  const { data } = useSuspenseQuery(dashboardQueries.metricsFeatures());

  if (data.length === 0) {
    return (
      <div className="text-muted-foreground rounded-lg border border-dashed p-8 text-center text-sm">
        {t('metrics.empty')}
      </div>
    );
  }

  const chartData = [...data].reverse();

  return (
    <div className="rounded-lg border">
      <div className="border-b px-4 py-3">
        <h3 className="text-sm font-semibold">{t('metrics.topFeatures')}</h3>
      </div>
      <div className="p-4">
        <ResponsiveContainer width="100%" height={Math.max(200, chartData.length * 32)}>
          <BarChart data={chartData} layout="vertical" margin={{ top: 0, right: 16, left: 0, bottom: 0 }}>
            <XAxis type="number" hide />
            <YAxis
              type="category"
              dataKey="featureKey"
              width={120}
              tick={{ fontSize: 12, fill: 'oklch(0.55 0.01 260)' }}
              tickLine={false}
              axisLine={false}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--popover)',
                border: '1px solid var(--border)',
                borderRadius: '0.375rem',
                fontSize: '0.875rem',
              }}
              cursor={{ fill: 'var(--accent)', opacity: 0.5 }}
            />
            <Bar
              dataKey="count"
              fill="oklch(0.55 0.15 250)"
              radius={[0, 4, 4, 0]}
              isAnimationActive={false}
            />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

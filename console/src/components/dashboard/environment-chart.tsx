import { useSuspenseQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Bar, BarChart, Cell, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';

import { Skeleton } from '@/components/ui/skeleton';
import { dashboardQueries } from '@/queries/dashboard-queries';

const ENV_COLORS: Record<string, string> = {
  dev: 'oklch(0.6 0.12 195)',
  development: 'oklch(0.6 0.12 195)',
  uat: 'oklch(0.75 0.17 75)',
  staging: 'oklch(0.75 0.17 75)',
  production: 'oklch(0.6 0.18 145)',
  prod: 'oklch(0.6 0.18 145)',
};

const FALLBACK_COLOR = 'oklch(0.55 0.15 250)';

export function EnvironmentChartSkeleton() {
  return <Skeleton className="h-56 w-full rounded-lg" />;
}

export function EnvironmentChart() {
  const { t } = useTranslation('dashboard');
  const { data } = useSuspenseQuery(dashboardQueries.metricsEnvironments());

  const entries = Object.entries(data);
  const total = entries.reduce((sum, [, count]) => sum + count, 0);

  if (total === 0) return null;

  const chartData = entries.map(([env, count]) => ({
    name: env,
    count,
    fill: ENV_COLORS[env.toLowerCase()] ?? FALLBACK_COLOR,
  }));

  return (
    <div className="rounded-lg border">
      <div className="border-b px-4 py-3">
        <h3 className="text-sm font-semibold">{t('metrics.environments')}</h3>
      </div>
      <div className="p-4">
        <ResponsiveContainer width="100%" height={200}>
          <BarChart data={chartData} margin={{ top: 0, right: 16, left: 0, bottom: 0 }}>
            <XAxis
              dataKey="name"
              tick={{ fontSize: 12, fill: 'oklch(0.55 0.01 260)' }}
              tickLine={false}
              axisLine={false}
            />
            <YAxis hide />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--popover)',
                border: '1px solid var(--border)',
                borderRadius: '0.375rem',
                fontSize: '0.875rem',
              }}
              cursor={{ fill: 'var(--accent)', opacity: 0.5 }}
            />
            <Bar dataKey="count" radius={[4, 4, 0, 0]} isAnimationActive={false}>
              {chartData.map((entry) => (
                <Cell key={entry.name} fill={entry.fill} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}


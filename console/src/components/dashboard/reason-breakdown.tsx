import { useSuspenseQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from 'recharts';

import { Skeleton } from '@/components/ui/skeleton';
import { dashboardQueries } from '@/queries/dashboard-queries';

const REASON_COLORS: Record<string, string> = {
  matched_rule: 'oklch(0.55 0.15 250)',
  default_value: 'oklch(0.55 0.01 260)',
  feature_disabled: 'oklch(0.75 0.17 75)',
  error: 'oklch(0.58 0.22 25)',
  not_yet_active: 'oklch(0.6 0.12 195)',
  expired: 'oklch(0.7 0.01 260)',
  environment_mismatch: 'oklch(0.65 0.17 75)',
};

const FALLBACK_COLOR = 'oklch(0.6 0.01 260)';

export function ReasonBreakdownSkeleton() {
  return <Skeleton className="h-72 w-full rounded-lg" />;
}

export function ReasonBreakdown() {
  const { t } = useTranslation('dashboard');
  const { data } = useSuspenseQuery(dashboardQueries.metricsReasons());

  const entries = Object.entries(data);
  const total = entries.reduce((sum, [, count]) => sum + count, 0);

  if (total === 0) {
    return (
      <div className="text-muted-foreground rounded-lg border border-dashed p-8 text-center text-sm">
        {t('metrics.empty')}
      </div>
    );
  }

  const chartData = entries.map(([reason, count]) => ({
    name: t(`metrics.reasons.${reason}`, reason),
    value: count,
    reason,
  }));

  return (
    <div className="rounded-lg border">
      <div className="border-b px-4 py-3">
        <h3 className="text-sm font-semibold">{t('metrics.reasonsTitle')}</h3>
      </div>
      <div className="flex flex-col items-center gap-4 p-4 sm:flex-row">
        <ResponsiveContainer width="100%" height={200} className="max-w-[200px]">
          <PieChart>
            <Pie
              data={chartData}
              dataKey="value"
              nameKey="name"
              cx="50%"
              cy="50%"
              innerRadius={50}
              outerRadius={80}
              strokeWidth={2}
              stroke="var(--card)"
              isAnimationActive={false}
            >
              {chartData.map((entry) => (
                <Cell key={entry.reason} fill={REASON_COLORS[entry.reason] ?? FALLBACK_COLOR} />
              ))}
            </Pie>
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--popover)',
                border: '1px solid var(--border)',
                borderRadius: '0.375rem',
                fontSize: '0.875rem',
              }}
            />
          </PieChart>
        </ResponsiveContainer>
        <div className="flex flex-wrap gap-x-4 gap-y-1.5 text-xs">
          {chartData.map((entry) => (
            <div key={entry.reason} className="flex items-center gap-1.5">
              <span
                className="inline-block h-2.5 w-2.5 rounded-full"
                style={{ backgroundColor: REASON_COLORS[entry.reason] ?? FALLBACK_COLOR }}
              />
              <span className="text-muted-foreground">{entry.name}</span>
              <span className="font-medium">{entry.value}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

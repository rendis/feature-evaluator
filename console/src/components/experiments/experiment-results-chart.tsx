import { useTranslation } from 'react-i18next';
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';

import type { ExperimentResults } from '@/api/types';

interface ExperimentResultsChartProps {
  results: ExperimentResults;
}

export function ExperimentResultsChart({ results }: ExperimentResultsChartProps) {
  const { t } = useTranslation('experiments');

  const data = results.variants.map((v) => ({
    name: v.variantKey,
    [t('results.conversionRate')]: Number((v.conversionRate * 100).toFixed(2)),
    [t('results.exposures')]: v.exposures,
  }));

  return (
    <div className="h-64 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={data}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="name" />
          <YAxis unit="%" />
          <Tooltip />
          <Bar dataKey={t('results.conversionRate')} fill="hsl(var(--chart-1))" radius={[4, 4, 0, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

import { useTranslation } from 'react-i18next';

import type { ExperimentResults } from '@/api/types';

import { Badge } from '@/components/ui/badge';

interface ExperimentResultsTableProps {
  results: ExperimentResults;
  winnerKey?: string;
}

export function ExperimentResultsTable({ results, winnerKey }: ExperimentResultsTableProps) {
  const { t } = useTranslation('experiments');

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-4 text-sm">
        <span>
          {t('results.totalExposures')}: <strong>{results.totalExposures}</strong>
        </span>
        <span>
          {t('results.totalConversions')}: <strong>{results.totalConversions}</strong>
        </span>
        {results.isSignificant && <Badge variant="success">{t('results.significant')}</Badge>}
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b">
              <th className="py-2 text-left font-medium">{t('results.variant')}</th>
              <th className="py-2 text-right font-medium">{t('results.exposures')}</th>
              <th className="py-2 text-right font-medium">{t('results.conversions')}</th>
              <th className="py-2 text-right font-medium">{t('results.conversionRate')}</th>
              <th className="py-2 text-right font-medium">{t('results.confidence')}</th>
            </tr>
          </thead>
          <tbody>
            {results.variants.map((v) => (
              <tr key={v.variantKey} className="border-b">
                <td className="py-2">
                  <span className="font-mono">{v.variantKey}</span>
                  {v.variantKey === winnerKey && (
                    <Badge variant="success" className="ml-2">
                      {t('winner')}
                    </Badge>
                  )}
                </td>
                <td className="py-2 text-right">{v.exposures}</td>
                <td className="py-2 text-right">{v.conversions}</td>
                <td className="py-2 text-right">{(v.conversionRate * 100).toFixed(2)}%</td>
                <td className="text-muted-foreground py-2 text-right">
                  {(v.confidenceLow * 100).toFixed(2)}% - {(v.confidenceHigh * 100).toFixed(2)}%
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

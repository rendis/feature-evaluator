import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import type { Experiment } from '@/api/types';

import { ExperimentResultsChart } from '@/components/experiments/experiment-results-chart';
import { ExperimentResultsTable } from '@/components/experiments/experiment-results-table';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { PermissionButton } from '@/components/shared/permission-button';
import { Badge } from '@/components/ui/badge';
import { useDeclareWinner } from '@/mutations/experiment-mutations';
import { experimentQueries } from '@/queries/experiment-queries';

interface ExperimentDashboardProps {
  experiment: Experiment;
}

export function ExperimentDashboard({ experiment }: ExperimentDashboardProps) {
  const { t } = useTranslation('experiments');
  const { data: results } = useQuery(experimentQueries.results(experiment.id));
  const declareWinner = useDeclareWinner();
  const [winnerCandidate, setWinnerCandidate] = useState<string | null>(null);

  const handleDeclareWinner = () => {
    if (!winnerCandidate) return;
    declareWinner.mutate(
      { id: experiment.id, variantKey: winnerCandidate },
      {
        onSuccess: () => {
          toast.success(t('actions.winnerDeclared'));
          setWinnerCandidate(null);
        },
        onError: () => toast.error(t('actions.winnerError')),
      },
    );
  };

  if (!results) return null;

  return (
    <div className="space-y-6">
      <ExperimentResultsChart results={results} />
      <ExperimentResultsTable results={results} winnerKey={experiment.winnerKey} />

      {!experiment.winnerKey &&
        (experiment.status === 'running' || experiment.status === 'completed') && (
          <div className="space-y-2">
            <h3 className="text-sm font-medium">{t('actions.declareWinner')}</h3>
            <div className="flex flex-wrap gap-2">
              {experiment.variants.map((v) => (
                <PermissionButton
                  key={v.key}
                  permission="experiments.write"
                  variant="outline"
                  size="sm"
                  onClick={() => setWinnerCandidate(v.key)}
                >
                  {v.key}
                </PermissionButton>
              ))}
            </div>
          </div>
        )}

      {experiment.winnerKey && (
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">{t('winner')}:</span>
          <Badge variant="success" className="font-mono">
            {experiment.winnerKey}
          </Badge>
        </div>
      )}

      <ConfirmDialog
        open={!!winnerCandidate}
        onOpenChange={(open) => !open && setWinnerCandidate(null)}
        title={t('actions.declareWinner')}
        description={t('actions.declareWinnerConfirm', { variant: winnerCandidate })}
        onConfirm={handleDeclareWinner}
        loading={declareWinner.isPending}
      />
    </div>
  );
}

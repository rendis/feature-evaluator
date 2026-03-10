import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import type { Experiment } from '@/api/types';

import { ExperimentStatusBadge } from '@/components/experiments/experiment-status-badge';
import { PermissionButton } from '@/components/shared/permission-button';
import {
  useCompleteExperiment,
  usePauseExperiment,
  useStartExperiment,
} from '@/mutations/experiment-mutations';

interface ExperimentDetailHeaderProps {
  experiment: Experiment;
}

export function ExperimentDetailHeader({ experiment }: ExperimentDetailHeaderProps) {
  const { t } = useTranslation('experiments');
  const startExperiment = useStartExperiment();
  const pauseExperiment = usePauseExperiment();
  const completeExperiment = useCompleteExperiment();

  const handleStart = () => {
    startExperiment.mutate(experiment.id, {
      onSuccess: () => toast.success(t('actions.started')),
      onError: () => toast.error(t('actions.startError')),
    });
  };

  const handlePause = () => {
    pauseExperiment.mutate(experiment.id, {
      onSuccess: () => toast.success(t('actions.paused')),
      onError: () => toast.error(t('actions.pauseError')),
    });
  };

  const handleComplete = () => {
    completeExperiment.mutate(experiment.id, {
      onSuccess: () => toast.success(t('actions.completed')),
      onError: () => toast.error(t('actions.completeError')),
    });
  };

  return (
    <div className="flex items-center justify-between">
      <div>
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold">{experiment.name}</h1>
          <ExperimentStatusBadge status={experiment.status} />
        </div>
        {experiment.description && (
          <p className="text-muted-foreground mt-1">{experiment.description}</p>
        )}
        <p className="text-muted-foreground mt-1 text-sm">
          {t('featureKey')}: <span className="font-mono">{experiment.featureKey}</span>
        </p>
      </div>
      <div className="flex gap-2">
        {(experiment.status === 'draft' || experiment.status === 'paused') && (
          <PermissionButton
            permission="experiments.write"
            onClick={handleStart}
            disabled={startExperiment.isPending}
          >
            {t('actions.start')}
          </PermissionButton>
        )}
        {experiment.status === 'running' && (
          <>
            <PermissionButton
              permission="experiments.write"
              variant="outline"
              onClick={handlePause}
              disabled={pauseExperiment.isPending}
            >
              {t('actions.pause')}
            </PermissionButton>
            <PermissionButton
              permission="experiments.write"
              variant="destructive"
              onClick={handleComplete}
              disabled={completeExperiment.isPending}
            >
              {t('actions.complete')}
            </PermissionButton>
          </>
        )}
      </div>
    </div>
  );
}

import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { useTranslation } from 'react-i18next';

import { ExperimentDashboard } from '@/components/experiments/experiment-dashboard';
import { ExperimentDetailHeader } from '@/components/experiments/experiment-detail-header';
import { ExperimentForm } from '@/components/experiments/experiment-form';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { experimentQueries } from '@/queries/experiment-queries';
import { featureQueries } from '@/queries/feature-queries';

export const Route = createFileRoute('/_authenticated/experiments/$experimentId')({
  component: ExperimentDetailPage,
  pendingComponent: () => <LoadingSkeleton rows={8} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function ExperimentDetailPage() {
  const { experimentId } = Route.useParams();
  const { t } = useTranslation('experiments');
  const { data: experiment } = useSuspenseQuery(experimentQueries.detail(experimentId));
  const { data: featuresData } = useSuspenseQuery(featureQueries.list({ pageSize: 1000 }));
  const featureKeys = featuresData.data.map((f) => f.key);

  const isDraft = experiment.status === 'draft';

  return (
    <div className="space-y-6">
      <ExperimentDetailHeader experiment={experiment} />

      {isDraft ? (
        <div>
          <h2 className="mb-4 text-lg font-semibold">{t('form.editTitle')}</h2>
          <ExperimentForm experiment={experiment} featureKeys={featureKeys} />
        </div>
      ) : (
        <div>
          <h2 className="mb-4 text-lg font-semibold">{t('results.title')}</h2>
          <ExperimentDashboard experiment={experiment} />
        </div>
      )}
    </div>
  );
}

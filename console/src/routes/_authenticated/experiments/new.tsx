import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { useTranslation } from 'react-i18next';

import { ExperimentForm } from '@/components/experiments/experiment-form';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { PageHeader } from '@/components/shared/page-header';
import { featureQueries } from '@/queries/feature-queries';

export const Route = createFileRoute('/_authenticated/experiments/new')({
  component: NewExperimentPage,
  pendingComponent: () => <LoadingSkeleton rows={5} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function NewExperimentPage() {
  const { t } = useTranslation('experiments');
  const { data } = useSuspenseQuery(featureQueries.list({ pageSize: 1000 }));
  const featureKeys = data.data.map((f) => f.key);

  return (
    <div className="space-y-6">
      <PageHeader title={t('form.createTitle')} description={t('form.createDescription')} />
      <ExperimentForm featureKeys={featureKeys} />
    </div>
  );
}

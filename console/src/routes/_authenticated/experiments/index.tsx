import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute, Link } from '@tanstack/react-router';
import { FlaskConical, Plus } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { ExperimentList } from '@/components/experiments/experiment-list';
import { EmptyState } from '@/components/shared/empty-state';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { PageHeader } from '@/components/shared/page-header';
import { PermissionButton } from '@/components/shared/permission-button';
import { experimentQueries } from '@/queries/experiment-queries';

export const Route = createFileRoute('/_authenticated/experiments/')({
  component: ExperimentsPage,
  pendingComponent: () => <LoadingSkeleton rows={5} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function ExperimentsPage() {
  const { t } = useTranslation('experiments');
  const { data: experiments } = useSuspenseQuery(experimentQueries.list());

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('title')}
        description={t('description')}
        actions={
          <PermissionButton permission="experiments.write" asChild>
            <Link to="/experiments/new">
              <Plus className="mr-2 h-4 w-4" />
              {t('create')}
            </Link>
          </PermissionButton>
        }
      />

      {experiments.length === 0 ? (
        <EmptyState
          icon={<FlaskConical className="h-10 w-10" />}
          title={t('empty.title')}
          description={t('empty.description')}
          action={
            <PermissionButton permission="experiments.write" asChild>
              <Link to="/experiments/new">{t('create')}</Link>
            </PermissionButton>
          }
        />
      ) : (
        <ExperimentList experiments={experiments} />
      )}
    </div>
  );
}

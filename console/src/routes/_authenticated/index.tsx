import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute, Link } from '@tanstack/react-router';
import { ToggleLeft } from 'lucide-react';
import { Suspense } from 'react';
import { useTranslation } from 'react-i18next';

import { ActivityFeed } from '@/components/dashboard/activity-feed';
import { ErrorSummaryCard } from '@/components/dashboard/error-summary-card';
import { MetricsSection } from '@/components/dashboard/metrics-section';
import { StatsGrid } from '@/components/dashboard/stats-grid';
import { SystemOperationsPanel } from '@/components/dashboard/system-operations-panel';
import { EmptyState } from '@/components/shared/empty-state';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { PageHeader } from '@/components/shared/page-header';
import { PermissionButton } from '@/components/shared/permission-button';
import { Skeleton } from '@/components/ui/skeleton';
import { dashboardQueries } from '@/queries/dashboard-queries';

export const Route = createFileRoute('/_authenticated/')({
  component: DashboardPage,
  pendingComponent: () => <LoadingSkeleton rows={6} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function StatsSkeleton() {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {Array.from({ length: 4 }).map((_, i) => (
        <Skeleton key={i} className="h-24 w-full rounded-lg" />
      ))}
    </div>
  );
}

function ContentSkeleton() {
  return <Skeleton className="h-48 w-full rounded-lg" />;
}

export function DashboardContent() {
  const { t } = useTranslation('dashboard');
  const { data } = useSuspenseQuery(dashboardQueries.stats());

  return (
    <>
      <StatsGrid />
      <SystemOperationsPanel />
      {data.totalFeatures === 0 ? (
        <EmptyState
          icon={<ToggleLeft className="h-10 w-10" />}
          title={t('empty.title')}
          description={t('empty.description')}
          action={
            <PermissionButton permission="features.write" asChild>
              <Link to="/features/new">{t('empty.cta')}</Link>
            </PermissionButton>
          }
        />
      ) : (
        <>
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
            <div className="lg:col-span-2">
              <Suspense fallback={<ContentSkeleton />}>
                <ActivityFeed />
              </Suspense>
            </div>
            <div>
              <Suspense fallback={<ContentSkeleton />}>
                <ErrorSummaryCard />
              </Suspense>
            </div>
          </div>
          <MetricsSection />
        </>
      )}
    </>
  );
}

function DashboardPage() {
  const { t } = useTranslation('dashboard');

  return (
    <div className="space-y-6">
      <PageHeader title={t('title')} description={t('description')} />
      <Suspense fallback={<StatsSkeleton />}>
        <DashboardContent />
      </Suspense>
    </div>
  );
}

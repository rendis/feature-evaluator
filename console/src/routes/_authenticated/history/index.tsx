import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { History } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { ListChangelogParams } from '@/api/changelog';

import { ChangeFilters } from '@/components/history/change-filters';
import { ChangeTimeline } from '@/components/history/change-timeline';
import { EmptyState } from '@/components/shared/empty-state';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { PageHeader } from '@/components/shared/page-header';
import { Button } from '@/components/ui/button';
import { changelogQueries } from '@/queries/changelog-queries';

export const Route = createFileRoute('/_authenticated/history/')({
  component: HistoryPage,
  pendingComponent: () => <LoadingSkeleton rows={8} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function HistoryPage() {
  const { t } = useTranslation('history');
  const [params, setParams] = useState<ListChangelogParams>({ page: 1, pageSize: 20 });
  const { data } = useSuspenseQuery(changelogQueries.list(params));

  const entries = data.data;
  const pagination = data.pagination;

  return (
    <div className="space-y-6">
      <PageHeader title={t('title')} description={t('description')} />
      <ChangeFilters params={params} onChange={setParams} />

      {entries.length === 0 ? (
        <EmptyState
          icon={<History className="h-10 w-10" />}
          title={t('empty.title')}
          description={t('empty.description')}
        />
      ) : (
        <>
          <ChangeTimeline entries={entries} />
          <HistoryPagination
            page={pagination.page}
            totalPages={pagination.totalPages}
            onPageChange={(page) => setParams((p) => ({ ...p, page }))}
          />
        </>
      )}
    </div>
  );
}

interface HistoryPaginationProps {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}

function HistoryPagination({ page, totalPages, onPageChange }: HistoryPaginationProps) {
  const { t } = useTranslation();

  if (totalPages <= 1) return null;

  return (
    <div className="flex items-center justify-center gap-4">
      <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => onPageChange(page - 1)}>
        {t('pagination.previous')}
      </Button>
      <span className="text-muted-foreground text-sm">
        {t('pagination.page')} {page} {t('pagination.of')} {totalPages}
      </span>
      <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => onPageChange(page + 1)}>
        {t('pagination.next')}
      </Button>
    </div>
  );
}

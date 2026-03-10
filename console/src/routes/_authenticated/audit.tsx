import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { AlertTriangle } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { ListAuditErrorsParams } from '@/api/audit';

import { AuditFilters } from '@/components/audit/audit-filters';
import { AuditTable } from '@/components/audit/audit-table';
import { EmptyState } from '@/components/shared/empty-state';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { PageHeader } from '@/components/shared/page-header';
import { Button } from '@/components/ui/button';
import { auditQueries } from '@/queries/audit-queries';

export const Route = createFileRoute('/_authenticated/audit')({
  component: AuditPage,
  pendingComponent: () => <LoadingSkeleton rows={8} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function AuditPage() {
  const { t } = useTranslation('audit');
  const [params, setParams] = useState<ListAuditErrorsParams>({
    page: 1,
    pageSize: 25,
  });
  const { data } = useSuspenseQuery(auditQueries.errors(params));

  const errors = data.data;
  const pagination = data.pagination;

  return (
    <div className="space-y-6">
      <PageHeader title={t('title')} description={t('description')} />
      <AuditFilters params={params} onChange={setParams} />

      {errors.length === 0 ? (
        <EmptyState
          icon={<AlertTriangle className="h-10 w-10" />}
          title={t('empty.title')}
          description={t('empty.description')}
        />
      ) : (
        <>
          <AuditTable errors={errors} />
          <AuditPagination
            page={pagination.page}
            totalPages={pagination.totalPages}
            onPageChange={(page) => setParams((p) => ({ ...p, page }))}
          />
        </>
      )}
    </div>
  );
}

interface AuditPaginationProps {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}

function AuditPagination({ page, totalPages, onPageChange }: AuditPaginationProps) {
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

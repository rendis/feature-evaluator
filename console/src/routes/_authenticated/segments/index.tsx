import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { FolderOpen, Plus } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { ListSegmentsParams } from '@/api/segments';

import { SegmentForm } from '@/components/segments/segment-form';
import { SegmentList } from '@/components/segments/segment-list';
import { EmptyState } from '@/components/shared/empty-state';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { PageHeader } from '@/components/shared/page-header';
import { PermissionButton } from '@/components/shared/permission-button';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { useDebounce } from '@/hooks/use-debounce';
import { segmentQueries } from '@/queries/segment-queries';

export const Route = createFileRoute('/_authenticated/segments/')({
  component: SegmentsPage,
  pendingComponent: () => <LoadingSkeleton rows={5} />,
  errorComponent: ({ error }) => <ApiErrorState error={error} />,
});

export function SegmentsPage() {
  const { t } = useTranslation('segments');
  const navigate = useNavigate();
  const [params, setParams] = useState<ListSegmentsParams>({ page: 1, pageSize: 20 });
  const [search, setSearch] = useState('');
  const debouncedSearch = useDebounce(search, 300);
  const [formOpen, setFormOpen] = useState(false);
  const { data } = useSuspenseQuery(segmentQueries.list(params));

  const segments = data.data;
  const pagination = data.pagination;
  const openCreateForm = () => setFormOpen(true);
  const handlePageChange = (page: number) => {
    setParams((previous) => ({ ...previous, page }));
  };

  useEffect(() => {
    const normalizedSearch = debouncedSearch.trim() || undefined;
    setParams((previous) =>
      previous.search === normalizedSearch
        ? previous
        : { ...previous, page: 1, search: normalizedSearch },
    );
  }, [debouncedSearch]);

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('title')}
        description={t('description')}
        actions={
          <PermissionButton permission="segments.write" onClick={openCreateForm}>
            <Plus className="mr-2 h-4 w-4" />
            {t('createSegment')}
          </PermissionButton>
        }
      />

      <Input
        placeholder={t('search', { ns: 'common', defaultValue: 'Buscar...' })}
        value={search}
        onChange={(event) => setSearch(event.target.value)}
        className="max-w-xs"
      />

      {segments.length === 0 ? (
        <EmptyState
          icon={<FolderOpen className="h-10 w-10" />}
          title={t('empty.title')}
          description={t('empty.description')}
          action={
            <PermissionButton permission="segments.write" onClick={openCreateForm}>
              {t('createSegment')}
            </PermissionButton>
          }
        />
      ) : (
        <>
          <SegmentList segments={segments} />
          {pagination.totalPages > 1 ? (
            <SegmentPagination
              page={pagination.page}
              totalPages={pagination.totalPages}
              onPageChange={handlePageChange}
            />
          ) : null}
        </>
      )}

      <SegmentForm
        open={formOpen}
        onOpenChange={setFormOpen}
        onSuccess={(key) => navigate({ to: '/segments/$segmentKey', params: { segmentKey: key } })}
      />
    </div>
  );
}

function SegmentPagination({
  page,
  totalPages,
  onPageChange,
}: {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="flex items-center justify-center gap-4">
      <Button
        variant="outline"
        size="sm"
        disabled={page <= 1}
        onClick={() => onPageChange(page - 1)}
      >
        {t('pagination.previous')}
      </Button>
      <span className="text-muted-foreground text-sm">
        {t('pagination.page')} {page} {t('pagination.of')} {totalPages}
      </span>
      <Button
        variant="outline"
        size="sm"
        disabled={page >= totalPages}
        onClick={() => onPageChange(page + 1)}
      >
        {t('pagination.next')}
      </Button>
    </div>
  );
}

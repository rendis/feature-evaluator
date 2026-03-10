import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { Package, Plus } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { PackForm } from '@/components/packs/pack-form';
import { PackList } from '@/components/packs/pack-list';
import { EmptyState } from '@/components/shared/empty-state';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { PageHeader } from '@/components/shared/page-header';
import { PermissionButton } from '@/components/shared/permission-button';
import { Input } from '@/components/ui/input';
import { packQueries } from '@/queries/pack-queries';

export const Route = createFileRoute('/_authenticated/settings/packs/')({
  component: PacksPage,
  pendingComponent: () => <LoadingSkeleton rows={5} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function PacksPage() {
  const { t } = useTranslation('packs');
  const navigate = useNavigate();
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(1);
  const [formOpen, setFormOpen] = useState(false);

  const { data } = useSuspenseQuery(packQueries.list({ search, page, pageSize: 20 }));
  const packs = data.data;
  const pagination = data.pagination;

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('title')}
        description={t('description')}
        actions={
          <PermissionButton permission="features.write" onClick={() => setFormOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t('createPack')}
          </PermissionButton>
        }
      />

      <Input
        placeholder={t('search')}
        value={search}
        onChange={(e) => {
          setSearch(e.target.value);
          setPage(1);
        }}
        className="max-w-sm"
      />

      {packs.length === 0 ? (
        <EmptyState
          icon={<Package className="h-10 w-10" />}
          title={t('empty.title')}
          description={t('empty.description')}
          action={
            <PermissionButton permission="features.write" onClick={() => setFormOpen(true)}>
              {t('createPack')}
            </PermissionButton>
          }
        />
      ) : (
        <>
          <PackList packs={packs} />
          {pagination.totalPages > 1 ? (
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground text-sm">
                {t('page', { ns: 'common', defaultValue: 'Page' })} {pagination.page}{' '}
                {t('of', { ns: 'common', defaultValue: 'of' })} {pagination.totalPages}
              </span>
              <div className="flex gap-2">
                <button
                  type="button"
                  disabled={pagination.page <= 1}
                  onClick={() => setPage((p) => p - 1)}
                  className="text-sm disabled:opacity-50"
                >
                  {t('previous', { ns: 'common', defaultValue: 'Previous' })}
                </button>
                <button
                  type="button"
                  disabled={pagination.page >= pagination.totalPages}
                  onClick={() => setPage((p) => p + 1)}
                  className="text-sm disabled:opacity-50"
                >
                  {t('next', { ns: 'common', defaultValue: 'Next' })}
                </button>
              </div>
            </div>
          ) : null}
        </>
      )}

      <PackForm
        open={formOpen}
        onOpenChange={setFormOpen}
        onSuccess={(key) => navigate({ to: '/settings/packs/$packKey', params: { packKey: key } })}
      />
    </div>
  );
}

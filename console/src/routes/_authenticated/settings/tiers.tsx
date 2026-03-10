import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { Plus } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { Tier } from '@/api/types';

import { EmptyState } from '@/components/shared/empty-state';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { PageHeader } from '@/components/shared/page-header';
import { PermissionButton } from '@/components/shared/permission-button';
import { TierForm } from '@/components/tiers/tier-form';
import { TierList } from '@/components/tiers/tier-list';
import { tierQueries } from '@/queries/tier-queries';

export const Route = createFileRoute('/_authenticated/settings/tiers')({
  component: TiersPage,
  pendingComponent: () => <LoadingSkeleton rows={5} />,
  errorComponent: ({ error }) => <ApiErrorState error={error} />,
});

function TiersPage() {
  const { t } = useTranslation('tiers');
  const [formOpen, setFormOpen] = useState(false);
  const [editTier, setEditTier] = useState<Tier | null>(null);
  const { data: tiers } = useSuspenseQuery(tierQueries.list());

  const handleEdit = (tier: Tier) => {
    setEditTier(tier);
    setFormOpen(true);
  };

  const handleOpenChange = (open: boolean) => {
    setFormOpen(open);
    if (!open) {
      setEditTier(null);
    }
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('title')}
        description={t('description')}
        actions={
          <PermissionButton permission="settings.manage" onClick={() => setFormOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t('createTier')}
          </PermissionButton>
        }
      />

      {tiers.length === 0 ? (
        <EmptyState title={t('empty.title')} description={t('empty.description')} />
      ) : (
        <TierList tiers={tiers} onEdit={handleEdit} />
      )}

      <TierForm open={formOpen} onOpenChange={handleOpenChange} tier={editTier} />
    </div>
  );
}

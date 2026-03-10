import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { Plus } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { ApiKeyCreatedDialog } from '@/components/settings/api-key-created-dialog';
import { ApiKeyForm } from '@/components/settings/api-key-form';
import { ApiKeyTable } from '@/components/settings/api-key-table';
import { EmptyState } from '@/components/shared/empty-state';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { PageHeader } from '@/components/shared/page-header';
import { PermissionButton } from '@/components/shared/permission-button';
import { apiKeyQueries } from '@/queries/api-key-queries';

export const Route = createFileRoute('/_authenticated/settings/api-keys')({
  component: ApiKeysPage,
  pendingComponent: () => <LoadingSkeleton rows={5} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function ApiKeysPage() {
  const { t } = useTranslation('settings');
  const [formOpen, setFormOpen] = useState(false);
  const [createdKey, setCreatedKey] = useState<string | null>(null);
  const { data: apiKeys } = useSuspenseQuery(apiKeyQueries.list());

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('apiKeys.title')}
        description={t('apiKeys.subtitle')}
        actions={
          <PermissionButton permission="members.manage" onClick={() => setFormOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t('apiKeys.create')}
          </PermissionButton>
        }
      />

      {apiKeys.length === 0 ? (
        <EmptyState
          title={t('apiKeys.empty.title')}
          description={t('apiKeys.empty.description')}
        />
      ) : (
        <ApiKeyTable apiKeys={apiKeys} />
      )}

      <ApiKeyForm open={formOpen} onOpenChange={setFormOpen} onCreated={setCreatedKey} />

      {createdKey ? (
        <ApiKeyCreatedDialog
          open={!!createdKey}
          rawKey={createdKey}
          onDismiss={() => setCreatedKey(null)}
        />
      ) : null}
    </div>
  );
}

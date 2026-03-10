import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { DatabaseZap, Pencil, Plus, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import type { ExternalApi } from '@/api/types';

import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { EmptyState } from '@/components/shared/empty-state';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { PageHeader } from '@/components/shared/page-header';
import { PermissionButton } from '@/components/shared/permission-button';
import { Badge } from '@/components/ui/badge';
import { getVisibleErrorMessage } from '@/lib/display-error';
import { useDeleteExternalApi } from '@/mutations/external-api-mutations';
import { externalApiQueries } from '@/queries/external-api-queries';

export const Route = createFileRoute('/_authenticated/settings/external-apis/')({
  component: ExternalApisPage,
  pendingComponent: () => <LoadingSkeleton rows={5} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function ExternalApisPage() {
  const { t } = useTranslation('external-apis');
  const navigate = useNavigate();
  const { data: externalApis } = useSuspenseQuery(externalApiQueries.list());
  const deleteExternalApi = useDeleteExternalApi();
  const [toDelete, setToDelete] = useState<ExternalApi | null>(null);

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('listTitle')}
        description={t('subtitle')}
        actions={
          <PermissionButton
            permission="settings.manage"
            onClick={() => void navigate({ to: '/settings/external-apis/new' })}
          >
            <Plus className="mr-2 h-4 w-4" />
            {t('actions.create')}
          </PermissionButton>
        }
      />

      {externalApis.length === 0 ? (
        <EmptyState title={t('empty.listTitle')} description={t('empty.listDescription')} />
      ) : (
        <div className="grid gap-4">
          {externalApis.map((api) => (
            <article key={api.key} className="rounded-xl border bg-card p-5 shadow-sm">
              <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div className="space-y-3">
                  <div className="flex items-center gap-2">
                    <DatabaseZap className="text-primary h-4 w-4" />
                    <h3 className="font-semibold">{api.name}</h3>
                    <Badge variant="outline">{api.request.method}</Badge>
                    <Badge variant={api.active ? 'success' : 'secondary'}>
                      {api.active ? t('badges.active') : t('badges.inactive')}
                    </Badge>
                    {api.hasSecrets ? (
                      <Badge variant="secondary">{t('badges.hasSecrets')}</Badge>
                    ) : null}
                  </div>
                  <p className="text-muted-foreground font-mono text-xs">{api.key}</p>
                  <p className="font-mono text-sm">{api.request.urlTemplate}</p>
                  <div className="text-muted-foreground flex flex-wrap gap-3 text-sm">
                    <span>{t('list.params', { count: api.params.length })}</span>
                    <span>{t('list.headers', { count: api.request.headers.length })}</span>
                    <span>{t('list.version', { version: api.version })}</span>
                  </div>
                </div>
                <div className="flex flex-wrap gap-2">
                  <PermissionButton
                    permission="settings.manage"
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() =>
                      void navigate({
                        to: '/settings/external-apis/$key',
                        params: { key: api.key },
                      })
                    }
                  >
                    <Pencil className="mr-2 h-4 w-4" />
                    {t('actions.edit')}
                  </PermissionButton>
                  <PermissionButton
                    permission="settings.manage"
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => setToDelete(api)}
                  >
                    <Trash2 className="mr-2 h-4 w-4" />
                    {t('actions.delete')}
                  </PermissionButton>
                </div>
              </div>
            </article>
          ))}
        </div>
      )}

      <ConfirmDialog
        open={toDelete != null}
        onOpenChange={(open) => {
          if (!open) {
            setToDelete(null);
          }
        }}
        title={t('delete.title')}
        description={t('delete.description', { key: toDelete?.key ?? '' })}
        variant="destructive"
        onConfirm={() => {
          if (!toDelete) return;
          deleteExternalApi.mutate(toDelete.key, {
            onSuccess: () => {
              toast.success(t('messages.deleted'));
              setToDelete(null);
            },
            onError: (error) =>
              toast.error(getVisibleErrorMessage(error, t('messages.deleteError'))),
          });
        }}
        loading={deleteExternalApi.isPending}
      />
    </div>
  );
}

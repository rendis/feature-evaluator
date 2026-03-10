import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { KeyRound, Pencil, Plus, ShieldCheck, Sparkles, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import type { AuthProfile } from '@/api/types';

import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { EmptyState } from '@/components/shared/empty-state';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { PageHeader } from '@/components/shared/page-header';
import { PermissionButton } from '@/components/shared/permission-button';
import { Badge } from '@/components/ui/badge';
import { useDeleteAuthProfile } from '@/mutations/auth-profile-mutations';
import { authProfileQueries } from '@/queries/auth-profile-queries';

export const Route = createFileRoute('/_authenticated/settings/auth-profiles/')({
  component: AuthProfilesPage,
  pendingComponent: () => <LoadingSkeleton rows={5} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function AuthProfilesPage() {
  const { t } = useTranslation('settings');
  const navigate = useNavigate();
  const { data: authProfiles } = useSuspenseQuery(authProfileQueries.list());
  const deleteAuthProfile = useDeleteAuthProfile();
  const [profileToDelete, setProfileToDelete] = useState<AuthProfile | null>(null);

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('authProfiles.title')}
        description={t('authProfiles.subtitle')}
        actions={
          <PermissionButton
            permission="settings.manage"
            onClick={() => void navigate({ to: '/settings/auth-profiles/new' })}
          >
            <Plus className="mr-2 h-4 w-4" />
            {t('authProfiles.create')}
          </PermissionButton>
        }
      />

      {authProfiles.length === 0 ? (
        <EmptyState
          title={t('authProfiles.empty.title')}
          description={t('authProfiles.empty.description')}
        />
      ) : (
        <div className="grid gap-4">
          {authProfiles.map((profile) => {
            const ProfileIcon = getProfileIcon(profile);
            return (
              <article key={profile.key} className="rounded-xl border bg-card p-5 shadow-sm">
                <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                  <div className="space-y-3">
                    <div className="flex items-center gap-2">
                      <ProfileIcon className="text-primary h-4 w-4" />
                      <h3 className="font-semibold">{profile.name}</h3>
                      <Badge variant={profile.active ? 'success' : 'secondary'}>
                        {profile.active ? 'Activo' : 'Inactivo'}
                      </Badge>
                    </div>
                    <p className="text-muted-foreground font-mono text-xs">{profile.key}</p>
                    <p className="text-sm">{describeProfile(profile)}</p>
                    <div className="text-muted-foreground flex flex-wrap gap-3 text-sm">
                      <span>{profile.type}</span>
                      <span>
                        {profile.cacheTTLSeconds
                          ? `cache ${profile.cacheTTLSeconds}s`
                          : 'sin cache'}
                      </span>
                      <span>{profile.hasSecret ? 'secreto configurado' : 'sin secreto'}</span>
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
                          to: '/settings/auth-profiles/$key',
                          params: { key: profile.key },
                        })
                      }
                    >
                      <Pencil className="mr-2 h-4 w-4" />
                      {t('actions.edit', { ns: 'common' })}
                    </PermissionButton>
                    <PermissionButton
                      permission="settings.manage"
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => setProfileToDelete(profile)}
                    >
                      <Trash2 className="mr-2 h-4 w-4" />
                      {t('actions.delete', { ns: 'common' })}
                    </PermissionButton>
                  </div>
                </div>
              </article>
            );
          })}
        </div>
      )}

      <ConfirmDialog
        open={!!profileToDelete}
        onOpenChange={(open) => {
          if (!open) setProfileToDelete(null);
        }}
        title={t('authProfiles.deleteTitle')}
        description={t('authProfiles.deleteDescription', { key: profileToDelete?.key ?? '' })}
        variant="destructive"
        onConfirm={() => {
          if (!profileToDelete) return;
          deleteAuthProfile.mutate(profileToDelete.key, {
            onSuccess: () => {
              toast.success(t('authProfiles.deleteSuccess'));
              setProfileToDelete(null);
            },
            onError: () => toast.error(t('authProfiles.deleteError')),
          });
        }}
        loading={deleteAuthProfile.isPending}
      />
    </div>
  );
}

function getProfileIcon(profile: AuthProfile) {
  if (profile.type === 'api_key') {
    return KeyRound;
  }
  if (profile.type === 'oidc_standard') {
    return ShieldCheck;
  }
  return Sparkles;
}

function describeProfile(profile: AuthProfile) {
  if (profile.type === 'api_key') {
    return `Compara una API key fija recibida por ${
      profile.config.location === 'query' ? 'query param' : 'header'
    } (${String(profile.config.name ?? 'sin nombre')}).`;
  }
  if (profile.type === 'oidc_standard') {
    return `Valida el bearer token con issuer ${String(profile.config.issuer ?? 'configurado')} y audience ${String(profile.config.audience ?? 'configurada')}.`;
  }
  return `Valida vía request custom hacia ${String(profile.config.url ?? 'la URL configurada')}.`;
}

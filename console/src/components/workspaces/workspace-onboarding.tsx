import { Building2, RotateCcw } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import type { Workspace } from '@/api/workspaces';

import { Button } from '@/components/ui/button';
import { WorkspaceFormDialog } from '@/components/workspaces/workspace-form-dialog';
import { useRestoreWorkspace } from '@/mutations/workspace-mutations';
import { useWorkspaceStore } from '@/stores/workspace-store';

interface WorkspaceOnboardingProps {
  archivedWorkspaces: Workspace[];
}

export function WorkspaceOnboarding({ archivedWorkspaces }: WorkspaceOnboardingProps) {
  const { t } = useTranslation('workspaces');
  const [open, setOpen] = useState(false);
  const restoreWorkspace = useRestoreWorkspace();
  const setWorkspace = useWorkspaceStore((state) => state.setWorkspace);

  const handleRestore = (key: string) => {
    restoreWorkspace.mutate(key, {
      onSuccess: () => {
        setWorkspace(key);
        toast.success(t('restore.success'));
      },
      onError: () => toast.error(t('restore.error')),
    });
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4 py-10">
      <div className="w-full max-w-3xl space-y-6">
        <section className="bg-background space-y-6 rounded-xl border px-6 py-10 shadow-sm">
          <div className="flex flex-col items-center text-center">
            <Building2 className="text-primary h-12 w-12" />
            <h1 className="mt-4 text-2xl font-semibold">{t('onboarding.title')}</h1>
            <p className="text-muted-foreground mt-2 max-w-2xl">{t('onboarding.description')}</p>
          </div>
          <div className="flex justify-center">
            <Button onClick={() => setOpen(true)}>{t('onboarding.create')}</Button>
          </div>
        </section>

        {archivedWorkspaces.length > 0 ? (
          <section className="bg-background space-y-4 rounded-xl border px-6 py-6 shadow-sm">
            <div>
              <h2 className="text-lg font-semibold">{t('sections.archived')}</h2>
              <p className="text-muted-foreground mt-1">{t('onboarding.restoreDescription')}</p>
            </div>
            <div className="space-y-3">
              {archivedWorkspaces.map((workspace) => (
                <div
                  key={workspace.key}
                  className="flex items-center justify-between rounded-md border p-4"
                >
                  <div>
                    <p className="font-medium">{workspace.name}</p>
                    <p className="text-muted-foreground text-sm">{workspace.key}</p>
                  </div>
                  <Button
                    variant="outline"
                    onClick={() => handleRestore(workspace.key)}
                    disabled={restoreWorkspace.isPending}
                  >
                    <RotateCcw className="mr-2 h-4 w-4" />
                    {t('restore.action')}
                  </Button>
                </div>
              ))}
            </div>
          </section>
        ) : null}

        <WorkspaceFormDialog
          open={open}
          onOpenChange={setOpen}
          onSuccess={(workspace) => setWorkspace(workspace.key)}
        />
      </div>
    </div>
  );
}

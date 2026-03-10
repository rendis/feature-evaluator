import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { Archive, Building2, Plus, RotateCcw } from 'lucide-react';
import { useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import type { Workspace } from '@/api/workspaces';

import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { EmptyState } from '@/components/shared/empty-state';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { PageHeader } from '@/components/shared/page-header';
import { PermissionButton } from '@/components/shared/permission-button';
import { Badge } from '@/components/ui/badge';
import { WorkspaceFormDialog } from '@/components/workspaces/workspace-form-dialog';
import { useArchiveWorkspace, useRestoreWorkspace } from '@/mutations/workspace-mutations';
import { workspaceQueries } from '@/queries/workspace-queries';
import { useWorkspaceStore } from '@/stores/workspace-store';

export const Route = createFileRoute('/_authenticated/settings/workspaces')({
  component: WorkspacesPage,
  pendingComponent: () => <LoadingSkeleton rows={5} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function WorkspaceCard({
  children,
  current,
  description,
  name,
  keyValue,
  muted = false,
}: {
  children: ReactNode;
  current?: boolean;
  description?: string;
  keyValue: string;
  muted?: boolean;
  name: string;
}) {
  const { t } = useTranslation('workspaces');

  return (
    <div className={`flex items-center justify-between rounded-md border p-4 ${muted ? 'opacity-80' : ''}`}>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="font-medium">{name}</span>
          <Badge variant="outline" className="font-mono text-xs">
            {keyValue}
          </Badge>
          {current ? <Badge variant="success">{t('current')}</Badge> : null}
        </div>
        {description ? <p className="text-muted-foreground mt-1 text-sm">{description}</p> : null}
      </div>
      {children}
    </div>
  );
}

function WorkspaceSection({
  emptyDescription,
  emptyTitle,
  title,
  workspaces,
}: {
  emptyDescription: string;
  emptyTitle: string;
  title: string;
  workspaces: ReactNode;
}) {
  return (
    <section className="space-y-2">
      <div>
        <h2 className="text-lg font-semibold">{title}</h2>
      </div>
      {workspaces}
      {workspaces ? null : <EmptyState title={emptyTitle} description={emptyDescription} />}
    </section>
  );
}

function ActiveWorkspacesSection({
  currentKey,
  onArchive,
  workspaces,
}: {
  currentKey: string;
  onArchive: (workspace: Workspace) => void;
  workspaces: Workspace[];
}) {
  const { t } = useTranslation('workspaces');
  const cards =
    workspaces.length > 0
      ? workspaces.map((workspace) => (
          <WorkspaceCard
            key={workspace.key}
            current={workspace.key === currentKey}
            description={workspace.description}
            keyValue={workspace.key}
            name={workspace.name}
          >
            <PermissionButton
              permission="workspace.delete"
              variant="ghost"
              size="icon"
              className="text-destructive"
              onClick={() => onArchive(workspace)}
            >
              <Archive className="h-4 w-4" />
            </PermissionButton>
          </WorkspaceCard>
        ))
      : null;

  return (
    <WorkspaceSection
      emptyDescription={t('empty.activeDescription')}
      emptyTitle={t('empty.activeTitle')}
      title={t('sections.active')}
      workspaces={cards}
    />
  );
}

function ArchivedWorkspacesSection({
  onRestore,
  restorePending,
  workspaces,
}: {
  onRestore: (key: string) => void;
  restorePending: boolean;
  workspaces: Workspace[];
}) {
  const { t } = useTranslation('workspaces');
  const cards =
    workspaces.length > 0
      ? workspaces.map((workspace) => (
          <WorkspaceCard
            key={workspace.key}
            description={workspace.description}
            keyValue={workspace.key}
            muted
            name={workspace.name}
          >
            <PermissionButton
              permission="workspace.delete"
              variant="outline"
              size="sm"
              onClick={() => onRestore(workspace.key)}
              disabled={restorePending}
            >
              <RotateCcw className="mr-2 h-4 w-4" />
              {t('restore.action')}
            </PermissionButton>
          </WorkspaceCard>
        ))
      : null;

  return (
    <WorkspaceSection
      emptyDescription={t('empty.archivedDescription')}
      emptyTitle={t('empty.archivedTitle')}
      title={t('sections.archived')}
      workspaces={cards}
    />
  );
}

function WorkspaceDialogs({
  archivePending,
  archiveTarget,
  formOpen,
  onArchiveOpenChange,
  onConfirmArchive,
  onFormOpenChange,
}: {
  archivePending: boolean;
  archiveTarget: Workspace | null;
  formOpen: boolean;
  onArchiveOpenChange: (open: boolean) => void;
  onConfirmArchive: () => void;
  onFormOpenChange: (open: boolean) => void;
}) {
  const { t } = useTranslation('workspaces');

  return (
    <>
      <WorkspaceFormDialog open={formOpen} onOpenChange={onFormOpenChange} />
      <ConfirmDialog
        open={!!archiveTarget}
        onOpenChange={onArchiveOpenChange}
        title={t('archive.title')}
        description={t('archive.description', { name: archiveTarget?.name })}
        variant="destructive"
        onConfirm={onConfirmArchive}
        loading={archivePending}
      />
    </>
  );
}

function WorkspacesPage() {
  const { t } = useTranslation('workspaces');
  const [formOpen, setFormOpen] = useState(false);
  const [archiveTarget, setArchiveTarget] = useState<Workspace | null>(null);
  const { data: workspaces } = useSuspenseQuery(workspaceQueries.list({ includeArchived: true }));
  const archiveWorkspace = useArchiveWorkspace();
  const restoreWorkspace = useRestoreWorkspace();
  const currentKey = useWorkspaceStore((s) => s.workspaceKey);
  const setWorkspace = useWorkspaceStore((s) => s.setWorkspace);
  const activeWorkspaces = workspaces.filter((workspace) => !workspace.archivedAt);
  const archivedWorkspaces = workspaces.filter((workspace) => !!workspace.archivedAt);

  const handleArchive = () => {
    if (!archiveTarget) return;
    archiveWorkspace.mutate(archiveTarget.key, {
      onSuccess: () => {
        toast.success(t('archive.success'));
        if (archiveTarget.key === currentKey) {
          const remaining = activeWorkspaces.filter(
            (workspace) => workspace.key !== archiveTarget.key,
          );
          setWorkspace(remaining[0]?.key ?? '');
        }
        setArchiveTarget(null);
      },
      onError: () => toast.error(t('archive.error')),
    });
  };

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
    <div className="space-y-6">
      <PageHeader
        title={t('title')}
        description={t('description')}
        actions={
          <PermissionButton permission="settings.manage" onClick={() => setFormOpen(true)}>
            <Plus className="mr-2 h-4 w-4" />
            {t('create')}
          </PermissionButton>
        }
      />

      {workspaces.length === 0 ? (
        <EmptyState
          icon={<Building2 className="h-10 w-10" />}
          title={t('empty.title')}
          description={t('empty.description')}
        />
      ) : (
        <div className="space-y-6">
          <ActiveWorkspacesSection
            currentKey={currentKey}
            onArchive={setArchiveTarget}
            workspaces={activeWorkspaces}
          />
          <ArchivedWorkspacesSection
            onRestore={handleRestore}
            restorePending={restoreWorkspace.isPending}
            workspaces={archivedWorkspaces}
          />
        </div>
      )}

      <WorkspaceDialogs
        archivePending={archiveWorkspace.isPending}
        archiveTarget={archiveTarget}
        formOpen={formOpen}
        onArchiveOpenChange={(open) => !open && setArchiveTarget(null)}
        onConfirmArchive={handleArchive}
        onFormOpenChange={setFormOpen}
      />
    </div>
  );
}

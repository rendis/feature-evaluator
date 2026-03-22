import { useQuery } from '@tanstack/react-query';
import { Navigate, Outlet, createFileRoute } from '@tanstack/react-router';
import { useEffect } from 'react';

import { AppShell } from '@/components/layout/app-shell';
import { FullScreenSpinner } from '@/components/shared/full-screen-spinner';
import { WorkspaceOnboarding } from '@/components/workspaces/workspace-onboarding';
import { useAuth } from '@/hooks/use-auth';
import { workspaceQueries } from '@/queries/workspace-queries';
import { useWorkspaceStore } from '@/stores/workspace-store';

export const Route = createFileRoute('/_authenticated')({
  component: AuthenticatedLayout,
});

function AuthenticatedLayout() {
  const { isAuthenticated, isLoading: authLoading, member, isMemberLoading } = useAuth();
  const { workspaceKey, setWorkspace } = useWorkspaceStore();
  const { data: workspaces = [], isLoading } = useQuery({
    ...workspaceQueries.list({ includeArchived: true }),
    enabled: isAuthenticated,
  });

  const activeWorkspaces = workspaces.filter((workspace) => !workspace.archivedAt);
  const archivedWorkspaces = workspaces.filter((workspace) => !!workspace.archivedAt);
  const currentWorkspace = activeWorkspaces.find((workspace) => workspace.key === workspaceKey);

  useEffect(() => {
    if (isLoading) return;
    if (activeWorkspaces.length === 0) {
      if (workspaceKey) setWorkspace('');
      return;
    }
    if (!currentWorkspace) {
      setWorkspace(activeWorkspaces[0].key);
    }
  }, [activeWorkspaces, currentWorkspace, isLoading, setWorkspace, workspaceKey]);

  if (authLoading) {
    return <FullScreenSpinner />;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" />;
  }

  if (isLoading || (activeWorkspaces.length > 0 && ((!member && isMemberLoading) || !currentWorkspace))) {
    return <FullScreenSpinner />;
  }

  if (activeWorkspaces.length === 0) {
    return <WorkspaceOnboarding archivedWorkspaces={archivedWorkspaces} />;
  }

  if (!member) {
    return <Navigate to="/auth/access-denied" />;
  }

  return (
    <AppShell>
      <Outlet />
    </AppShell>
  );
}

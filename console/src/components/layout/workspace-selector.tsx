import { useQuery } from '@tanstack/react-query';
import { ChevronsUpDown } from 'lucide-react';
import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { workspaceQueries } from '@/queries/workspace-queries';
import { useWorkspaceStore } from '@/stores/workspace-store';

interface WorkspaceSelectorProps {
  collapsed: boolean;
}

export function WorkspaceSelector({ collapsed }: WorkspaceSelectorProps) {
  const { t } = useTranslation('workspaces');
  const { workspaceKey, setWorkspace } = useWorkspaceStore();
  const { data: workspaces = [] } = useQuery(workspaceQueries.list());
  const activeWorkspace = workspaces.find((workspace) => workspace.key === workspaceKey) ?? workspaces[0];

  useEffect(() => {
    if (workspaces.length === 0) return;
    if (activeWorkspace) {
      if (activeWorkspace.key !== workspaceKey) {
        setWorkspace(activeWorkspace.key);
      }
      return;
    }
    setWorkspace(workspaces[0].key);
  }, [activeWorkspace, setWorkspace, workspaceKey, workspaces]);

  const handleChange = (key: string) => {
    if (key === workspaceKey) return;
    setWorkspace(key);
  };

  if (workspaces.length === 0) return null;

  if (collapsed) {
    return (
      <button
        type="button"
        onClick={() => {
          const idx = workspaces.findIndex((w) => w.key === activeWorkspace?.key);
          const next = workspaces[(idx + 1) % workspaces.length];
          handleChange(next.key);
        }}
        className="flex w-full items-center justify-center rounded-md border px-2 py-1.5"
        title={activeWorkspace ? t('selector.switchTo', { name: activeWorkspace.name }) : t('selector.switch')}
      >
        <ChevronsUpDown className="h-4 w-4" />
      </button>
    );
  }

  return (
    <Select value={activeWorkspace?.key ?? ''} onValueChange={handleChange}>
      <SelectTrigger className="h-8 text-xs">
        <SelectValue placeholder={t('selector.placeholder')} />
      </SelectTrigger>
      <SelectContent>
        {workspaces.map((ws) => (
          <SelectItem key={ws.key} value={ws.key}>
            {ws.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

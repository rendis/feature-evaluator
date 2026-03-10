import { create } from 'zustand';

import { queryClient } from '@/config/query-client';

const STORAGE_KEY = 'fe-workspace';

function storage() {
  const candidate = globalThis.localStorage;
  if (
    candidate &&
    typeof candidate.getItem === 'function' &&
    typeof candidate.setItem === 'function' &&
    typeof candidate.removeItem === 'function'
  ) {
    return candidate;
  }
  return null;
}

function getInitialWorkspaceKey() {
  return storage()?.getItem(STORAGE_KEY) ?? '';
}

interface WorkspaceState {
  workspaceKey: string;
  setWorkspace: (key: string) => void;
}

export const useWorkspaceStore = create<WorkspaceState>()((set) => ({
  workspaceKey: getInitialWorkspaceKey(),
  setWorkspace: (key) => {
    if (key) {
      storage()?.setItem(STORAGE_KEY, key);
    } else {
      storage()?.removeItem(STORAGE_KEY);
    }
    set({ workspaceKey: key });
    queryClient.clear();
  },
}));

/** Get the current workspace key (non-reactive, for use outside React). */
export function getWorkspaceKey(): string {
  return useWorkspaceStore.getState().workspaceKey;
}

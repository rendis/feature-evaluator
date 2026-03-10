import { create } from 'zustand';

const COLLAPSED_KEY = 'sidebar-collapsed';

interface SidebarState {
  open: boolean;
  collapsed: boolean;
  setOpen: (open: boolean) => void;
  toggle: () => void;
  toggleCollapsed: () => void;
}

export const useSidebarStore = create<SidebarState>()((set) => ({
  open: false,
  collapsed: localStorage.getItem(COLLAPSED_KEY) === 'true',
  setOpen: (open) => set({ open }),
  toggle: () => set((s) => ({ open: !s.open })),
  toggleCollapsed: () =>
    set((s) => {
      const next = !s.collapsed;
      localStorage.setItem(COLLAPSED_KEY, String(next));
      return { collapsed: next };
    }),
}));

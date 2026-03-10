import { AppHeader } from './app-header';
import { AppSidebar } from './app-sidebar';

import type { ReactNode } from 'react';

export function AppShell({ children }: { children: ReactNode }) {
  return (
    <div className="flex h-screen overflow-hidden bg-surface-canvas text-foreground">
      <AppSidebar />
      <div className="flex flex-1 flex-col overflow-hidden bg-surface-canvas">
        <AppHeader />
        <main className="flex-1 overflow-auto bg-surface-canvas p-4 lg:p-6">{children}</main>
      </div>
    </div>
  );
}

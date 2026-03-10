import { Menu } from 'lucide-react';

import { LanguageToggle } from './language-toggle';
import { ThemeToggle } from './theme-toggle';
import { UserMenu } from './user-menu';

import { Button } from '@/components/ui/button';
import { useMobile } from '@/hooks/use-mobile';
import { useSidebarStore } from '@/stores/sidebar-store';

export function AppHeader() {
  const isMobile = useMobile();
  const { toggle } = useSidebarStore();

  return (
    <header className="flex h-14 items-center gap-4 border-b border-border-strong bg-surface-canvas/90 px-4 backdrop-blur-sm lg:px-6">
      {isMobile ? (
        <Button variant="ghost" size="icon" onClick={toggle}>
          <Menu className="h-5 w-5" />
        </Button>
      ) : null}
      <div className="flex-1" />
      <div className="flex items-center gap-1">
        <LanguageToggle />
        <ThemeToggle />
        <UserMenu />
      </div>
    </header>
  );
}

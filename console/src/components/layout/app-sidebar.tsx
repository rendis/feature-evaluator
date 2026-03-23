import { Link, useRouterState } from '@tanstack/react-router';
import {
  Building2,
  ChevronLeft,
  ChevronRight,
  DatabaseZap,
  FileWarning,
  FlaskConical,
  History,
  Key,
  LayoutDashboard,
  Package,
  Settings,
  Shield,
  ShieldCheck,
  ToggleLeft,
  Users,
  X,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { ReactNode } from 'react';
import type { Permission } from '@/auth/roles';

import { usePermissions } from '@/hooks/use-permissions';
import { WorkspaceSelector } from '@/components/layout/workspace-selector';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Separator } from '@/components/ui/separator';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { useMobile } from '@/hooks/use-mobile';
import { cn } from '@/lib/utils';
import { useSidebarStore } from '@/stores/sidebar-store';

interface NavItem {
  labelKey: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  permission?: Permission;
}

const navItems: NavItem[] = [
  { labelKey: 'nav.dashboard', href: '/', icon: LayoutDashboard },
  { labelKey: 'nav.features', href: '/features', icon: ToggleLeft },
  { labelKey: 'nav.segments', href: '/segments', icon: Users },
  { labelKey: 'nav.packs', href: '/settings/packs', icon: Package },
  { labelKey: 'nav.experiments', href: '/experiments', icon: FlaskConical },
  { labelKey: 'nav.audit', href: '/audit', icon: FileWarning },
  { labelKey: 'nav.history', href: '/history', icon: History },
];

const configItems: NavItem[] = [
  {
    labelKey: 'nav.security',
    href: '/settings/security',
    icon: Shield,
    permission: 'security.manage',
  },
  { labelKey: 'nav.externalApis', href: '/settings/external-apis', icon: DatabaseZap },
  { labelKey: 'nav.authProfiles', href: '/settings/auth-profiles', icon: ShieldCheck },
  { labelKey: 'nav.members', href: '/settings/members', icon: Settings },
  { labelKey: 'nav.apiKeys', href: '/settings/api-keys', icon: Key },
];

const workspacesItem: NavItem = {
  labelKey: 'nav.workspaces',
  href: '/settings/workspaces',
  icon: Building2,
};

function NavGroup({
  collapsed,
  isActive,
  isMobile,
  items,
  onClose,
  t,
}: {
  collapsed: boolean;
  isActive: (href: string) => boolean;
  isMobile: boolean;
  items: NavItem[];
  onClose: () => void;
  t: (key: string) => string;
}) {
  return (
    <nav className="flex flex-col gap-1">
      {items.map((item) => (
        <NavLink
          key={item.href}
          item={item}
          active={isActive(item.href)}
          collapsed={collapsed}
          onClick={isMobile ? onClose : undefined}
          t={t}
        />
      ))}
    </nav>
  );
}

function NavLink({
  item,
  active,
  collapsed,
  onClick,
  t,
}: {
  item: NavItem;
  active: boolean;
  collapsed: boolean;
  onClick?: () => void;
  t: (key: string) => string;
}) {
  const link = (
    <Link
      to={item.href}
      onClick={onClick}
      className={cn(
        'flex min-h-[44px] items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
        active
          ? 'border border-sidebar-border bg-sidebar-accent text-sidebar-primary shadow-sm'
          : 'text-sidebar-foreground',
        collapsed && 'justify-center px-0',
      )}
    >
      <item.icon className="h-4 w-4 shrink-0" />
      {!collapsed && <span>{t(item.labelKey)}</span>}
    </Link>
  );

  if (collapsed) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>{link}</TooltipTrigger>
        <TooltipContent side="right">{t(item.labelKey)}</TooltipContent>
      </Tooltip>
    );
  }

  return link;
}

function SidebarContent({
  collapsed,
  currentPath,
  isMobile,
  onClose,
  t,
  toggleCollapsed,
}: {
  collapsed: boolean;
  currentPath: string;
  isMobile: boolean;
  onClose: () => void;
  t: (key: string) => string;
  toggleCollapsed: () => void;
}) {
  const { can } = usePermissions();
  const isCollapsed = collapsed && !isMobile;
  const isActive = (href: string) => {
    if (href === '/') return currentPath === '/';
    return currentPath.startsWith(href);
  };
  const visibleNavItems = navItems.filter((item) => !item.permission || can(item.permission));
  const visibleConfigItems = configItems.filter((item) => !item.permission || can(item.permission));

  return (
    <div className="flex h-full flex-col">
      <div
        className={cn(
          'flex h-14 items-center border-b border-sidebar-border px-4',
          isCollapsed ? 'justify-center px-2' : 'justify-between',
        )}
      >
        {isCollapsed ? (
          <span className="text-lg font-semibold">F</span>
        ) : (
          <span className="text-lg font-semibold">{t('app.name')}</span>
        )}
        {isMobile ? (
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="h-4 w-4" />
          </Button>
        ) : null}
      </div>
      <div className={cn('border-b border-sidebar-border px-3 py-2', isCollapsed && 'px-1')}>
        <WorkspaceSelector collapsed={isCollapsed} />
      </div>
      <ScrollArea className={cn('flex-1 py-4', isCollapsed ? 'px-1' : 'px-3')}>
        <NavGroup
          collapsed={isCollapsed}
          isActive={isActive}
          isMobile={isMobile}
          items={visibleNavItems}
          onClose={onClose}
          t={t}
        />
        <Separator className="my-4" />
        {!isCollapsed ? (
          <p className="text-muted-foreground mb-2 px-3 text-xs font-medium uppercase tracking-wider">
            {t('nav.settings')}
          </p>
        ) : null}
        <NavGroup
          collapsed={isCollapsed}
          isActive={isActive}
          isMobile={isMobile}
          items={visibleConfigItems}
          onClose={onClose}
          t={t}
        />
      </ScrollArea>
      <div
        className={cn(
          'border-t border-sidebar-border p-2',
          isCollapsed ? 'flex flex-col items-center gap-1' : 'flex flex-col gap-1',
        )}
      >
        <NavLink
          item={workspacesItem}
          active={isActive(workspacesItem.href)}
          collapsed={isCollapsed}
          onClick={isMobile ? onClose : undefined}
          t={t}
        />
        {!isMobile ? (
          <div className={cn(isCollapsed ? 'flex justify-center' : 'flex justify-end')}>
            <Button variant="ghost" size="icon" onClick={toggleCollapsed}>
              {collapsed ? (
                <ChevronRight className="h-4 w-4" />
              ) : (
                <ChevronLeft className="h-4 w-4" />
              )}
            </Button>
          </div>
        ) : null}
      </div>
    </div>
  );
}

function MobileSidebar({
  children,
  open,
  onClose,
}: {
  children: ReactNode;
  open: boolean;
  onClose: () => void;
}) {
  return (
    <>
      {open ? <div className="fixed inset-0 z-40 bg-surface-overlay" onClick={onClose} /> : null}
      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-50 w-64 transform border-r border-sidebar-border bg-sidebar-background transition-transform duration-200',
          open ? 'translate-x-0' : '-translate-x-full',
        )}
      >
        {children}
      </aside>
    </>
  );
}

function DesktopSidebar({ children, collapsed }: { children: ReactNode; collapsed: boolean }) {
  return (
    <aside
      className={cn(
        'hidden flex-shrink-0 border-r border-sidebar-border bg-sidebar-background transition-[width] duration-200 lg:block',
        collapsed ? 'w-16' : 'w-64',
      )}
    >
      {children}
    </aside>
  );
}

export function AppSidebar() {
  const { t } = useTranslation();
  const isMobile = useMobile();
  const { open, setOpen, collapsed, toggleCollapsed } = useSidebarStore();
  const routerState = useRouterState();
  const currentPath = routerState.location.pathname;

  const content = (
    <SidebarContent
      collapsed={collapsed}
      currentPath={currentPath}
      isMobile={isMobile}
      onClose={() => setOpen(false)}
      t={t}
      toggleCollapsed={toggleCollapsed}
    />
  );

  if (isMobile) {
    return (
      <MobileSidebar open={open} onClose={() => setOpen(false)}>
        {content}
      </MobileSidebar>
    );
  }

  return <DesktopSidebar collapsed={collapsed}>{content}</DesktopSidebar>;
}

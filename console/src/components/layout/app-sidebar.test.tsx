import { within } from '@testing-library/react';

import { AppSidebar } from './app-sidebar';

import type { AnchorHTMLAttributes, ReactNode } from 'react';

import { render } from '@/test/test-utils';

vi.mock('@tanstack/react-router', () => ({
  Link: ({
    to,
    children,
    ...props
  }: {
    to: string;
    children: ReactNode;
  } & AnchorHTMLAttributes<HTMLAnchorElement>) => (
    <a href={to} {...props}>
      {children}
    </a>
  ),
  useRouterState: () => ({ location: { pathname: '/' } }),
}));

vi.mock('@/hooks/use-mobile', () => ({
  useMobile: () => false,
}));

vi.mock('@/components/layout/workspace-selector', () => ({
  WorkspaceSelector: ({ collapsed }: { collapsed: boolean }) => (
    <div data-collapsed={String(collapsed)} data-testid="workspace-selector" />
  ),
}));

vi.mock('@/stores/sidebar-store', () => ({
  useSidebarStore: () => ({
    open: false,
    collapsed: false,
    setOpen: vi.fn(),
    toggleCollapsed: vi.fn(),
  }),
}));

describe('AppSidebar', () => {
  it('renders packs in the primary navigation segment', () => {
    const { container } = render(<AppSidebar />);
    const navSections = container.querySelectorAll('nav');

    expect(navSections).toHaveLength(2);

    const [primaryNav, settingsNav] = Array.from(navSections);
    const packsLink = within(primaryNav).getByRole('link', { name: 'nav.packs' });

    expect(packsLink).toHaveAttribute('href', '/settings/packs');
    expect(within(settingsNav).queryByRole('link', { name: 'nav.packs' })).not.toBeInTheDocument();
  });
});

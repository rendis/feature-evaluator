import * as rq from '@tanstack/react-query';

import { DashboardContent } from './index';

import type { AnchorHTMLAttributes, ReactNode } from 'react';

import { render, screen } from '@/test/test-utils';

vi.mock('@tanstack/react-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-query')>();
  return {
    ...actual,
    useSuspenseQuery: vi.fn(),
  };
});

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
  createFileRoute: () => () => ({}),
}));

vi.mock('@/components/dashboard/stats-grid', () => ({
  StatsGrid: () => <div>stats-grid</div>,
}));

vi.mock('@/components/dashboard/system-operations-panel', () => ({
  SystemOperationsPanel: () => <div>system-operations-panel</div>,
}));

vi.mock('@/components/dashboard/activity-feed', () => ({
  ActivityFeed: () => <div>activity-feed</div>,
}));

vi.mock('@/components/dashboard/error-summary-card', () => ({
  ErrorSummaryCard: () => <div>error-summary-card</div>,
}));

vi.mock('@/components/dashboard/metrics-section', () => ({
  MetricsSection: () => <div>metrics-section</div>,
}));

vi.mock('@/components/shared/permission-button', () => ({
  PermissionButton: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

const mockedUseSuspenseQuery = vi.mocked(rq.useSuspenseQuery);

describe('DashboardContent', () => {
  it('keeps the operations panel visible when there are no features', () => {
    mockedUseSuspenseQuery.mockReturnValue({
      data: {
        totalFeatures: 0,
        activeFeatures: 0,
        totalSegments: 0,
        totalSegmentMembers: 0,
      },
    } as ReturnType<typeof rq.useSuspenseQuery>);

    render(<DashboardContent />);

    expect(screen.getByText('stats-grid')).toBeInTheDocument();
    expect(screen.getByText('system-operations-panel')).toBeInTheDocument();
    expect(screen.getByText('empty.title')).toBeInTheDocument();
    expect(screen.queryByText('activity-feed')).not.toBeInTheDocument();
    expect(screen.queryByText('metrics-section')).not.toBeInTheDocument();
  });
});

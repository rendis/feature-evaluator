import * as rq from '@tanstack/react-query';
import { act } from '@testing-library/react';

import { FeaturesPage } from './index';

import type { ReactNode } from 'react';

import { fireEvent, render, screen } from '@/test/test-utils';
import { featureQueries } from '@/queries/feature-queries';

vi.mock('@tanstack/react-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-query')>();
  return {
    ...actual,
    useQuery: vi.fn(),
    useSuspenseQuery: vi.fn(),
  };
});

vi.mock('@tanstack/react-router', () => ({
  Link: ({ children }: { children: ReactNode }) => <a href="/features/new">{children}</a>,
  createFileRoute: () => () => ({}),
}));

vi.mock('@/components/features/feature-list', () => ({
  FeatureList: () => <div>feature-list</div>,
}));

vi.mock('@/components/shared/page-header', () => ({
  PageHeader: ({
    title,
    description,
    actions,
  }: {
    title: string;
    description: string;
    actions: ReactNode;
  }) => (
    <div>
      <h1>{title}</h1>
      <p>{description}</p>
      {actions}
    </div>
  ),
}));

vi.mock('@/components/shared/permission-button', () => ({
  PermissionButton: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock('@/queries/feature-queries', () => ({
  featureQueries: {
    summaryList: vi.fn((params = {}) => ({ queryKey: ['features', 'list', params] })),
  },
}));

vi.mock('@/queries/tag-queries', () => ({
  tagQueries: {
    list: () => ({ queryKey: ['tags'] }),
  },
}));

const mockedUseQuery = vi.mocked(rq.useQuery);
const mockedUseSuspenseQuery = vi.mocked(rq.useSuspenseQuery);
const mockedSummaryList = vi.mocked(featureQueries.summaryList);

describe('FeaturesPage', () => {
  it('debounces search before rebuilding the summary query', async () => {
    vi.useFakeTimers();

    mockedUseQuery.mockReturnValue({ data: [] } as ReturnType<typeof rq.useQuery>);
    mockedUseSuspenseQuery.mockReturnValue({
      data: {
        data: [],
        pagination: { page: 1, pageSize: 20, total: 0, totalPages: 0 },
      },
    } as ReturnType<typeof rq.useSuspenseQuery>);

    render(<FeaturesPage />, {
      providerProps: { namespaces: ['features', 'common'] },
    });

    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'beta' } });

    expect(mockedSummaryList).not.toHaveBeenCalledWith(expect.objectContaining({ search: 'beta' }));

    act(() => {
      vi.advanceTimersByTime(300);
    });

    expect(mockedSummaryList).toHaveBeenCalledWith({ page: 1, pageSize: 20, search: 'beta' });

    vi.useRealTimers();
  });
});

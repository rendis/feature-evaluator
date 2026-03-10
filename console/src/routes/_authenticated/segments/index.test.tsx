import * as rq from '@tanstack/react-query';
import { act } from '@testing-library/react';

import { SegmentsPage } from './index';

import type { ReactNode } from 'react';

import { fireEvent, render, screen } from '@/test/test-utils';
import { segmentQueries } from '@/queries/segment-queries';

vi.mock('@tanstack/react-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-query')>();
  return {
    ...actual,
    useSuspenseQuery: vi.fn(),
  };
});

vi.mock('@tanstack/react-router', () => ({
  createFileRoute: () => () => ({}),
  useNavigate: () => vi.fn(),
}));

vi.mock('@/components/segments/segment-list', () => ({
  SegmentList: () => <div>segment-list</div>,
}));

vi.mock('@/components/segments/segment-form', () => ({
  SegmentForm: () => null,
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

vi.mock('@/queries/segment-queries', () => ({
  segmentQueries: {
    list: vi.fn((params = {}) => ({ queryKey: ['segments', 'list', params] })),
  },
}));

const mockedUseSuspenseQuery = vi.mocked(rq.useSuspenseQuery);
const mockedSegmentList = vi.mocked(segmentQueries.list);

describe('SegmentsPage', () => {
  it('debounces search before rebuilding the list query', async () => {
    vi.useFakeTimers();

    mockedUseSuspenseQuery.mockReturnValue({
      data: {
        data: [],
        pagination: { page: 1, pageSize: 20, total: 0, totalPages: 0 },
      },
    } as ReturnType<typeof rq.useSuspenseQuery>);

    render(<SegmentsPage />, {
      providerProps: { namespaces: ['segments', 'common'] },
    });

    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'beta' } });

    expect(mockedSegmentList).not.toHaveBeenCalledWith(expect.objectContaining({ search: 'beta' }));

    act(() => {
      vi.advanceTimersByTime(300);
    });

    expect(mockedSegmentList).toHaveBeenCalledWith({ page: 1, pageSize: 20, search: 'beta' });

    vi.useRealTimers();
  });
});

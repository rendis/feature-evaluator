import { featureQueries } from './feature-queries';

import { featuresApi } from '@/api/features';

vi.mock('@/api/features', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/features')>();
  return {
    ...actual,
    featuresApi: {
      ...actual.featuresApi,
      list: vi.fn(),
    },
  };
});

describe('featureQueries.summaryList', () => {
  it('requests the summary view payload', async () => {
    vi.mocked(featuresApi.list).mockResolvedValue({
      data: [],
      pagination: { page: 1, pageSize: 20, total: 0, totalPages: 0 },
    });

    const query = featureQueries.summaryList({ page: 2, pageSize: 10, search: 'beta' });
    const queryFn = query.queryFn;
    if (!queryFn) {
      throw new Error('summaryList queryFn is required');
    }

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    await queryFn({ queryKey: query.queryKey, meta: undefined, signal: new AbortController().signal } as any);

    expect(query.queryKey).toEqual([
      'features',
      'list',
      { page: 2, pageSize: 10, search: 'beta', view: 'summary' },
    ]);
    expect(featuresApi.list).toHaveBeenCalledWith({
      page: 2,
      pageSize: 10,
      search: 'beta',
      view: 'summary',
    });
  });
});

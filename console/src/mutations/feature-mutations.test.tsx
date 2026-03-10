import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react';

import { useToggleFeature } from './feature-mutations';

import type {
  Feature,
  FeatureSummary,
  PaginatedResponse,
  ToggleFeatureResponse,
} from '@/api/types';
import type { ReactNode } from 'react';

import { featuresApi } from '@/api/features';
import { featureQueries } from '@/queries/feature-queries';

vi.mock('@/api/features', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/features')>();
  return {
    ...actual,
    featuresApi: {
      ...actual.featuresApi,
      create: vi.fn(),
      update: vi.fn(),
      remove: vi.fn(),
      toggle: vi.fn(),
    },
  };
});

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe('useToggleFeature', () => {
  it('patches detail and summary list caches optimistically without global invalidation', async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');

    const summaryParams = { page: 1, pageSize: 20 };
    const summaryFeature: FeatureSummary = {
      id: 'feat-1',
      key: 'checkout_v2',
      name: 'Checkout V2',
      description: 'new checkout',
      enabled: true,
      valueType: 'boolean',
      tags: [],
      packCount: 1,
      ruleCount: 2,
      createdAt: '2026-03-08T12:00:00Z',
      updatedAt: '2026-03-08T12:00:00Z',
      createdBy: 'tester',
      updatedBy: 'tester',
    };
    const detailFeature: Feature = {
      ...summaryFeature,
      defaultValue: false,
      metadata: {},
      inputContract: { headers: [] },
      packs: [{ key: 'starter', name: 'Starter' }],
      rules: [],
    };

    queryClient.setQueryData(featureQueries.summaryList(summaryParams).queryKey, {
      data: [summaryFeature],
      pagination: { page: 1, pageSize: 20, total: 1, totalPages: 1 },
    });
    queryClient.setQueryData(featureQueries.detail(detailFeature.key).queryKey, detailFeature);

    let resolveToggle: ((value: ToggleFeatureResponse) => void) | undefined;
    vi.mocked(featuresApi.toggle).mockImplementation(
      () =>
        new Promise<ToggleFeatureResponse>((resolve) => {
          resolveToggle = resolve;
        }),
    );

    const { result } = renderHook(() => useToggleFeature(), {
      wrapper: createWrapper(queryClient),
    });

    let mutationPromise: Promise<unknown> | undefined;
    act(() => {
      mutationPromise = result.current.mutateAsync({ key: detailFeature.key, enabled: false });
    });

    await waitFor(() => {
      expect(
        queryClient.getQueryData<Feature>(featureQueries.detail(detailFeature.key).queryKey)
          ?.enabled,
      ).toBe(false);
    });
    expect(
      queryClient.getQueryData<PaginatedResponse<FeatureSummary>>(
        featureQueries.summaryList(summaryParams).queryKey,
      )?.data[0]?.enabled,
    ).toBe(false);

    act(() => {
      resolveToggle?.({ message: 'feature toggled', enabled: false });
    });
    if (!mutationPromise) {
      throw new Error('mutation promise was not created');
    }
    await mutationPromise;

    expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: featureQueries.lists() });
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: featureQueries.detail(detailFeature.key).queryKey,
      exact: true,
    });
    expect(invalidateSpy).not.toHaveBeenCalledWith({ queryKey: featureQueries.all() });
  });
});

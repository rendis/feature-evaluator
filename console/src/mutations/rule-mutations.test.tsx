import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react';

import { useToggleRule } from './rule-mutations';

import type { Feature, Rule } from '@/api/types';
import type { ReactNode } from 'react';

import { rulesApi } from '@/api/rules';
import { featureQueries } from '@/queries/feature-queries';
import { ruleQueries } from '@/queries/rule-queries';

vi.mock('@/api/rules', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/rules')>();
  return {
    ...actual,
    rulesApi: {
      ...actual.rulesApi,
      create: vi.fn(),
      update: vi.fn(),
      remove: vi.fn(),
      reorder: vi.fn(),
    },
  };
});

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
}

function createRule(overrides: Partial<Rule> = {}): Rule {
  return {
    id: 'rule-1',
    name: 'Rule A',
    priority: 1,
    enabled: true,
    expression: 'user.id == "u-1"',
    value: true,
    sourceBindings: { segments: [{ segmentKey: 'vip', lookupPath: 'user.id' }] },
    externalApiBindings: [
      {
        externalApiKey: 'identity-check',
        paramMappings: [{ paramName: 'userId', mode: 'input', inputPath: 'user.id' }],
        failMode: 'closed',
        cacheTTL: 60,
      },
    ],
    metadata: { source: 'test' },
    createdAt: '2026-03-26T00:00:00Z',
    updatedAt: '2026-03-26T00:00:00Z',
    ...overrides,
  };
}

function createFeature(featureKey: string, rules: Rule[]): Feature {
  return {
    id: 'feature-1',
    key: featureKey,
    name: 'Checkout V2',
    description: 'new checkout',
    enabled: true,
    valueType: 'boolean',
    defaultValue: false,
    metadata: {},
    tags: [],
    inputContract: { headers: [] },
    rules,
    createdAt: '2026-03-26T00:00:00Z',
    updatedAt: '2026-03-26T00:00:00Z',
    createdBy: 'tester',
    updatedBy: 'tester',
  };
}

afterEach(() => {
  vi.clearAllMocks();
});

it('updates the rule optimistically and sends the full update payload', async () => {
  const featureKey = 'checkout_v2';
  const queryClient = createQueryClient();
  const invalidateSpy = vi.spyOn(queryClient, 'invalidateQueries');
  const rule = createRule();
  const detail = createFeature(featureKey, [rule]);

  queryClient.setQueryData(ruleQueries.list(featureKey).queryKey, [rule]);
  queryClient.setQueryData(featureQueries.detail(featureKey).queryKey, detail);

  let resolveUpdate: ((value: Rule) => void) | undefined;
  vi.mocked(rulesApi.update).mockImplementation(
    () =>
      new Promise<Rule>((resolve) => {
        resolveUpdate = resolve;
      }),
  );

  const { result } = renderHook(() => useToggleRule(featureKey), {
    wrapper: createWrapper(queryClient),
  });

  let mutationPromise: Promise<unknown> | undefined;
  act(() => {
    mutationPromise = result.current.mutateAsync({ rule });
  });

  await waitFor(() => {
    expect(queryClient.getQueryData<Rule[]>(ruleQueries.list(featureKey).queryKey)?.[0]?.enabled).toBe(
      false,
    );
  });
  expect(
    queryClient.getQueryData<Feature>(featureQueries.detail(featureKey).queryKey)?.rules?.[0]
      ?.enabled,
  ).toBe(false);
  expect(rulesApi.update).toHaveBeenCalledWith(featureKey, rule.id, {
    name: rule.name,
    priority: rule.priority,
    enabled: false,
    expression: rule.expression,
    value: rule.value,
    sourceBindings: rule.sourceBindings,
    externalApiBindings: rule.externalApiBindings,
    metadata: rule.metadata,
    rolloutPercentage: null,
  });

  act(() => {
    resolveUpdate?.({ ...rule, enabled: false });
  });
  if (!mutationPromise) {
    throw new Error('mutation promise was not created');
  }
  await mutationPromise;

  expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ruleQueries.all(featureKey) });
  expect(invalidateSpy).toHaveBeenCalledWith({
    queryKey: featureQueries.detail(featureKey).queryKey,
    exact: true,
  });
});

it('rolls back the optimistic cache when the toggle request fails', async () => {
  const featureKey = 'checkout_v2';
  const queryClient = createQueryClient();
  const rule = createRule();
  const detail = createFeature(featureKey, [rule]);

  queryClient.setQueryData(ruleQueries.list(featureKey).queryKey, [rule]);
  queryClient.setQueryData(featureQueries.detail(featureKey).queryKey, detail);

  let rejectUpdate: ((reason?: unknown) => void) | undefined;
  vi.mocked(rulesApi.update).mockImplementation(
    () =>
      new Promise<Rule>((_resolve, reject) => {
        rejectUpdate = reject;
      }),
  );

  const { result } = renderHook(() => useToggleRule(featureKey), {
    wrapper: createWrapper(queryClient),
  });

  let mutationPromise: Promise<unknown> | undefined;
  act(() => {
    mutationPromise = result.current.mutateAsync({ rule });
  });

  await waitFor(() => {
    expect(queryClient.getQueryData<Rule[]>(ruleQueries.list(featureKey).queryKey)?.[0]?.enabled).toBe(
      false,
    );
  });

  act(() => {
    rejectUpdate?.(new Error('boom'));
  });

  await expect(mutationPromise).rejects.toThrow('boom');

  expect(queryClient.getQueryData<Rule[]>(ruleQueries.list(featureKey).queryKey)?.[0]?.enabled).toBe(
    true,
  );
  expect(
    queryClient.getQueryData<Feature>(featureQueries.detail(featureKey).queryKey)?.rules?.[0]
      ?.enabled,
  ).toBe(true);
  });

import * as rq from '@tanstack/react-query';

import { NewRulePage } from './new';

import { render, screen } from '@/test/test-utils';

const routeState = {
  params: { featureKey: 'feature-a' },
  search: {} as { cloneFrom?: string },
};

const capturedBuilder = {
  props: null as Record<string, unknown> | null,
};

vi.mock('@tanstack/react-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-query')>();
  return {
    ...actual,
    useSuspenseQuery: vi.fn(),
  };
});

vi.mock('@tanstack/react-router', () => ({
  createFileRoute: () => () => ({
    useParams: () => routeState.params,
    useSearch: () => routeState.search,
  }),
}));

vi.mock('@/components/rules/rule-builder', () => ({
  RuleBuilder: (props: Record<string, unknown>) => {
    capturedBuilder.props = props;
    return <div>rule-builder</div>;
  },
}));

vi.mock('@/queries/feature-queries', () => ({
  featureQueries: {
    detail: vi.fn((featureKey: string) => ({ queryKey: ['feature', featureKey] })),
  },
}));

vi.mock('@/queries/rule-queries', () => ({
  ruleQueries: {
    list: vi.fn((featureKey: string) => ({ queryKey: ['rules', featureKey] })),
  },
}));

const mockedUseSuspenseQuery = vi.mocked(rq.useSuspenseQuery);

describe('NewRulePage', () => {
  const feature = {
    id: 'feature-id',
    key: 'feature-a',
    name: 'Feature A',
    description: '',
    enabled: true,
    valueType: 'json' as const,
    defaultValue: {},
    metadata: {},
    tags: [],
    inputContract: { headers: [] },
    createdAt: '',
    updatedAt: '',
    createdBy: '',
    updatedBy: '',
  };

  const sourceRule = {
    id: 'rule-1',
    name: 'VIP rule',
    priority: 1,
    enabled: false,
    expression: 'tenant.id == "acme"',
    value: { mode: 'vip' },
    rolloutPercentage: 25,
    sourceBindings: { segments: [{ segmentKey: 'vip', lookupPath: 'user.id' }] },
    externalApiBindings: [
      {
        externalApiKey: 'kyc',
        paramMappings: [{ paramName: 'userId', mode: 'input', inputPath: 'user.id' }],
        failMode: 'open' as const,
        cacheTTL: 60,
      },
    ],
    metadata: { source: 'mcp', nested: { group: 'vip' } },
    createdAt: '',
    updatedAt: '',
  };

  beforeEach(() => {
    capturedBuilder.props = null;
    routeState.search = {};

    mockedUseSuspenseQuery.mockImplementation((options) => {
      if (Array.isArray(options.queryKey) && options.queryKey[0] === 'feature') {
        return { data: feature } as ReturnType<typeof rq.useSuspenseQuery>;
      }

      return { data: [sourceRule, { ...sourceRule, id: 'rule-2', priority: 2, name: 'Fallback' }] } as ReturnType<
        typeof rq.useSuspenseQuery
      >;
    });
  });

  afterEach(() => {
    mockedUseSuspenseQuery.mockReset();
  });

  it('builds a cloned draft for the create flow when cloneFrom is valid', () => {
    routeState.search = { cloneFrom: 'rule-1' };

    render(<NewRulePage />, {
      providerProps: { namespaces: ['rules', 'common'] },
    });

    expect(screen.getByText('rule-builder')).toBeInTheDocument();
    expect(capturedBuilder.props).toMatchObject({
      feature,
      nextPriority: 3,
    });

    const initialDraft = capturedBuilder.props?.initialDraft as Record<string, unknown>;
    expect(initialDraft).toMatchObject({
      name: 'Copia de VIP rule',
      priority: 3,
      enabled: false,
      expression: 'tenant.id == "acme"',
      value: JSON.stringify({ mode: 'vip' }, null, 2),
      rolloutEnabled: true,
      rolloutLimit: 25,
    });
    expect(initialDraft.metadata).toEqual(sourceRule.metadata);
    expect(initialDraft.metadata).not.toBe(sourceRule.metadata);
    expect(initialDraft.sourceBindings).toEqual(sourceRule.sourceBindings);
    expect(initialDraft.sourceBindings).not.toBe(sourceRule.sourceBindings);
    expect(initialDraft.externalApiBindings).toEqual(sourceRule.externalApiBindings);
    expect(initialDraft.externalApiBindings).not.toBe(sourceRule.externalApiBindings);
    expect(capturedBuilder.props?.rule).toBeUndefined();
  });

  it('shows an error state when cloneFrom does not match any rule', () => {
    routeState.search = { cloneFrom: 'missing-rule' };

    render(<NewRulePage />, {
      providerProps: { namespaces: ['rules', 'common'] },
    });

    expect(screen.getByText('No encontramos la regla que quieres clonar.')).toBeInTheDocument();
    expect(screen.queryByText('rule-builder')).not.toBeInTheDocument();
  });
});

import { fireEvent, render, screen } from '@/test/test-utils';

import { FeatureObservabilityPage } from './observability';

import type { ReactNode } from 'react';

const queryState = vi.hoisted(() => ({
  canAuditRead: true,
}));

vi.mock('@tanstack/react-router', () => ({
  Link: ({ children }: { children: ReactNode }) => <a href="#">{children}</a>,
  useNavigate: () => vi.fn(),
  createFileRoute: () => () => ({
    useParams: () => ({ featureKey: 'feature-a' }),
  }),
}));

vi.mock('@/hooks/use-permissions', () => ({
  usePermissions: () => ({
    can: (permission: string) => permission === 'audit.read' && queryState.canAuditRead,
    role: 'owner',
  }),
}));

vi.mock('@/components/features/feature-detail-header', () => ({
  FeatureDetailHeader: ({ feature }: { feature: { name: string } }) => (
    <div>Header: {feature.name}</div>
  ),
}));

vi.mock('@tanstack/react-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-query')>();
  return {
    ...actual,
    useQuery: vi.fn((options?: { queryKey?: unknown[] }) => {
      const key = options?.queryKey ?? [];
      if (key.includes('overview')) {
        return {
          data: {
            featureKey: 'feature-a',
            summary: [
              { key: 'totalEvaluations', label: 'Total evals', value: 12, unit: 'count' },
              { key: 'usedRedisRatio', label: 'Redis usage', value: 75, unit: 'ratio' },
              { key: 'averageDurationMs', label: 'Avg latency', value: 35, unit: 'ms' },
              { key: 'p95DurationMs', label: 'p95 latency', value: 80, unit: 'ms' },
            ],
            components: [
              {
                component: 'feature_snapshot',
                cacheEnabled: true,
                cacheBackend: 'redis',
                cacheStatus: 'hit',
                ttlSeconds: 120,
                totalDurationMs: 100,
                averageDurationMs: 20,
                p95DurationMs: 30,
                hitCount: 8,
                missCount: 4,
                computedCount: 0,
                disabledCount: 0,
                outcome: 'hit',
              },
            ],
            totalEvaluations: 12,
            usedRedisCount: 9,
            usedRedisRatio: 0.75,
            averageDurationMs: 35,
            p95DurationMs: 80,
            ruleCount: 1,
            matchedRuleCount: 1,
            slowEvaluations: 0,
          },
          isLoading: false,
          error: null,
        };
      }

      if (key.includes('rule') && !key.includes('rules')) {
        return {
          data: {
            ruleId: 'rule-a',
            ruleName: 'Rule A',
            priority: 1,
            matchedCount: 6,
            totalCount: 8,
            averageDurationMs: 20,
            p95DurationMs: 40,
            cacheEnabled: true,
            cacheStatus: 'hit',
            cacheBackend: 'redis',
            expressionDurationMs: 12,
            expressionCompileCacheHitCount: 5,
            expressionCompileCacheMissCount: 1,
            externalCalls: [
              {
                apiKey: 'validator',
                cacheEnabled: true,
                cacheStatus: 'hit',
                durationMs: 18,
                totalCalls: 3,
                hitCount: 2,
                missCount: 1,
                computedCount: 0,
                ttlSeconds: 120,
                passedCount: 3,
                failedCount: 0,
                httpStatus: 200,
              },
            ],
            traces: [
              {
                id: 'trace-1',
                requestId: 'req-1',
                createdAt: '2026-03-26T12:00:00.000Z',
                totalDurationMs: 42,
                usedRedis: true,
                cacheStatus: 'hit',
                matched: true,
                ruleId: 'rule-a',
                ruleName: 'Rule A',
                reason: 'matched rule',
                steps: [],
              },
            ],
            generatedAt: '2026-03-26T12:00:00.000Z',
          },
          isLoading: false,
          error: null,
        };
      }

      if (key.includes('rules')) {
        return {
          data: {
            data: [
              {
                ruleId: 'rule-a',
                ruleName: 'Rule A',
                priority: 1,
                matchedCount: 6,
                totalCount: 8,
                averageDurationMs: 20,
                p95DurationMs: 40,
                cacheEnabled: true,
                cacheStatus: 'hit',
                cacheBackend: 'redis',
                expressionDurationMs: 12,
                expressionCompileCacheHitCount: 5,
                expressionCompileCacheMissCount: 1,
                externalCalls: [],
              },
            ],
          },
          isLoading: false,
          error: null,
        };
      }

      if (key.includes('traces')) {
        return {
          data: {
            data: [
              {
                id: 'trace-1',
                requestId: 'req-1',
                createdAt: '2026-03-26T12:00:00.000Z',
                totalDurationMs: 42,
                usedRedis: true,
                cacheStatus: 'hit',
                matched: true,
                ruleId: 'rule-a',
                ruleName: 'Rule A',
                reason: 'matched rule',
                steps: [],
              },
            ],
            pagination: { page: 1, pageSize: 10, total: 1, totalPages: 1 },
          },
          isLoading: false,
          error: null,
        };
      }

      return { data: undefined, isLoading: false, error: null };
    }),
    useSuspenseQuery: vi.fn((options?: { queryKey?: unknown[] }) => {
      const key = options?.queryKey ?? [];
      if (key.includes('detail') && !key.includes('observability')) {
        return {
          data: {
            id: 'feature-1',
            key: 'feature-a',
            name: 'Feature A',
            description: 'Demo feature',
            enabled: true,
            valueType: 'boolean',
            defaultValue: true,
            metadata: {},
            tags: [],
            inputContract: { headers: [] },
            createdAt: '',
            updatedAt: '',
            createdBy: '',
            updatedBy: '',
          },
        };
      }

      return { data: undefined };
    }),
  };
});

describe('FeatureObservabilityPage', () => {
  afterEach(() => {
    queryState.canAuditRead = true;
  });

  it('renders summary, rules and trace tables and opens the rule drawer', async () => {
    render(<FeatureObservabilityPage featureKey="feature-a" />, {
      providerProps: { namespaces: ['features', 'common', 'auth'] },
    });

    expect(screen.getByText('Observability')).toBeInTheDocument();
    expect(screen.getByText('Total evals')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Rule A' })).toBeInTheDocument();
    expect(screen.getByText('req-1')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Rule A' }));

    expect(await screen.findByText('External calls')).toBeInTheDocument();
    expect(screen.getByText('External calls')).toBeInTheDocument();
    expect(screen.getByText('Recent traces')).toBeInTheDocument();
  });

  it('blocks users without audit.read', () => {
    queryState.canAuditRead = false;

    render(<FeatureObservabilityPage featureKey="feature-a" />, {
      providerProps: { namespaces: ['features', 'common', 'auth'] },
    });

    expect(screen.getByText('Access denied')).toBeInTheDocument();
    expect(screen.getByText('You do not have permission to view feature observability.')).toBeInTheDocument();
  });
});

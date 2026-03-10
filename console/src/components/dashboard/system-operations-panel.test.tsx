import * as rq from '@tanstack/react-query';
import userEvent from '@testing-library/user-event';

import { SystemOperationsPanel } from './system-operations-panel';

import { render, screen } from '@/test/test-utils';

vi.mock('@tanstack/react-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-query')>();
  return {
    ...actual,
    useQuery: vi.fn(),
    useQueryClient: vi.fn(),
  };
});

const mockedUseQuery = vi.mocked(rq.useQuery);
const mockedUseQueryClient = vi.mocked(rq.useQueryClient);

const healthyOperations = {
  checkedAt: '2026-03-05T12:30:00Z',
  overallStatus: 'healthy' as const,
  services: {
    postgresql: {
      status: 'healthy' as const,
      latencyMs: 8,
    },
    redis: {
      status: 'healthy' as const,
      latencyMs: 4,
      circuitOpen: false,
      openUntil: null,
    },
  },
  metrics: {
    evaluationsToday: 120,
    errorsToday: 3,
    cacheHitRatio: 0.75,
    rateLimitRejectsToday: 2,
    externalP50Ms: 18,
    externalP95Ms: 42,
    circuitBreakerOpenEvents: 1,
    circuitBreakerCloseEvents: 1,
  },
};

describe('SystemOperationsPanel', () => {
  beforeEach(() => {
    mockedUseQueryClient.mockReturnValue({
      invalidateQueries: vi.fn().mockResolvedValue(undefined),
    } as unknown as ReturnType<typeof rq.useQueryClient>);

    mockedUseQuery.mockReturnValue({
      data: healthyOperations,
      error: null,
      isLoading: false,
      isFetching: false,
    } as ReturnType<typeof rq.useQuery>);
  });

  it('renders healthy services and runtime metrics', () => {
    render(<SystemOperationsPanel />);

    expect(screen.getByTestId('system-operations-panel')).toBeInTheDocument();
    expect(screen.getByText('operations.title')).toBeInTheDocument();
    expect(screen.getByText('operations.globalLabel')).toBeInTheDocument();
    expect(screen.getByText('operations.services.postgresql')).toBeInTheDocument();
    expect(screen.getByText('operations.services.redis')).toBeInTheDocument();
    expect(screen.getByText('operations.metrics.evaluationsToday')).toBeInTheDocument();
    expect(screen.getByText('120')).toBeInTheDocument();
  });

  it('renders degraded redis state details', () => {
    mockedUseQuery.mockReturnValue({
      data: {
        ...healthyOperations,
        overallStatus: 'degraded',
        services: {
          ...healthyOperations.services,
          redis: {
            status: 'degraded',
            latencyMs: null,
            circuitOpen: true,
            openUntil: '2026-03-05T12:45:00Z',
          },
        },
      },
      error: null,
      isLoading: false,
      isFetching: false,
    } as ReturnType<typeof rq.useQuery>);

    render(<SystemOperationsPanel />);

    expect(screen.getAllByText('operations.status.degraded').length).toBeGreaterThan(0);
    expect(screen.getByText('operations.circuitOpenUntil')).toBeInTheDocument();
  });

  it('refreshes only the operations query', async () => {
    const user = userEvent.setup();
    const invalidateQueries = vi.fn().mockResolvedValue(undefined);

    mockedUseQueryClient.mockReturnValue({
      invalidateQueries,
    } as unknown as ReturnType<typeof rq.useQueryClient>);

    render(<SystemOperationsPanel />);

    await user.click(screen.getByRole('button', { name: 'operations.refresh' }));

    expect(invalidateQueries).toHaveBeenCalledWith({
      queryKey: ['dashboard', 'operations'],
      exact: true,
    });
  });
});

import * as rq from '@tanstack/react-query';

import { WorkspaceSelector } from './workspace-selector';

import { useWorkspaceStore } from '@/stores/workspace-store';
import { render, screen, waitFor } from '@/test/test-utils';


vi.mock('@tanstack/react-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-query')>();
  return {
    ...actual,
    useQuery: vi.fn().mockReturnValue({
      data: [
        { key: 'default', name: 'Default' },
        { key: 'staging', name: 'Staging' },
        { key: 'prod', name: 'Production' },
      ],
      isLoading: false,
    }),
  };
});

const mockedUseQuery = vi.mocked(rq.useQuery);

describe('WorkspaceSelector', () => {
  beforeEach(() => {
    useWorkspaceStore.setState({ workspaceKey: 'default' });
    mockedUseQuery.mockReturnValue({
      data: [
        { key: 'default', name: 'Default' },
        { key: 'staging', name: 'Staging' },
        { key: 'prod', name: 'Production' },
      ],
      isLoading: false,
    } as ReturnType<typeof rq.useQuery>);
  });

  it('renders select trigger when not collapsed', () => {
    render(<WorkspaceSelector collapsed={false} />);
    expect(screen.getByRole('combobox')).toBeInTheDocument();
  });

  it('renders collapsed button when collapsed', () => {
    render(<WorkspaceSelector collapsed={true} />);
    expect(screen.getByRole('button')).toBeInTheDocument();
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument();
  });

  it('renders select trigger when only one workspace exists', () => {
    mockedUseQuery.mockReturnValue({
      data: [{ key: 'default', name: 'Default' }],
      isLoading: false,
    } as ReturnType<typeof rq.useQuery>);

    render(<WorkspaceSelector collapsed={false} />);
    expect(screen.getByRole('combobox')).toBeInTheDocument();
  });

  it('returns null when no workspaces exist', () => {
    mockedUseQuery.mockReturnValue({
      data: [],
      isLoading: false,
    } as ReturnType<typeof rq.useQuery>);

    const { container } = render(<WorkspaceSelector collapsed={false} />);
    expect(container.firstChild).toBeNull();
  });

  it('reconciles an invalid workspace key to the first active workspace', async () => {
    useWorkspaceStore.setState({ workspaceKey: 'missing' });

    render(<WorkspaceSelector collapsed={false} />);

    await waitFor(() => {
      expect(useWorkspaceStore.getState().workspaceKey).toBe('default');
    });
  });
});

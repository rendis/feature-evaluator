import userEvent from '@testing-library/user-event';
import type { ReactNode } from 'react';

import { SecurityPolicyPage, SecurityPolicyRoutePage } from './security-policy-page';

import { render, screen } from '@/test/test-utils';

function currentOrigin() {
  return window.location.origin;
}

const state = vi.hoisted(() => ({
  canManageSecurity: true,
  policy: {
    corsOrigins: {
      managed: ['http://localhost'],
      inherited: [],
      effective: ['http://localhost'],
    },
    externalApiAllowHosts: {
      managed: ['api.example.com'],
      inherited: ['core.example.com'],
      effective: ['core.example.com', 'api.example.com'],
    },
    updatedAt: '2026-03-23T12:00:00Z',
    updatedBy: 'owner@example.com',
  },
  mutate: vi.fn(),
  toastSuccess: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock('@tanstack/react-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-query')>();
  return {
    ...actual,
    useSuspenseQuery: vi.fn(() => ({ data: state.policy })),
  };
});

vi.mock('@tanstack/react-router', () => ({
  Navigate: ({ to }: { to: string }) => <div>redirect:{to}</div>,
}));

vi.mock('@/hooks/use-permissions', () => ({
  usePermissions: () => ({
    can: (permission: string) => permission === 'security.manage' && state.canManageSecurity,
  }),
}));

vi.mock('@/mutations/security-policy-mutations', () => ({
  useUpdateSecurityPolicy: () => ({
    mutate: state.mutate,
    isPending: false,
  }),
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

vi.mock('@/components/shared/confirm-dialog', () => ({
  ConfirmDialog: ({
    open,
    title,
    description,
    onConfirm,
  }: {
    open: boolean;
    title?: string;
    description?: string;
    onConfirm: () => void;
  }) =>
    open ? (
      <div>
        <p>{title}</p>
        <p>{description}</p>
        <button type="button" onClick={onConfirm}>
          confirm-warning
        </button>
      </div>
    ) : null,
}));

vi.mock('sonner', () => ({
  toast: {
    success: (message: string) => state.toastSuccess(message),
    error: (message: string) => state.toastError(message),
  },
}));

describe('SecurityPolicyRoutePage', () => {
  beforeEach(() => {
    state.canManageSecurity = true;
    state.policy = {
      corsOrigins: {
        managed: [currentOrigin()],
        inherited: [],
        effective: [currentOrigin()],
      },
      externalApiAllowHosts: {
        managed: ['api.example.com'],
        inherited: ['core.example.com'],
        effective: ['core.example.com', 'api.example.com'],
      },
      updatedAt: '2026-03-23T12:00:00Z',
      updatedBy: 'owner@example.com',
    };
    state.mutate.mockReset();
    state.toastSuccess.mockReset();
    state.toastError.mockReset();
  });

  it('redirects non-owners to access denied', () => {
    state.canManageSecurity = false;

    render(<SecurityPolicyRoutePage />, {
      providerProps: { namespaces: ['security-policies', 'common'] },
    });

    expect(screen.getByText('redirect:/auth/access-denied')).toBeInTheDocument();
  });

  it('renders inherited values disabled and excludes them from the mutation payload', async () => {
    const user = userEvent.setup();

    render(<SecurityPolicyPage />, {
      providerProps: { namespaces: ['security-policies', 'common'] },
    });

    expect(screen.getByRole('button', { name: 'core.example.com' })).toBeDisabled();

    await user.type(screen.getByPlaceholderText('hosts.placeholder'), 'billing.example.com');
    await user.click(screen.getAllByRole('button', { name: 'hosts.add' })[0]);
    await user.click(screen.getByRole('button', { name: 'actions.save' }));

    expect(state.mutate).toHaveBeenCalledWith(
      {
        corsOrigins: [currentOrigin()],
        externalApiAllowHosts: ['api.example.com', 'billing.example.com'],
      },
      expect.objectContaining({
        onSuccess: expect.any(Function),
        onError: expect.any(Function),
      }),
    );
  });

  it('shows a destructive warning before removing the current browser origin', async () => {
    const user = userEvent.setup();

    render(<SecurityPolicyPage />, {
      providerProps: { namespaces: ['security-policies', 'common'] },
    });

    await user.click(
      screen.getAllByRole('button', {
        name: 'managed.removeValue',
      })[0],
    );
    await user.click(screen.getByRole('button', { name: 'actions.save' }));

    expect(screen.getByText('warning.title')).toBeInTheDocument();
    expect(state.mutate).not.toHaveBeenCalled();
  });

  it('surfaces mutation errors through toast', async () => {
    const user = userEvent.setup();

    render(<SecurityPolicyPage />, {
      providerProps: { namespaces: ['security-policies', 'common'] },
    });

    await user.type(screen.getByPlaceholderText('hosts.placeholder'), 'billing.example.com');
    await user.click(screen.getAllByRole('button', { name: 'hosts.add' })[0]);
    await user.click(screen.getByRole('button', { name: 'actions.save' }));

    const [, options] = state.mutate.mock.calls[0];
    options.onError(new Error('boom'));

    expect(state.toastError).toHaveBeenCalledWith('messages.saveError');
  });

  it('handles null lists from the API without crashing', () => {
    state.policy = {
      corsOrigins: {
        managed: null as unknown as string[],
        inherited: null as unknown as string[],
        effective: null as unknown as string[],
      },
      externalApiAllowHosts: {
        managed: null as unknown as string[],
        inherited: null as unknown as string[],
        effective: null as unknown as string[],
      },
      updatedAt: undefined,
      updatedBy: '',
    };

    render(<SecurityPolicyPage />, {
      providerProps: { namespaces: ['security-policies', 'common'] },
    });

    expect(screen.getAllByText('managed.empty')[0]).toBeInTheDocument();
  });
});

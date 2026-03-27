import userEvent from '@testing-library/user-event';

import { RuleToggle } from './rule-toggle';

import type { Rule } from '@/api/types';
import type { ReactNode } from 'react';

import { render, screen } from '@/test/test-utils';

const { toggleMutate, successToast, errorToast } = vi.hoisted(() => ({
  toggleMutate: vi.fn(),
  successToast: vi.fn(),
  errorToast: vi.fn(),
}));

let canWrite = true;
let isPending = false;

vi.mock('sonner', () => ({
  toast: {
    success: successToast,
    error: errorToast,
  },
}));

vi.mock('@/hooks/use-permissions', () => ({
  usePermissions: () => ({
    can: () => canWrite,
    role: canWrite ? 'editor' : 'viewer',
  }),
}));

vi.mock('@/components/ui/tooltip', () => ({
  Tooltip: ({ children }: { children: ReactNode }) => <>{children}</>,
  TooltipTrigger: ({ children }: { children: ReactNode }) => <>{children}</>,
  TooltipContent: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock('@/mutations/rule-mutations', () => ({
  useToggleRule: () => ({
    mutate: toggleMutate,
    isPending,
  }),
}));

function createRule(overrides: Partial<Rule> = {}): Rule {
  return {
    id: 'rule-1',
    name: 'Rule A',
    priority: 1,
    enabled: true,
    expression: 'true',
    value: true,
    sourceBindings: { segments: [] },
    externalApiBindings: [],
    metadata: {},
    createdAt: '',
    updatedAt: '',
    ...overrides,
  };
}

describe('RuleToggle', () => {
  afterEach(() => {
    canWrite = true;
    isPending = false;
    toggleMutate.mockReset();
    successToast.mockReset();
    errorToast.mockReset();
  });

  it('shows a success toast for the new enabled state', async () => {
    const user = userEvent.setup();

    render(<RuleToggle featureKey="feature-a" rule={createRule({ enabled: false })} />, {
      providerProps: { namespaces: ['rules'] },
    });

    await user.click(screen.getByRole('switch', { name: 'Activar o desactivar Rule A' }));

    const options = toggleMutate.mock.calls[0]?.[1];
    options?.onSuccess?.();

    expect(successToast).toHaveBeenCalledWith('toggle.enableSuccess');
  });

  it('disables the switch when the user lacks write permission', () => {
    canWrite = false;

    render(<RuleToggle featureKey="feature-a" rule={createRule()} />, {
      providerProps: { namespaces: ['rules'] },
    });

    expect(screen.getByRole('switch', { name: 'Activar o desactivar Rule A' })).toBeDisabled();
  });
});

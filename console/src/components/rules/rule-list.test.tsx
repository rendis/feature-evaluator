import userEvent from '@testing-library/user-event';

import { RuleList } from './rule-list';

import type { ComponentProps } from 'react';

import { render, screen } from '@/test/test-utils';

const navigateMock = vi.fn();
const deleteMutate = vi.fn();
const reorderMutate = vi.fn();
const toggleMutate = vi.fn();
let toggleIsPending = false;

vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => navigateMock,
}));

vi.mock('@/components/shared/confirm-dialog', () => ({
  ConfirmDialog: () => null,
}));

vi.mock('@/components/shared/permission-button', () => ({
  PermissionButton: ({
    children,
    permission: _permission,
    tooltipMessage: _tooltipMessage,
    variant: _variant,
    size: _size,
    asChild: _asChild,
    ...props
  }: ComponentProps<'button'> & {
    permission: string;
    tooltipMessage?: string;
    variant?: string;
    size?: string;
    asChild?: boolean;
  }) => (
    <button type="button" {...props}>
      {children}
    </button>
  ),
}));

vi.mock('@/hooks/use-permissions', () => ({
  usePermissions: () => ({
    can: () => true,
    role: 'editor',
  }),
}));

vi.mock('@/mutations/rule-mutations', () => ({
  useDeleteRule: () => ({
    mutate: deleteMutate,
    isPending: false,
  }),
  useReorderRules: () => ({
    mutate: reorderMutate,
  }),
  useToggleRule: () => ({
    mutate: toggleMutate,
    isPending: toggleIsPending,
  }),
}));

function createRule() {
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
  };
}

function renderRuleList() {
  return render(<RuleList featureKey="feature-a" rules={[createRule()]} />, {
    providerProps: { namespaces: ['rules', 'common'] },
  });
}

afterEach(() => {
  navigateMock.mockReset();
  deleteMutate.mockReset();
  reorderMutate.mockReset();
  toggleMutate.mockReset();
  toggleIsPending = false;
});

it('navigates to the create route with cloneFrom when cloning a rule', async () => {
  const user = userEvent.setup();

  renderRuleList();

  await user.click(screen.getByRole('button', { name: 'Clonar' }));

  expect(navigateMock).toHaveBeenCalledWith({
    to: '/features/$featureKey/rules/new',
    params: { featureKey: 'feature-a' },
    search: { cloneFrom: 'rule-1' },
  });
});

it('renders a toggle for each rule and dispatches the toggle mutation when clicked', async () => {
  const user = userEvent.setup();

  renderRuleList();

  await user.click(screen.getByRole('switch', { name: 'Activar o desactivar Rule A' }));

  expect(toggleMutate).toHaveBeenCalledTimes(1);
  expect(toggleMutate).toHaveBeenCalledWith(
    {
      rule: expect.objectContaining({
        id: 'rule-1',
        enabled: true,
        name: 'Rule A',
      }),
    },
    expect.objectContaining({
      onSuccess: expect.any(Function),
      onError: expect.any(Function),
    }),
  );
});

it('disables the toggle while a rule mutation is pending', () => {
  toggleIsPending = true;

  renderRuleList();

  expect(screen.getByRole('switch', { name: 'Activar o desactivar Rule A' })).toBeDisabled();
});

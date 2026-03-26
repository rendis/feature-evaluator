import userEvent from '@testing-library/user-event';

import { RuleList } from './rule-list';

import type { ComponentProps } from 'react';

import { render, screen } from '@/test/test-utils';

const navigateMock = vi.fn();
const deleteMutate = vi.fn();
const reorderMutate = vi.fn();

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

vi.mock('@/mutations/rule-mutations', () => ({
  useDeleteRule: () => ({
    mutate: deleteMutate,
    isPending: false,
  }),
  useReorderRules: () => ({
    mutate: reorderMutate,
  }),
}));

describe('RuleList', () => {
  afterEach(() => {
    navigateMock.mockReset();
    deleteMutate.mockReset();
    reorderMutate.mockReset();
  });

  it('navigates to the create route with cloneFrom when cloning a rule', async () => {
    const user = userEvent.setup();

    render(
      <RuleList
        featureKey="feature-a"
        rules={[
          {
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
          },
        ]}
      />,
      {
        providerProps: { namespaces: ['rules', 'common'] },
      },
    );

    await user.click(screen.getByRole('button', { name: 'Clonar' }));

    expect(navigateMock).toHaveBeenCalledWith({
      to: '/features/$featureKey/rules/new',
      params: { featureKey: 'feature-a' },
      search: { cloneFrom: 'rule-1' },
    });
  });
});

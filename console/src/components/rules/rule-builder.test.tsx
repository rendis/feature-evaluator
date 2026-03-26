import userEvent from '@testing-library/user-event';

import { RuleBuilder } from './rule-builder';

import { fireEvent, render, screen } from '@/test/test-utils';

const ruleMutationState = vi.hoisted(() => ({
  createMutate: vi.fn(),
  createPending: false,
  deleteMutate: vi.fn(),
  deletePending: false,
  updateMutate: vi.fn(),
  updatePending: false,
}));

vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => vi.fn(),
}));

vi.mock('@/hooks/use-unsaved-changes', () => ({
  useUnsavedChanges: () => ({
    handleBack: vi.fn(),
    UnsavedDialog: () => null,
    markClean: vi.fn(),
  }),
}));

vi.mock('@/components/shared/page-header', () => ({
  PageHeader: ({
    title,
    description,
    actions,
  }: {
    title: string;
    description?: string;
    actions?: React.ReactNode;
  }) => (
    <div>
      <h1>{title}</h1>
      {description ? <p>{description}</p> : null}
      {actions}
    </div>
  ),
}));

vi.mock('./rule-conditions-canvas', async () => {
  const React = await import('react');

  return {
    RuleConditionsCanvas: ({
      onChange,
    }: {
      onChange: (value: {
        expression: string;
        metadata: Record<string, unknown>;
        sourceBindings: { segments: unknown[] };
        externalApiBindings: unknown[];
      }) => void;
    }) => {
      React.useEffect(() => {
        onChange({
          expression: 'true',
          metadata: {},
          sourceBindings: { segments: [] },
          externalApiBindings: [],
        });
      }, [onChange]);

      return <div>conditions-canvas</div>;
    },
  };
});

vi.mock('@/mutations/rule-mutations', () => ({
  useCreateRule: () => ({
    mutate: ruleMutationState.createMutate,
    isPending: ruleMutationState.createPending,
  }),
  useDeleteRule: () => ({
    mutate: ruleMutationState.deleteMutate,
    isPending: ruleMutationState.deletePending,
  }),
  useUpdateRule: () => ({
    mutate: ruleMutationState.updateMutate,
    isPending: ruleMutationState.updatePending,
  }),
}));

describe('RuleBuilder', () => {
  const feature = {
    id: 'feature-id',
    key: 'feature-a',
    name: 'Feature A',
    description: '',
    enabled: true,
    valueType: 'string' as const,
    defaultValue: '',
    metadata: {},
    tags: [] as { key: string; name: string; color: string }[],
    inputContract: { headers: [] },
    createdAt: '',
    updatedAt: '',
    createdBy: '',
    updatedBy: '',
  };

  afterEach(() => {
    ruleMutationState.createMutate.mockReset();
    ruleMutationState.deleteMutate.mockReset();
    ruleMutationState.updateMutate.mockReset();
    ruleMutationState.createPending = false;
    ruleMutationState.deletePending = false;
    ruleMutationState.updatePending = false;
  });

  it('renders the rule step with enabled switch', () => {
    render(<RuleBuilder feature={feature} />);

    expect(screen.getByRole('switch', { name: 'fields.enabled' })).toBeInTheDocument();
  });

  it('renders rollout toggle on the rollout step', async () => {
    const user = userEvent.setup();

    render(<RuleBuilder feature={feature} />);

    // Navigate to the rollout step
    const rolloutTab = screen.getByRole('button', { name: /despliegue/i });
    await user.click(rolloutTab);

    expect(screen.getByRole('switch', { name: 'form.rolloutToggle' })).toBeInTheDocument();
  });

  it('remembers the rollout limit while toggling the section', async () => {
    const user = userEvent.setup();

    render(<RuleBuilder feature={feature} />);

    // Navigate to the rollout step
    const rolloutTab = screen.getByRole('button', { name: /despliegue/i });
    await user.click(rolloutTab);

    const rolloutToggle = screen.getByRole('switch', { name: 'form.rolloutToggle' });
    const rolloutInput = screen.getByRole('spinbutton', { name: 'rollout.limitLabel' });

    expect(rolloutInput).toBeDisabled();
    expect(rolloutInput).toHaveValue(100);

    await user.click(rolloutToggle);
    expect(rolloutInput).toBeEnabled();

    fireEvent.change(rolloutInput, { target: { value: '42' } });
    expect(rolloutInput).toHaveValue(42);

    await user.click(rolloutToggle);
    expect(rolloutInput).toBeDisabled();
    expect(rolloutInput).toHaveValue(42);

    await user.click(rolloutToggle);
    expect(rolloutInput).toBeEnabled();
    expect(rolloutInput).toHaveValue(42);
  });

  it('submits the expected payload for a new rule when optional sections are disabled', async () => {
    const user = userEvent.setup();

    render(<RuleBuilder feature={feature} />);

    await user.type(screen.getByLabelText('fields.name'), 'Rule A');
    await user.type(screen.getByLabelText('fields.value'), 'enabled');

    // Navigate to expression step so the mock canvas fires and sets expression='true'
    const expressionTab = screen.getByRole('button', { name: /expresion/i });
    await user.click(expressionTab);

    const saveButtons = screen.getAllByRole('button', { name: /actions\.save/i });
    await user.click(saveButtons[0]);

    expect(ruleMutationState.createMutate).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'Rule A',
        enabled: true,
        externalApiBindings: [],
        rolloutPercentage: null,
        value: 'enabled',
      }),
      expect.any(Object),
    );
  });

  it('uses create mutation for a cloned draft instead of updating the original rule', async () => {
    const user = userEvent.setup();

    render(
      <RuleBuilder
        feature={feature}
        initialDraft={{
          name: 'Copia de Rule A',
          priority: 3,
          enabled: true,
          value: 'enabled',
          expression: 'true',
          metadata: { source: 'mcp' },
          sourceBindings: { segments: [{ segmentKey: 'vip', lookupPath: 'user.id' }] },
          externalApiBindings: [
            {
              externalApiKey: 'kyc',
              paramMappings: [{ paramName: 'userId', mode: 'input', inputPath: 'user.id' }],
              failMode: 'open',
              cacheTTL: 60,
            },
          ],
          rolloutEnabled: true,
          rolloutLimit: 25,
        }}
      />,
    );

    expect(screen.getByDisplayValue('Copia de Rule A')).toBeInTheDocument();
    expect(screen.getByDisplayValue('3')).toBeInTheDocument();

    const saveButtons = screen.getAllByRole('button', { name: /actions\.save/i });
    await user.click(saveButtons[0]);

    expect(ruleMutationState.createMutate).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'Copia de Rule A',
        priority: 3,
        expression: 'true',
        rolloutPercentage: 25,
      }),
      expect.any(Object),
    );
    expect(ruleMutationState.updateMutate).not.toHaveBeenCalled();
  });

  it('renders even when a legacy feature does not include inputContract', async () => {
    const user = userEvent.setup();

    render(
      <RuleBuilder feature={{ ...feature, inputContract: undefined as never }} />,
    );

    // Navigate to expression step to see the canvas
    const expressionTab = screen.getByRole('button', { name: /expresion/i });
    await user.click(expressionTab);

    expect(screen.getByText('conditions-canvas')).toBeInTheDocument();
  });
});

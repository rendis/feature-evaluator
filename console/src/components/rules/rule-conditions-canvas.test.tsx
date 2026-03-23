import userEvent from '@testing-library/user-event';

import { RuleConditionsCanvas } from './rule-conditions-canvas';
import { emptyGroup, withBuilderMetadata } from './conditions-builder-types';

import { render, screen } from '@/test/test-utils';

vi.mock('@tanstack/react-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-query')>();

  return {
    ...actual,
    useQuery: vi.fn(() => ({ data: undefined })),
    useMutation: vi.fn(() => ({
      mutateAsync: vi.fn(),
      isPending: false,
    })),
  };
});

describe('RuleConditionsCanvas', () => {
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

  it('hides the return-to-builder action for advanced-only expressions', () => {
    render(
      <RuleConditionsCanvas
        feature={feature}
        initialExpression={'user.age == tenant.maxAge'}
        initialMetadata={{}}
        initialSourceBindings={{ segments: [] }}
        onChange={vi.fn()}
      />,
    );

    expect(screen.queryByRole('button', { name: 'Volver al builder' })).not.toBeInTheDocument();
  });

  it('shows the return-to-builder action only when guided metadata exists', async () => {
    const user = userEvent.setup();

    render(
      <RuleConditionsCanvas
        feature={feature}
        initialExpression={'authenticated == true'}
        initialMetadata={withBuilderMetadata({}, emptyGroup())}
        initialSourceBindings={{ segments: [] }}
        onChange={vi.fn()}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'Avanzado' }));
    await user.click(await screen.findByRole('button', { name: 'Guardar en avanzado' }));

    expect(screen.getByRole('button', { name: 'Volver al builder' })).toBeInTheDocument();
  });
});

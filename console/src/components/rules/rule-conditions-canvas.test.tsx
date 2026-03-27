import userEvent from '@testing-library/user-event';

import { RuleConditionsCanvas } from './rule-conditions-canvas';
import { emptyGroup, withBuilderMetadata } from './conditions-builder-types';

import { render, screen } from '@/test/test-utils';

const queryState = vi.hoisted(() => ({
  useQuery: vi.fn((options?: { queryKey?: unknown[] }) => {
    const firstKey = options?.queryKey?.[0];
    if (firstKey === 'external-apis') {
      return {
        data: [
          {
            id: 'api-1',
            key: 'payment-validator',
            name: 'Payment Validator',
            active: true,
            request: { method: 'POST', urlTemplate: 'https://example.com', headers: [] },
            params: [{ name: 'userId', type: 'string', required: true, locations: ['body'] }],
            responseValidation: {
              mode: 'httpCode',
              http: { mode: 'any_2xx' },
              body: { expression: '' },
            },
            hasSecrets: false,
            version: 1,
            createdAt: '',
            updatedAt: '',
            createdBy: '',
            updatedBy: '',
          },
        ],
      };
    }

    return { data: undefined };
  }),
}));

vi.mock('@tanstack/react-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-query')>();

  return {
    ...actual,
    useQuery: queryState.useQuery,
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

  it('rehydrates saved external api params when editing a rule', async () => {
    render(
      <RuleConditionsCanvas
        feature={{
          ...feature,
          inputContract: {
            headers: [
              {
                headerName: 'X-User-Id',
                expressionKey: 'userId',
                label: 'User ID',
                type: 'string',
                required: false,
              },
            ],
          },
        }}
        initialExpression={'externalApi("payment-validator")'}
        initialMetadata={withBuilderMetadata(
          {},
          {
            id: 'root',
            kind: 'group',
            connector: 'and',
            items: [
          {
            id: 'external-api-condition',
            kind: 'condition',
            conditionKind: 'externalApi',
            externalApiKey: 'payment-validator',
            externalApiName: '',
            cacheEnabled: false,
            cacheTTL: 300,
            paramMappings: [],
            negate: false,
          },
        ],
      },
        )}
        initialSourceBindings={{ segments: [] }}
        initialExternalApiBindings={[
          {
            externalApiKey: 'payment-validator',
            cacheEnabled: true,
            cacheTTL: 120,
            paramMappings: [
              {
                paramName: 'userId',
                mode: 'input',
                inputPath: 'headers.userId',
              },
            ],
            failMode: 'open',
          },
        ]}
        onChange={vi.fn()}
      />,
    );

    expect(await screen.findByText('Parametros')).toBeInTheDocument();
    expect(screen.getAllByText(/userId/).length).toBeGreaterThan(0);
    expect(screen.getByRole('button', { name: /Payment Validator/ })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /User ID/ })).toBeInTheDocument();
    expect(screen.getByRole('switch', { name: 'Cache de API externa' })).toBeChecked();
    expect(screen.getByDisplayValue('120')).toBeInTheDocument();
  });
});

import userEvent from '@testing-library/user-event';
import type { ReactNode } from 'react';

import { ExternalApiBuilder } from './external-api-builder';

import { render, screen, waitFor } from '@/test/test-utils';

const externalApiMutationState = vi.hoisted(() => ({
  createMutate: vi.fn(),
  createPending: false,
  updateMutate: vi.fn(),
  updatePending: false,
  deleteMutate: vi.fn(),
  deletePending: false,
  testMutate: vi.fn(),
  testPending: false,
}));
const fetchMock = vi.hoisted(() => vi.fn());
const navigateMock = vi.hoisted(() => vi.fn());
const unsavedChangesState = vi.hoisted(() => ({
  handleBack: vi.fn(),
  hookMock: vi.fn(),
}));

vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => navigateMock,
}));

vi.mock('@/hooks/use-unsaved-changes', () => ({
  useUnsavedChanges: (options: unknown) => {
    unsavedChangesState.hookMock(options);

    return {
      handleBack: unsavedChangesState.handleBack,
      UnsavedDialog: () => null,
      markClean: vi.fn(),
    };
  },
}));

vi.mock('@/components/shared/page-header', () => ({
  PageHeader: ({
    title,
    description,
    actions,
  }: {
    title: string;
    description?: string;
    actions?: ReactNode;
  }) => (
    <div>
      <h1>{title}</h1>
      {description ? <p>{description}</p> : null}
      {actions}
    </div>
  ),
}));

vi.mock('@/mutations/external-api-mutations', () => ({
  useCreateExternalApi: () => ({
    mutate: externalApiMutationState.createMutate,
    isPending: externalApiMutationState.createPending,
  }),
  useUpdateExternalApi: () => ({
    mutate: externalApiMutationState.updateMutate,
    isPending: externalApiMutationState.updatePending,
  }),
  useDeleteExternalApi: () => ({
    mutate: externalApiMutationState.deleteMutate,
    isPending: externalApiMutationState.deletePending,
  }),
  useTestExternalApi: () => ({
    mutate: externalApiMutationState.testMutate,
    isPending: externalApiMutationState.testPending,
  }),
}));

function mockJsonResponse(body: unknown, ok = true, status = 200): Response {
  return {
    ok,
    status,
    json: vi.fn().mockResolvedValue(body),
  } as unknown as Response;
}

const existingExternalApi = {
  id: 'api-existing',
  key: 'eligibility_api',
  name: 'Eligibility API',
  active: true,
  request: {
    method: 'POST' as const,
    urlTemplate: 'https://api.example.com/check',
    headers: [],
    bodyTemplate: {},
  },
  params: [],
  responseValidation: {
    mode: 'both' as const,
    http: { mode: 'any_2xx' as const, codes: [] },
    body: {
      expression: 'response.body.approved == true',
      schema: {},
      sampleResponseText: '',
    },
  },
  hasSecrets: false,
  version: 1,
  createdAt: '',
  updatedAt: '',
  createdBy: '',
  updatedBy: '',
};

describe('ExternalApiBuilder step 3', () => {
  beforeAll(() => {
    vi.stubGlobal('fetch', fetchMock);
  });

  beforeEach(() => {
    fetchMock.mockReset();
    fetchMock.mockImplementation(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes('/admin/external-apis/expression-profile')) {
        return mockJsonResponse({
          keywords: ['and', 'or', 'not', 'true', 'false', 'null', 'nil'],
          symbols: [
            { path: 'response', type: 'object', description: 'Response envelope.' },
            { path: 'response.status', type: 'number', description: 'HTTP status code.' },
            {
              path: 'response.header',
              type: 'object',
              description: 'Normalized response headers.',
            },
            { path: 'response.body', type: 'unknown', description: 'Parsed response body.' },
          ],
          actions: [
            {
              id: 'bool-eq-true',
              label: '== true',
              category: 'comparison',
              appliesTo: ['boolean'],
              template: '{{path}} == true',
              priority: 100,
            },
            {
              id: 'bool-eq-false',
              label: '== false',
              category: 'comparison',
              appliesTo: ['boolean'],
              template: '{{path}} == false',
              priority: 99,
            },
            {
              id: 'bool-ne-true',
              label: '!= true',
              category: 'comparison',
              appliesTo: ['boolean'],
              template: '{{path}} != true',
              priority: 98,
            },
            {
              id: 'bool-ne-false',
              label: '!= false',
              category: 'comparison',
              appliesTo: ['boolean'],
              template: '{{path}} != false',
              priority: 97,
            },
            {
              id: 'nullable-eq-null',
              label: '== null',
              category: 'literal',
              appliesTo: ['boolean', 'string', 'number', 'array', 'object', 'unknown', 'null'],
              template: '{{path}} == null',
              priority: 80,
            },
            {
              id: 'nullable-ne-null',
              label: '!= null',
              category: 'literal',
              appliesTo: ['boolean', 'string', 'number', 'array', 'object', 'unknown', 'null'],
              template: '{{path}} != null',
              priority: 79,
            },
          ],
        });
      }

      if (url.includes('/admin/external-apis/expression/validate')) {
        const body = init?.body ? JSON.parse(String(init.body)) : {};
        const expression = String(body.expression ?? '');
        return mockJsonResponse(
          expression === 'response.body.results =='
            ? { valid: false, error: 'invalid response condition' }
            : { valid: true },
        );
      }

      throw new Error(`Unhandled fetch in test: ${url}`);
    });
  });

  afterEach(() => {
    externalApiMutationState.createMutate.mockReset();
    externalApiMutationState.updateMutate.mockReset();
    externalApiMutationState.deleteMutate.mockReset();
    externalApiMutationState.testMutate.mockReset();
    externalApiMutationState.createPending = false;
    externalApiMutationState.updatePending = false;
    externalApiMutationState.deletePending = false;
    externalApiMutationState.testPending = false;
    navigateMock.mockReset();
    unsavedChangesState.handleBack.mockReset();
    unsavedChangesState.hookMock.mockReset();
    fetchMock.mockReset();
    vi.useRealTimers();
  });

  afterAll(() => {
    vi.unstubAllGlobals();
  });

  it('shows the empty schema card and opens the schema modal', async () => {
    const user = userEvent.setup();

    render(<ExternalApiBuilder />);

    await user.click(screen.getByRole('button', { name: 'steps.validation' }));

    expect(screen.getByRole('button', { name: 'actions.configureSchema' })).toBeInTheDocument();
    expect(screen.getByText('schemaModal.emptySchemaTitle')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'actions.configureSchema' }));

    expect(screen.getByText('schemaModal.title')).toBeInTheDocument();
  });

  it('keeps save disabled until there are persisted changes and pulses on first activation', async () => {
    const user = userEvent.setup();

    render(<ExternalApiBuilder />);

    const saveButton = screen.getByRole('button', { name: 'actions.save' });
    const nameInput = screen.getAllByRole('textbox')[0];

    expect(saveButton).toBeDisabled();

    await user.type(nameInput, 'Eligibility API');

    expect(saveButton).toBeEnabled();
    await waitFor(() => {
      expect(saveButton).toHaveClass('animate-pulse');
    });
  });

  it('resets the draft to clean state after a successful update', async () => {
    const user = userEvent.setup();
    externalApiMutationState.updateMutate.mockImplementation((_variables, options) => {
      options?.onSuccess?.({
        ...existingExternalApi,
        name: 'Eligibility API Updated',
      });
    });

    render(<ExternalApiBuilder externalApi={existingExternalApi} />);

    const saveButton = screen.getByRole('button', { name: 'actions.save' });
    const nameInput = screen.getAllByRole('textbox')[0];

    expect(saveButton).toBeDisabled();

    await user.type(nameInput, ' Updated');

    expect(saveButton).toBeEnabled();

    await user.click(saveButton);

    expect(externalApiMutationState.updateMutate).toHaveBeenCalledWith(
      expect.objectContaining({
        key: existingExternalApi.key,
        data: expect.objectContaining({
          name: 'Eligibility API Updated',
        }),
      }),
      expect.any(Object),
    );
    await waitFor(() => expect(saveButton).toBeDisabled());
    expect(navigateMock).toHaveBeenCalledWith(
      expect.objectContaining({
        to: '/settings/external-apis/$key',
        params: { key: existingExternalApi.key },
      }),
    );
  });

  it('routes back through the unsaved changes guard when the draft is dirty', async () => {
    const user = userEvent.setup();

    render(<ExternalApiBuilder />);

    const nameInput = screen.getAllByRole('textbox')[0];

    await user.type(nameInput, 'Eligibility API');
    await user.click(screen.getByRole('button', { name: 'actions.back' }));

    expect(unsavedChangesState.handleBack).toHaveBeenCalledTimes(1);
    expect(unsavedChangesState.hookMock).toHaveBeenLastCalledWith(
      expect.objectContaining({
        isDirty: true,
        backTo: '/settings/external-apis',
        blockNavigation: true,
      }),
    );
    expect(navigateMock).not.toHaveBeenCalled();
  });

  it('keeps previous and next in the tab header disabled at the ends and never replaces next with save', async () => {
    const user = userEvent.setup();

    render(<ExternalApiBuilder />);

    expect(screen.getByRole('button', { name: 'actions.previous' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'actions.next' })).toBeEnabled();
    expect(screen.getAllByRole('button', { name: 'actions.save' })).toHaveLength(1);

    await user.click(screen.getByRole('button', { name: 'actions.next' }));
    await user.click(screen.getByRole('button', { name: 'actions.next' }));
    await user.click(screen.getByRole('button', { name: 'actions.next' }));

    expect(screen.getByRole('button', { name: 'steps.test' })).toHaveAttribute(
      'data-active',
      'true',
    );
    expect(screen.getByRole('button', { name: 'actions.previous' })).toBeEnabled();
    expect(screen.getByRole('button', { name: 'actions.next' })).toBeDisabled();
    expect(screen.getAllByRole('button', { name: 'actions.save' })).toHaveLength(1);
  });

  it('stretches the request body editor and secrets panel to the available height', async () => {
    const user = userEvent.setup();
    const { container } = render(<ExternalApiBuilder />);

    await user.click(screen.getByRole('button', { name: 'steps.request' }));

    const bodyTextarea = container.querySelector('textarea');
    expect(bodyTextarea?.parentElement).toHaveClass('h-full', 'min-h-[400px]');

    const secretsPanel = screen.getByText('sections.secrets').closest('.fe-panel-subtle');
    expect(secretsPanel).toHaveClass('h-full', 'min-h-[400px]');
  });

  it('shows the schema preview card when an api already has schema configured', async () => {
    const user = userEvent.setup();

    render(
      <ExternalApiBuilder
        externalApi={{
          ...existingExternalApi,
          id: 'api-1',
          responseValidation: {
            mode: 'both',
            http: { mode: 'any_2xx', codes: [] },
            body: {
              expression: 'response.body.approved == true',
              schema: {
                type: 'object',
                properties: {
                  approved: { type: 'boolean' },
                },
              },
              sampleResponseText: '{\n  "approved": true\n}',
            },
          },
        }}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'steps.validation' }));

    expect(screen.getByRole('button', { name: 'actions.editSchema' })).toBeInTheDocument();
    expect(screen.getByText(/"approved": \{/)).toBeInTheDocument();
    expect(screen.getByText(/"type": "boolean"/)).toBeInTheDocument();
  });

  it('shows only url variables in detected parameters and exposes the variables manager', () => {
    render(
      <ExternalApiBuilder
        externalApi={{
          ...existingExternalApi,
          id: 'api-3',
          request: {
            method: 'POST',
            urlTemplate: 'https://api.{{env}}.example.com/users/{{user_id}}?campus={{campus_code}}',
            headers: [{ keyTemplate: 'Authorization', valueTemplate: 'Bearer {{token_value}}' }],
            bodyTemplate: {
              channel: '{{channel_code}}',
            },
          },
          params: [
            { name: 'env', type: 'string', required: true, locations: ['url'], urlKind: 'domain' },
            {
              name: 'user_id',
              type: 'string',
              required: true,
              locations: ['url'],
              urlKind: 'path',
            },
            {
              name: 'campus_code',
              type: 'string',
              required: false,
              locations: ['url'],
              urlKind: 'query',
            },
            { name: 'token_value', type: 'string', required: false, locations: ['header'] },
            { name: 'channel_code', type: 'string', required: false, locations: ['body'] },
          ],
          responseValidation: {
            mode: 'both',
            http: { mode: 'any_2xx', codes: [] },
            body: {
              expression: 'response.body.approved == true',
              schema: {},
              sampleResponseText: '',
            },
          },
        }}
      />,
    );

    expect(screen.getByRole('button', { name: 'actions.variablesManager' })).toBeInTheDocument();
    expect(screen.getAllByText('{{env}}').length).toBeGreaterThan(0);
    expect(screen.getAllByText('{{user_id}}').length).toBeGreaterThan(0);
    expect(screen.getAllByText('{{campus_code}}').length).toBeGreaterThan(0);
    expect(screen.queryByText('{{token_value}}')).not.toBeInTheDocument();
    expect(screen.queryByText('{{channel_code}}')).not.toBeInTheDocument();
  });

  it('shares the real test result between step 4 and the schema modal', async () => {
    const user = userEvent.setup();
    externalApiMutationState.testMutate.mockImplementation((_variables, options) => {
      options?.onSuccess?.({
        ok: true,
        attempted: true,
        httpStatus: 200,
        details: {
          request: {
            method: 'POST',
            url: 'https://api.example.com/check',
            headers: {},
            body: { user_id: '123' },
          },
          responseBody: { approved: true },
          responseText: '{"approved":true}',
          responseHeaders: { 'Content-Type': 'application/json' },
        },
      });
    });

    render(
      <ExternalApiBuilder
        externalApi={{
          ...existingExternalApi,
          id: 'api-2',
          params: [{ name: 'user_id', type: 'string', required: false, locations: ['body'] }],
          responseValidation: {
            mode: 'both',
            http: { mode: 'any_2xx', codes: [] },
            body: {
              expression: 'response.body.approved == true',
              schema: {},
              sampleResponseText: '',
            },
          },
        }}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'steps.test' }));
    await user.click(screen.getByRole('button', { name: 'actions.test' }));

    expect(await screen.findByText('actions.useResponseAsSchema')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'steps.validation' }));
    await user.click(screen.getByRole('button', { name: 'actions.configureSchema' }));
    await user.click(screen.getByRole('button', { name: 'schemaModal.tabs.test' }));

    expect(screen.getAllByText('actions.useResponseAsSchema').length).toBeGreaterThan(0);
    expect(screen.getByText(/https:\/\/api\.example\.com\/check/)).toBeInTheDocument();
  });

  it('runs schema test without requiring a response expression', async () => {
    const user = userEvent.setup();

    render(
      <ExternalApiBuilder
        externalApi={{
          ...existingExternalApi,
          id: 'api-4',
          params: [{ name: 'user_id', type: 'string', required: false, locations: ['body'] }],
          responseValidation: {
            mode: 'both',
            http: { mode: 'any_2xx', codes: [] },
            body: {
              expression: '',
              schema: {},
              sampleResponseText: '',
            },
          },
        }}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'steps.validation' }));
    await user.click(screen.getByRole('button', { name: 'actions.configureSchema' }));
    await user.click(screen.getByRole('button', { name: 'schemaModal.tabs.test' }));
    await user.click(screen.getByRole('button', { name: 'actions.test' }));

    expect(externalApiMutationState.testMutate).toHaveBeenCalledWith(
      expect.objectContaining({
        responseValidation: {
          mode: 'httpCode',
          http: { mode: 'any_2xx', codes: [] },
          body: {
            expression: '',
            schema: { type: 'object' },
            sampleResponseText: '{}',
          },
        },
      }),
      expect.any(Object),
    );
  });

  it('mounts the CodeMirror expression editor with the current expression', async () => {
    const user = userEvent.setup();

    render(
      <ExternalApiBuilder
        externalApi={{
          ...existingExternalApi,
          id: 'api-5',
          params: [],
          responseValidation: {
            mode: 'both',
            http: { mode: 'any_2xx', codes: [] },
            body: {
              expression: 'response.body.results.campus_premium_pack.',
              schema: {
                type: 'object',
                properties: {
                  results: {
                    type: 'object',
                    properties: {
                      campus_premium_pack: { type: 'boolean' },
                    },
                  },
                },
              },
              sampleResponseText: '{\n  "results": {\n    "campus_premium_pack": true\n  }\n}',
            },
          },
        }}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'steps.validation' }));

    await waitFor(() => {
      expect(document.querySelector('.cm-content')).not.toBeNull();
    });

    const editor = document.querySelector('.cm-content') as HTMLElement;
    await user.click(editor);

    expect(editor.textContent).toContain('response.body.results.campus_premium_pack.');
  });

  it('shows inline validation errors from the expression validator', async () => {
    const user = userEvent.setup();

    render(
      <ExternalApiBuilder
        externalApi={{
          ...existingExternalApi,
          id: 'api-6',
          params: [],
          responseValidation: {
            mode: 'both',
            http: { mode: 'any_2xx', codes: [] },
            body: {
              expression: 'response.body.results ==',
              schema: {
                type: 'object',
                properties: {
                  results: { type: 'object' },
                },
              },
              sampleResponseText: '{\n  "results": {}\n}',
            },
          },
        }}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'steps.validation' }));

    expect(await screen.findByText('invalid response condition')).toBeInTheDocument();
  });

  it('includes manual variables in test inputs and serializes them separately from params', async () => {
    const user = userEvent.setup();

    render(
      <ExternalApiBuilder
        externalApi={{
          ...existingExternalApi,
          id: 'api-7',
          request: {
            method: 'POST',
            urlTemplate: 'https://api.example.com/check',
            headers: [],
            bodyTemplate: {
              user_id: '{{user_id}}',
            },
          },
          params: [{ name: 'user_id', type: 'string', required: false, locations: ['body'] }],
          expressionVariables: [{ name: 'campus_code', type: 'string', required: true }],
          responseValidation: {
            mode: 'both',
            http: { mode: 'any_2xx', codes: [] },
            body: {
              expression: 'vars.campus_code == "north"',
              schema: { type: 'object' },
              sampleResponseText: '{}',
            },
          },
        }}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'steps.test' }));

    expect(screen.getByText(/campus_code/)).toBeInTheDocument();

    await user.type(screen.getByLabelText(/campus_code/), 'north');
    await user.click(screen.getByRole('button', { name: 'actions.test' }));

    expect(externalApiMutationState.testMutate).toHaveBeenCalledWith(
      expect.objectContaining({
        params: [{ name: 'user_id', type: 'string', required: false, locations: ['body'] }],
        expressionVariables: [{ name: 'campus_code', type: 'string', required: true }],
        paramValues: { campus_code: 'north' },
      }),
      expect.any(Object),
    );
  });
});

import userEvent from '@testing-library/user-event';

import { inferJsonSchema } from './external-api-builder-utils';
import { ResponseSchemaModal } from './response-schema-modal';

import type { ExternalApiTestResponse } from '@/api/types';

import { fireEvent, render, screen } from '@/test/test-utils';

describe('ResponseSchemaModal', () => {
  const baseProps = {
    inputs: [],
    paramInputValues: {},
    onParamInputValuesChange: vi.fn(),
    testResult: null as ExternalApiTestResponse | null,
    isTesting: false,
    onRunTest: vi.fn(),
  };

  it('infers and returns schema from a pasted example', async () => {
    const user = userEvent.setup();
    const onApply = vi.fn();

    render(
      <ResponseSchemaModal
        open
        onOpenChange={vi.fn()}
        schema={{}}
        sampleResponseText=""
        {...baseProps}
        onApply={onApply}
      />,
    );

    fireEvent.change(screen.getByLabelText('fields.sampleResponse'), {
      target: { value: '{\n  "approved": true,\n  "amount": 5\n}' },
    });

    await user.click(screen.getByRole('button', { name: 'actions.confirmSchema' }));

    expect(onApply).toHaveBeenCalledWith({
      schema: inferJsonSchema({ approved: true, amount: 5 }),
      sampleResponseText: '{\n  "approved": true,\n  "amount": 5\n}',
    });
  });

  it('initializes the sample editor with an empty object when there is no sample or schema', () => {
    render(
      <ResponseSchemaModal
        open
        onOpenChange={vi.fn()}
        schema={{}}
        sampleResponseText=""
        {...baseProps}
        onApply={vi.fn()}
      />,
    );

    expect(screen.getByLabelText('fields.sampleResponse')).toHaveValue('{}');
    expect(screen.getByTestId('response-schema-modal-content')).toHaveClass('h-[560px]');
  });

  it('uses the last test response as schema when available', async () => {
    const user = userEvent.setup();
    const onApply = vi.fn();
    const responseBody = { approved: true, user: { id: '123' } };
    const testResult: ExternalApiTestResponse = {
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
        responseBody,
        responseText: JSON.stringify(responseBody),
        responseHeaders: { 'Content-Type': 'application/json' },
      },
    };

    render(
      <ResponseSchemaModal
        open
        onOpenChange={vi.fn()}
        schema={{}}
        sampleResponseText=""
        {...baseProps}
        testResult={testResult}
        onApply={onApply}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'schemaModal.tabs.test' }));
    await user.click(screen.getByRole('button', { name: 'actions.useResponseAsSchema' }));
    expect(screen.getByLabelText('fields.sampleResponse')).toHaveValue(
      JSON.stringify(responseBody, null, 2),
    );
    await user.click(screen.getByRole('button', { name: 'actions.confirmSchema' }));

    expect(onApply).toHaveBeenCalledWith({
      schema: inferJsonSchema(responseBody),
      sampleResponseText: JSON.stringify(responseBody, null, 2),
    });
  });

  it('delegates test execution from the modal tab', async () => {
    const user = userEvent.setup();
    const onRunTest = vi.fn();

    render(
      <ResponseSchemaModal
        open
        onOpenChange={vi.fn()}
        schema={{}}
        sampleResponseText=""
        {...baseProps}
        onRunTest={onRunTest}
        onApply={vi.fn()}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'schemaModal.tabs.test' }));
    await user.click(screen.getByRole('button', { name: 'actions.test' }));

    expect(onRunTest).toHaveBeenCalledTimes(1);
  });

  it('renders manual variables in the shared test tab inputs', async () => {
    const user = userEvent.setup();

    render(
      <ResponseSchemaModal
        open
        onOpenChange={vi.fn()}
        schema={{}}
        sampleResponseText=""
        {...baseProps}
        inputs={[
          { name: 'campus_code', type: 'string', required: true, origin: 'manual' },
        ]}
        onApply={vi.fn()}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'schemaModal.tabs.test' }));

    expect(screen.getByLabelText(/campus_code/)).toBeInTheDocument();
  });

  it('shows the shared empty test state and disables save when there is no test response', async () => {
    const user = userEvent.setup();

    render(
      <ResponseSchemaModal
        open
        onOpenChange={vi.fn()}
        schema={{}}
        sampleResponseText=""
        {...baseProps}
        onApply={vi.fn()}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'schemaModal.tabs.test' }));

    expect(screen.getByText('empty.testPending')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'actions.confirmSchema' })).toBeDisabled();
  });

  it('hides useResponseAsSchema when the response is not a successful json object', async () => {
    const user = userEvent.setup();
    const testResult: ExternalApiTestResponse = {
      ok: false,
      attempted: true,
      httpStatus: 500,
      details: {
        request: {
          method: 'POST',
          url: 'https://api.example.com/check',
          headers: {},
          body: { user_id: '123' },
        },
        responseBody: 'plain text',
        responseText: 'plain text',
        responseHeaders: { 'Content-Type': 'text/plain' },
      },
    };

    render(
      <ResponseSchemaModal
        open
        onOpenChange={vi.fn()}
        schema={{}}
        sampleResponseText=""
        {...baseProps}
        testResult={testResult}
        onApply={vi.fn()}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'schemaModal.tabs.test' }));

    expect(screen.queryByRole('button', { name: 'actions.useResponseAsSchema' })).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'actions.confirmSchema' })).toBeDisabled();
  });

  it('does not render the preview side panel in the visual tab', async () => {
    const user = userEvent.setup();

    render(
      <ResponseSchemaModal
        open
        onOpenChange={vi.fn()}
        schema={{}}
        sampleResponseText=""
        {...baseProps}
        onApply={vi.fn()}
      />,
    );

    await user.click(screen.getByRole('button', { name: 'schemaModal.tabs.visual' }));

    expect(screen.queryByText('sections.bodyPreview')).not.toBeInTheDocument();
  });
});

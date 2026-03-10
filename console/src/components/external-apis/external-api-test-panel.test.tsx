import userEvent from '@testing-library/user-event';

import { ExternalApiTestPanel } from './external-api-test-panel';

import type { ComponentProps } from 'react';
import { act } from 'react';

import { render, screen } from '@/test/test-utils';

function buildPanel(props: Partial<ComponentProps<typeof ExternalApiTestPanel>> = {}) {
  return (
    <ExternalApiTestPanel
      inputs={[]}
      paramInputValues={{}}
      onParamInputValuesChange={vi.fn()}
      testResult={null}
      isTesting={false}
      onRunTest={vi.fn()}
      previewRequest={{
        method: 'POST',
        url: 'https://example.com/check',
        headers: { Authorization: 'Bearer token' },
        body: { user_id: '123' },
      }}
      {...props}
    />
  );
}

function renderPanel(props: Partial<ComponentProps<typeof ExternalApiTestPanel>> = {}) {
  return render(buildPanel(props), {
    providerProps: {
      locale: 'en',
      namespaces: ['external-apis'],
      useRealTranslations: true,
    },
  });
}

function buildTestResult(
  overrides: Partial<ComponentProps<typeof ExternalApiTestPanel>['testResult']> = {},
): ComponentProps<typeof ExternalApiTestPanel>['testResult'] {
  return {
    ok: true,
    attempted: true,
    httpStatus: 200,
    details: {
      request: {
        method: 'POST',
        url: 'https://example.com/check',
        headers: { Authorization: 'Bearer token' },
        body: { user_id: '123' },
      },
      responseHeaders: { 'Content-Type': 'application/json' },
      responseText: '{"approved":true}',
      responseBody: { approved: true },
      evaluations: {
        final: { mode: 'both' as const, passed: true },
        http: {
          applied: true,
          passed: true,
          mode: 'any_2xx' as const,
          expectedCodes: [],
          actualStatus: 200,
        },
        expression: {
          applied: true,
          passed: true,
          expression: 'response.body.approved == true',
          resolvedExpression: 'true == true',
          error: null,
        },
      },
    },
    ...overrides,
  };
}

describe('ExternalApiTestPanel', () => {
  it('stretches the request and response panes to the available height', () => {
    renderPanel();

    const requestBlock = screen.getByTestId('external-api-request-block');
    const responseBlock = screen.getByTestId('external-api-response-block');
    const requestBody = requestBlock.querySelector('pre');
    const responseBody = responseBlock.querySelector('pre');

    expect(screen.getByTestId('external-api-traffic-grid')).toHaveClass('h-full', 'flex-1');
    expect(requestBlock).toHaveClass('flex', 'h-full', 'min-h-[420px]', 'flex-col');
    expect(responseBlock).toHaveClass('flex', 'h-full', 'min-h-[420px]', 'flex-col');
    expect(requestBody).toHaveClass('flex-1', 'overflow-auto');
    expect(responseBody).toHaveClass('flex-1', 'overflow-auto');
  });

  it('shows a disabled debug icon without pulse while the test is running', async () => {
    const user = userEvent.setup();

    renderPanel({ isTesting: true });

    const debugButton = screen.getByTestId('external-api-evaluation-debug-button');
    expect(debugButton).toBeDisabled();
    expect(debugButton).not.toHaveClass('animate-pulse');

    const triggerWrapper = debugButton.parentElement;
    if (!triggerWrapper) {
      throw new Error('missing debug trigger wrapper');
    }

    await user.hover(triggerWrapper);

    expect(
      await screen.findAllByText('Evaluation details will be available when the test finishes.'),
    ).not.toHaveLength(0);
  });

  it('pulses for 3 seconds when a new evaluation arrives', async () => {
    vi.useFakeTimers();

    try {
      const view = renderPanel({ isTesting: true });

      await act(async () => {
        view.rerender(buildPanel({ testResult: buildTestResult() }));
      });

      const debugButton = screen.getByTestId('external-api-evaluation-debug-button');
      expect(debugButton).toHaveClass('animate-pulse');

      act(() => {
        vi.advanceTimersByTime(2999);
      });
      expect(debugButton).toHaveClass('animate-pulse');

      act(() => {
        vi.advanceTimersByTime(1);
      });
      expect(debugButton).not.toHaveClass('animate-pulse');
    } finally {
      vi.useRealTimers();
    }
  });

  it('opens the evaluation modal when the test returns evaluation details', async () => {
    const user = userEvent.setup();

    renderPanel({
      testResult: buildTestResult(),
    });

    const debugButton = screen.getByTestId('external-api-evaluation-debug-button');
    const statusRow = debugButton.closest('div.flex');
    if (!statusRow) {
      throw new Error('missing evaluation status row');
    }

    expect(debugButton).not.toHaveClass('animate-pulse');
    expect(statusRow.firstElementChild?.querySelector('button')).toBe(debugButton);

    await user.click(debugButton);

    expect(await screen.findByTestId('external-api-evaluation-modal')).toBeInTheDocument();
    expect(screen.getByText('Test evaluations')).toBeInTheDocument();
    expect(screen.getByText('HTTP validation')).toBeInTheDocument();
    expect(screen.getByText('Expression validation')).toBeInTheDocument();
    expect(screen.getByText('response.body.approved == true')).toBeInTheDocument();
    expect(screen.getByText('true == true')).toBeInTheDocument();
  });

  it('shows expression validation as not applicable in http-only mode', async () => {
    const user = userEvent.setup();

    renderPanel({
      testResult: buildTestResult({
        details: {
          request: {
            method: 'GET',
            url: 'https://example.com/check',
            headers: {},
            body: undefined,
          },
          responseHeaders: { 'Content-Type': 'application/json' },
          responseText: '{"approved":true}',
          responseBody: { approved: true },
          evaluations: {
            final: { mode: 'httpCode', passed: true },
            http: {
              applied: true,
              passed: true,
              mode: 'status_codes',
              expectedCodes: [200, 204],
              actualStatus: 200,
            },
            expression: {
              applied: false,
              passed: false,
              expression: '',
              resolvedExpression: null,
              error: null,
            },
          },
        },
      }),
    });

    await user.click(screen.getByTestId('external-api-evaluation-debug-button'));

    expect(await screen.findByText('Specific Codes...')).toBeInTheDocument();
    expect(screen.getByText('200, 204')).toBeInTheDocument();
    expect(screen.getAllByText('Not applicable')).not.toHaveLength(0);
  });

  it('shows HTTP as not applicable and renders expression errors in body-only mode', async () => {
    const user = userEvent.setup();

    renderPanel({
      testResult: buildTestResult({
        ok: false,
        details: {
          request: {
            method: 'POST',
            url: 'https://example.com/check',
            headers: { Authorization: 'Bearer token' },
            body: { user_id: '123' },
          },
          responseHeaders: { 'Content-Type': 'application/json' },
          responseText: '{"approved":true}',
          responseBody: { approved: true },
          evaluations: {
            final: { mode: 'responseBody', passed: false },
            http: {
              applied: false,
              passed: false,
              mode: 'any_2xx',
              expectedCodes: [],
              actualStatus: 500,
            },
            expression: {
              applied: true,
              passed: false,
              expression: 'response.body.approved + 1',
              resolvedExpression: 'true + 1',
              error: 'response condition returned non-boolean: int',
            },
          },
        },
      }),
    });

    await user.click(screen.getByTestId('external-api-evaluation-debug-button'));

    expect(await screen.findAllByText('Response body only')).not.toHaveLength(0);
    expect(screen.getAllByText('Not applicable')).not.toHaveLength(0);
    expect(screen.getByText('response.body.approved + 1')).toBeInTheDocument();
    expect(screen.getByText('true + 1')).toBeInTheDocument();
    expect(screen.getByText('response condition returned non-boolean: int')).toBeInTheDocument();
  });
});

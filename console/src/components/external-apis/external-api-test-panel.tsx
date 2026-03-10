import { ListChecks, Play } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import {
  formatExternalApiPreview,
  getExternalApiTestResponseBody,
} from './external-api-test-utils';

import type {
  ExternalApiInputDescriptor,
  RenderedDraftRequest,
} from './external-api-builder-utils';
import type {
  ExternalApiHTTPValidationMode,
  ExternalApiTestDetails,
  ExternalApiTestEvaluations,
  ExternalApiTestResponse,
} from '@/api/types';
import type { Dispatch, ReactNode, SetStateAction } from 'react';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { cn } from '@/lib/utils';

interface ExternalApiTestPanelProps {
  inputs: ExternalApiInputDescriptor[];
  paramInputValues: Record<string, string>;
  onParamInputValuesChange: Dispatch<SetStateAction<Record<string, string>>>;
  testResult: ExternalApiTestResponse | null;
  isTesting: boolean;
  onRunTest: () => void;
  onUseResponseAsSchema?: () => void;
  className?: string;
  title?: string;
  description?: string;
  variant?: 'default' | 'modal-dark';
  previewRequest?: RenderedDraftRequest;
}

export function ExternalApiTestPanel({
  inputs,
  paramInputValues,
  onParamInputValuesChange,
  testResult,
  isTesting,
  onRunTest,
  onUseResponseAsSchema,
  className,
  title,
  description,
  variant = 'default',
  previewRequest,
}: ExternalApiTestPanelProps) {
  const { t } = useTranslation('external-apis');
  const [evaluationModalOpen, setEvaluationModalOpen] = useState(false);
  const [highlightEvaluations, setHighlightEvaluations] = useState(false);
  const previousEvaluationsRef = useRef<ExternalApiTestEvaluations | undefined>(undefined);
  const hasInitializedEvaluationsRef = useRef(false);
  const evaluationPulseTimeoutRef = useRef<number | null>(null);
  const requestPreview = getRequestPreview(testResult?.details, previewRequest);
  const responseBody = getExternalApiTestResponseBody(testResult);
  const responseText = getResponseText(testResult?.details, responseBody);
  const responseHeaders = testResult?.details?.responseHeaders ?? {};
  const evaluations = testResult?.details?.evaluations;
  const httpStatus = testResult?.httpStatus;
  const hasPreviewRequest =
    Boolean(requestPreview.method) ||
    Boolean(requestPreview.url) ||
    Object.keys(requestPreview.headers).length > 0 ||
    requestPreview.body != null;
  const showDebugStatus = variant === 'default' && (isTesting || testResult != null);
  const debugDisabled = !evaluations;
  const debugTooltip = debugDisabled
    ? t('evaluationDebug.pendingTooltip')
    : t('evaluationDebug.openTooltip');

  useEffect(() => {
    if (!hasInitializedEvaluationsRef.current) {
      hasInitializedEvaluationsRef.current = true;
      previousEvaluationsRef.current = evaluations;
      return;
    }

    const previousEvaluations = previousEvaluationsRef.current;
    previousEvaluationsRef.current = evaluations;

    if (!evaluations || evaluations === previousEvaluations) {
      if (!evaluations) {
        setHighlightEvaluations(false);
      }
      return;
    }

    setHighlightEvaluations(true);

    if (evaluationPulseTimeoutRef.current != null) {
      window.clearTimeout(evaluationPulseTimeoutRef.current);
    }

    evaluationPulseTimeoutRef.current = window.setTimeout(() => {
      setHighlightEvaluations(false);
      evaluationPulseTimeoutRef.current = null;
    }, 3000);
  }, [evaluations]);

  useEffect(() => {
    return () => {
      if (evaluationPulseTimeoutRef.current != null) {
        window.clearTimeout(evaluationPulseTimeoutRef.current);
      }
    };
  }, []);

  if (variant === 'modal-dark') {
    return (
      <div className={cn('grid h-full gap-6 xl:grid-cols-12', className)}>
        <div className="flex flex-col xl:col-span-4 xl:border-r xl:border-border-strong xl:pr-6">
          <h3 className="text-content-body mb-4 border-b border-border pb-2 text-sm font-medium">
            {title ?? t('sections.testInputs')}
          </h3>

          <div className="flex-1 overflow-y-auto pr-2">
            {inputs.length === 0 ? (
              <p className="text-content-subtle text-sm italic">{t('empty.params')}</p>
            ) : (
              <div className="space-y-4">
                {inputs.map((param) => (
                  <div key={param.name}>
                    <label
                      htmlFor={`external-api-test-input-${param.name}`}
                      className="text-content-muted mb-1 block text-xs"
                    >
                      {param.name} <span className="text-content-subtle">({param.type})</span>
                      {param.required ? <span className="text-destructive ml-1">*</span> : null}
                    </label>
                    {param.type === 'bool' ? (
                      <select
                        id={`external-api-test-input-${param.name}`}
                        value={paramInputValues[param.name] ?? ''}
                        onChange={(event) =>
                          onParamInputValuesChange((current) => ({
                            ...current,
                            [param.name]: event.target.value,
                          }))
                        }
                        className="fe-editor w-full px-3 py-2 pr-10 text-sm focus:border-ring focus:outline-none"
                      >
                        <option value="">-</option>
                        <option value="true">true</option>
                        <option value="false">false</option>
                      </select>
                    ) : (
                      <input
                        type={param.type === 'number' ? 'number' : 'text'}
                        value={paramInputValues[param.name] ?? ''}
                        onChange={(event) =>
                          onParamInputValuesChange((current) => ({
                            ...current,
                            [param.name]: event.target.value,
                          }))
                        }
                        className="fe-editor w-full px-3 py-2 text-sm focus:border-ring focus:outline-none"
                        placeholder={t('placeholders.paramValueFor', { name: `{{${param.name}}}` })}
                      />
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="mt-4 border-t border-border-strong pt-4">
            <Button type="button" onClick={onRunTest} disabled={isTesting} className="w-full">
              <Play className="h-4 w-4" />
              {isTesting ? t('actions.testing') : t('actions.test')}
            </Button>
          </div>
        </div>

        <div className="flex h-full flex-col xl:col-span-8">
          <div className="mb-2 flex items-center justify-between">
            <span className="text-content-body text-sm font-medium">
              {t('sections.trafficInspection')}
            </span>
            {responseBody !== undefined && onUseResponseAsSchema ? (
              <Button
                type="button"
                variant="outline"
                onClick={onUseResponseAsSchema}
                className="h-8 border-success-soft bg-success-soft px-3 text-xs text-success-soft-foreground hover:bg-success-soft/80"
              >
                {t('actions.useResponseAsSchema')}
              </Button>
            ) : null}
          </div>

          <div className="fe-traffic-shell flex flex-1 flex-col overflow-hidden">
            {!testResult && !hasPreviewRequest ? (
              <div className="text-content-subtle flex flex-1 items-center justify-center text-sm italic">
                {t('empty.testPending')}
              </div>
            ) : (
              <div className="flex h-full flex-col">
                <TrafficBlock
                  title={t('sections.requestSent')}
                  header={
                    requestPreview.method && requestPreview.url
                      ? `${requestPreview.method} ${requestPreview.url}`
                      : undefined
                  }
                  headers={requestPreview.headers}
                  body={requestPreview.body}
                  emptyText={t('empty.requestPreview')}
                  variant="modal-dark-request"
                />
                <TrafficBlock
                  title={t('sections.responseReceived')}
                  header={
                    typeof httpStatus === 'number'
                      ? `${t('fields.httpStatus')} ${httpStatus}`
                      : undefined
                  }
                  headers={responseHeaders}
                  body={responseText}
                  emptyText={t('empty.responsePreview')}
                  variant={
                    httpStatus != null && httpStatus < 300
                      ? 'modal-dark-response-ok'
                      : 'modal-dark-response-error'
                  }
                  statusBadge={typeof httpStatus === 'number' ? `STATUS ${httpStatus}` : undefined}
                />
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={cn('grid min-h-0 flex-1 gap-6 xl:grid-cols-[340px_1fr]', className)}>
      <div className="flex min-h-0 flex-col gap-4">
        <div className="shrink-0">
          <h3 className="text-sm font-semibold">{title ?? t('sections.testInputs')}</h3>
          <p className="text-muted-foreground text-sm">{description ?? t('hints.testInputs')}</p>
        </div>

        {inputs.length === 0 ? (
          <p className="text-muted-foreground text-sm">{t('empty.params')}</p>
        ) : (
          <div className="min-h-0 flex-1 overflow-y-auto px-1 py-1">
            <div className="space-y-3">
              {inputs.map((param) => (
                <div key={param.name} className="space-y-2">
                  <div className="flex items-center gap-2">
                    <Label htmlFor={`external-api-test-input-${param.name}`}>{param.name}</Label>
                    <Badge variant={param.required ? 'destructive' : 'secondary'}>
                      {param.required ? t('badges.required') : t('badges.optional')}
                    </Badge>
                  </div>
                  {param.type === 'bool' ? (
                    <select
                      id={`external-api-test-input-${param.name}`}
                      value={paramInputValues[param.name] ?? ''}
                      onChange={(event) =>
                        onParamInputValuesChange((current) => ({
                          ...current,
                          [param.name]: event.target.value,
                        }))
                      }
                      className="h-10 w-full rounded-md border bg-background px-3 pr-10 text-sm"
                    >
                      <option value="">-</option>
                      <option value="true">true</option>
                      <option value="false">false</option>
                    </select>
                  ) : (
                    <Input
                      id={`external-api-test-input-${param.name}`}
                      value={paramInputValues[param.name] ?? ''}
                      onChange={(event) =>
                        onParamInputValuesChange((current) => ({
                          ...current,
                          [param.name]: event.target.value,
                        }))
                      }
                      placeholder={
                        param.type === 'any'
                          ? t('placeholders.requestBody')
                          : t('placeholders.paramValue')
                      }
                    />
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        <Button
          type="button"
          className="mt-auto w-full shrink-0"
          onClick={onRunTest}
          disabled={isTesting}
        >
          <Play className="mr-2 h-4 w-4" />
          {isTesting ? t('actions.testing') : t('actions.test')}
        </Button>
      </div>

      <div className="flex min-h-0 flex-1 flex-col gap-4">
        <div className="flex items-center justify-between gap-3">
          <div>
            <h3 className="text-sm font-semibold">{t('sections.trafficInspection')}</h3>
            <p className="text-muted-foreground text-sm">{t('hints.testInspection')}</p>
          </div>

          {responseBody !== undefined && onUseResponseAsSchema ? (
            <Button type="button" variant="outline" size="sm" onClick={onUseResponseAsSchema}>
              {t('actions.useResponseAsSchema')}
            </Button>
          ) : null}
        </div>

        {showDebugStatus ? (
          <div className="flex flex-wrap items-center gap-2">
            <TooltipProvider delayDuration={120}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span
                    tabIndex={debugDisabled ? 0 : undefined}
                    className="relative inline-flex h-7 w-7 items-center justify-center"
                  >
                    {highlightEvaluations ? (
                      <>
                        <span
                          aria-hidden
                          className="absolute -inset-1 rounded-full bg-primary/20 animate-ping"
                        />
                        <span
                          aria-hidden
                          className="absolute -inset-1 rounded-full border border-primary/40 animate-pulse"
                        />
                      </>
                    ) : null}
                    <button
                      type="button"
                      data-testid="external-api-evaluation-debug-button"
                      aria-label={t('evaluationDebug.buttonLabel')}
                      onClick={() => {
                        if (!debugDisabled) {
                          setEvaluationModalOpen(true);
                        }
                      }}
                      disabled={debugDisabled}
                      className={cn(
                        'relative z-10 inline-flex h-7 w-7 items-center justify-center rounded-full border border-border bg-background text-content-muted transition-colors hover:bg-accent hover:text-accent-foreground disabled:cursor-not-allowed disabled:opacity-70',
                        highlightEvaluations &&
                          'animate-pulse border-primary/60 bg-primary/10 text-primary shadow-md shadow-primary/30',
                      )}
                    >
                      <ListChecks className="h-4 w-4" />
                    </button>
                  </span>
                </TooltipTrigger>
                <TooltipContent>{debugTooltip}</TooltipContent>
              </Tooltip>
            </TooltipProvider>

            <Badge
              variant={isTesting ? 'neutral-soft' : testResult?.ok ? 'success-soft' : 'danger-soft'}
            >
              {isTesting
                ? t('actions.testing')
                : testResult?.ok
                  ? t('badges.passed')
                  : t('badges.failed')}
            </Badge>
            {typeof httpStatus === 'number' ? (
              <Badge variant="neutral-soft">HTTP {httpStatus}</Badge>
            ) : null}
          </div>
        ) : null}

        {!testResult ? (
          hasPreviewRequest ? (
            <div className="flex min-h-0 flex-1 overflow-hidden rounded-md border bg-muted/10">
              <div
                data-testid="external-api-traffic-grid"
                className="grid h-full flex-1 gap-px bg-border xl:grid-cols-2"
              >
                <TrafficBlock
                  testId="external-api-request-block"
                  title={t('sections.requestSent')}
                  header={
                    requestPreview.method && requestPreview.url
                      ? `${requestPreview.method} ${requestPreview.url}`
                      : undefined
                  }
                  headers={requestPreview.headers}
                  body={requestPreview.body}
                  emptyText={t('empty.requestPreview')}
                />
                <TrafficBlock
                  testId="external-api-response-block"
                  title={t('sections.responseReceived')}
                  headers={{}}
                  body={undefined}
                  emptyText={t('empty.responsePreview')}
                />
              </div>
            </div>
          ) : (
            <div className="text-muted-foreground flex min-h-[420px] flex-1 items-center justify-center rounded-md border bg-muted/10 text-sm italic">
              {t('empty.testPending')}
            </div>
          )
        ) : (
          <div className="flex min-h-0 flex-1 overflow-hidden rounded-md border bg-muted/10">
            <div
              data-testid="external-api-traffic-grid"
              className="grid h-full flex-1 gap-px bg-border xl:grid-cols-2"
            >
              <TrafficBlock
                testId="external-api-request-block"
                title={t('sections.requestSent')}
                header={
                  requestPreview.method && requestPreview.url
                    ? `${requestPreview.method} ${requestPreview.url}`
                    : undefined
                }
                headers={requestPreview.headers}
                body={requestPreview.body}
                emptyText={t('empty.requestPreview')}
              />
              <TrafficBlock
                testId="external-api-response-block"
                title={t('sections.responseReceived')}
                header={
                  typeof httpStatus === 'number'
                    ? `${t('fields.httpStatus')} ${httpStatus}`
                    : undefined
                }
                headers={responseHeaders}
                body={responseText}
                emptyText={t('empty.responsePreview')}
              />
            </div>
          </div>
        )}

        <EvaluationDebugDialog
          open={evaluationModalOpen}
          onOpenChange={setEvaluationModalOpen}
          evaluations={evaluations}
        />
      </div>
    </div>
  );
}

interface TrafficBlockProps {
  testId?: string;
  title: string;
  header?: string;
  headers: Record<string, string>;
  body: unknown;
  emptyText: string;
  variant?:
    | 'default'
    | 'modal-dark-request'
    | 'modal-dark-response-ok'
    | 'modal-dark-response-error';
  statusBadge?: string;
}

function TrafficBlock({
  testId,
  title,
  header,
  headers,
  body,
  emptyText,
  variant = 'default',
  statusBadge,
}: TrafficBlockProps) {
  const hasHeaders = Object.keys(headers).length > 0;
  const bodyText = formatPreview(body);

  if (variant !== 'default') {
    const isResponse = variant !== 'modal-dark-request';
    const bodyClass =
      variant === 'modal-dark-response-error'
        ? 'text-danger-soft-foreground'
        : 'text-success-soft-foreground';

    return (
      <div
        className={cn(
          'flex min-h-0 flex-1 flex-col overflow-y-auto',
          variant === 'modal-dark-request' && 'border-b-4 border-border-strong',
        )}
      >
        <div className="fe-traffic-header">
          <span className="fe-traffic-title">{title}</span>
          {isResponse && statusBadge ? (
            <Badge
              variant={variant === 'modal-dark-response-error' ? 'danger-soft' : 'success-soft'}
              className="px-2 py-0.5 text-[10px] font-bold"
            >
              {statusBadge}
            </Badge>
          ) : null}
        </div>

        <pre
          className={cn(
            'flex-1 whitespace-pre-wrap p-4 font-mono text-xs',
            variant === 'modal-dark-request' ? 'text-content-body' : bodyClass,
          )}
        >
          {header ? (
            <div className="mb-2">
              {variant === 'modal-dark-request' ? (
                <>
                  <span className="text-syntax-template-valid font-bold">
                    {header.split(' ')[0]}
                  </span>{' '}
                  {header.substring(header.indexOf(' ') + 1)}
                </>
              ) : (
                header
              )}
            </div>
          ) : null}
          {hasHeaders ? (
            <div className="text-content-muted mb-2">
              {Object.entries(headers).map(([key, value]) => (
                <div key={key}>
                  {key}: {value}
                </div>
              ))}
            </div>
          ) : null}
          {bodyText ? (
            <div
              className={cn(
                variant === 'modal-dark-request' && hasHeaders
                  ? 'text-success/70 mt-3 border-t border-border-strong pt-3'
                  : undefined,
              )}
            >
              {bodyText}
            </div>
          ) : (
            <div className="text-content-subtle italic">{emptyText}</div>
          )}
        </pre>
      </div>
    );
  }

  return (
    <div data-testid={testId} className="flex h-full min-h-[420px] flex-col bg-background">
      <div className="shrink-0 border-b px-4 py-2">
        <p className="text-muted-foreground text-[11px] font-semibold uppercase tracking-wide">
          {title}
        </p>
      </div>
      <div className="flex min-h-0 flex-1 flex-col gap-3 p-4">
        {header ? <p className="font-mono text-xs font-medium">{header}</p> : null}
        {hasHeaders ? (
          <div className="space-y-1 rounded-md border bg-muted/20 p-3 font-mono text-xs">
            {Object.entries(headers).map(([key, value]) => (
              <div key={key}>
                <span className="text-muted-foreground">{key}:</span> {value}
              </div>
            ))}
          </div>
        ) : null}
        <pre className="flex-1 overflow-auto rounded-md border bg-muted/20 p-3 text-xs whitespace-pre-wrap break-words">
          {bodyText || emptyText}
        </pre>
      </div>
    </div>
  );
}

interface RequestPreview {
  method?: string;
  url?: string;
  headers: Record<string, string>;
  body: unknown;
}

function getRequestPreview(
  details: ExternalApiTestDetails | undefined,
  previewRequest?: RenderedDraftRequest,
): RequestPreview {
  const request = details?.request;
  if (!request && previewRequest) {
    return {
      method: previewRequest.method,
      url: previewRequest.url,
      headers: previewRequest.headers,
      body: previewRequest.body,
    };
  }

  return {
    method: request?.method,
    url: request?.url,
    headers: request?.headers ?? {},
    body: request?.body,
  };
}

function getResponseText(
  details: ExternalApiTestDetails | undefined,
  responseBody: unknown,
): string | undefined {
  const responseText = details?.responseText;
  if (responseText != null) {
    return responseText;
  }
  return responseBody === undefined ? undefined : formatPreview(responseBody);
}

interface EvaluationDebugDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  evaluations?: ExternalApiTestEvaluations;
}

function EvaluationDebugDialog({ open, onOpenChange, evaluations }: EvaluationDebugDialogProps) {
  const { t } = useTranslation('external-apis');

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        data-testid="external-api-evaluation-modal"
        className="max-h-[85vh] max-w-2xl overflow-y-auto"
      >
        <DialogHeader>
          <DialogTitle>{t('evaluationDebug.title')}</DialogTitle>
          <DialogDescription>{t('evaluationDebug.description')}</DialogDescription>
        </DialogHeader>

        {evaluations ? (
          <div className="space-y-4">
            <div className="rounded-lg border border-border bg-muted/20 p-4">
              <div className="mb-3 flex flex-wrap items-center gap-2">
                <Badge variant={evaluations.final.passed ? 'success-soft' : 'danger-soft'}>
                  {evaluations.final.passed ? t('badges.passed') : t('badges.failed')}
                </Badge>
                <Badge variant="neutral-soft">{t(`responseModes.${evaluations.final.mode}`)}</Badge>
              </div>

              <div className="grid gap-3 sm:grid-cols-2">
                <EvaluationRow label={t('evaluationDebug.fields.finalResult')}>
                  <EvaluationResultBadge passed={evaluations.final.passed} />
                </EvaluationRow>
                <EvaluationRow label={t('evaluationDebug.fields.mode')}>
                  <span>{t(`responseModes.${evaluations.final.mode}`)}</span>
                </EvaluationRow>
              </div>
            </div>

            <div className="rounded-lg border border-border bg-background p-4">
              <h4 className="text-sm font-semibold">{t('evaluationDebug.httpTitle')}</h4>
              <div className="mt-3 space-y-3">
                <EvaluationRow label={t('evaluationDebug.fields.applied')}>
                  <span>
                    {evaluations.http.applied
                      ? t('evaluationDebug.values.yes')
                      : t('evaluationDebug.notApplicable')}
                  </span>
                </EvaluationRow>

                {evaluations.http.applied ? (
                  <>
                    <EvaluationRow label={t('evaluationDebug.fields.mode')}>
                      <span>{getHTTPModeLabel(evaluations.http.mode, t)}</span>
                    </EvaluationRow>
                    {evaluations.http.mode === 'status_codes' ? (
                      <EvaluationRow label={t('evaluationDebug.fields.expectedCodes')}>
                        <span>{evaluations.http.expectedCodes.join(', ')}</span>
                      </EvaluationRow>
                    ) : null}
                    <EvaluationRow label={t('evaluationDebug.fields.actualStatus')}>
                      <span>{evaluations.http.actualStatus}</span>
                    </EvaluationRow>
                    <EvaluationRow label={t('evaluationDebug.fields.passed')}>
                      <EvaluationResultBadge passed={evaluations.http.passed} />
                    </EvaluationRow>
                  </>
                ) : null}
              </div>
            </div>

            <div className="rounded-lg border border-border bg-background p-4">
              <h4 className="text-sm font-semibold">{t('evaluationDebug.expressionTitle')}</h4>
              <div className="mt-3 space-y-3">
                <EvaluationRow label={t('evaluationDebug.fields.applied')}>
                  <span>
                    {evaluations.expression.applied
                      ? t('evaluationDebug.values.yes')
                      : t('evaluationDebug.notApplicable')}
                  </span>
                </EvaluationRow>

                {evaluations.expression.applied ? (
                  <>
                    <EvaluationRow label={t('evaluationDebug.fields.expression')} stacked>
                      <pre className="overflow-auto rounded-md border bg-muted/20 p-3 font-mono text-xs whitespace-pre-wrap break-words">
                        {evaluations.expression.expression}
                      </pre>
                    </EvaluationRow>
                    <EvaluationRow label={t('evaluationDebug.fields.resolution')} stacked>
                      <pre className="overflow-auto rounded-md border bg-muted/20 p-3 font-mono text-xs whitespace-pre-wrap break-words">
                        {evaluations.expression.resolvedExpression ??
                          evaluations.expression.expression}
                      </pre>
                    </EvaluationRow>
                    <EvaluationRow label={t('evaluationDebug.fields.passed')}>
                      <EvaluationResultBadge passed={evaluations.expression.passed} />
                    </EvaluationRow>
                    {evaluations.expression.error ? (
                      <EvaluationRow label={t('evaluationDebug.fields.error')} stacked>
                        <div className="rounded-md border border-danger-soft bg-danger-soft/40 p-3 text-sm text-danger-soft-foreground">
                          {evaluations.expression.error}
                        </div>
                      </EvaluationRow>
                    ) : null}
                  </>
                ) : null}
              </div>
            </div>
          </div>
        ) : (
          <p className="text-muted-foreground text-sm">{t('evaluationDebug.pendingTooltip')}</p>
        )}
      </DialogContent>
    </Dialog>
  );
}

function EvaluationRow({
  label,
  children,
  stacked = false,
}: {
  label: string;
  children: ReactNode;
  stacked?: boolean;
}) {
  return (
    <div className={cn('gap-2', stacked ? 'grid' : 'grid grid-cols-[140px_1fr] items-start')}>
      <span className="text-muted-foreground text-xs font-semibold uppercase tracking-wide">
        {label}
      </span>
      <div className="text-sm">{children}</div>
    </div>
  );
}

function EvaluationResultBadge({ passed }: { passed: boolean }) {
  const { t } = useTranslation('external-apis');

  return (
    <Badge variant={passed ? 'success-soft' : 'danger-soft'}>
      {passed ? t('evaluationDebug.values.true') : t('evaluationDebug.values.false')}
    </Badge>
  );
}

function getHTTPModeLabel(mode: ExternalApiHTTPValidationMode, t: (key: string) => string): string {
  return mode === 'any_2xx' ? t('httpModes.2xx') : t('httpModes.custom');
}

function formatPreview(value: unknown): string {
  return formatExternalApiPreview(value);
}

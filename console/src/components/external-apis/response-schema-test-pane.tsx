import { Play } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import {
  formatExternalApiPreview,
  getExternalApiTestResponseBody,
  isExternalApiSchemaUsableBody,
} from './external-api-test-utils';

import type {
  ExternalApiInputDescriptor,
  RenderedDraftRequest,
} from './external-api-builder-utils';
import type { ExternalApiTestResponse } from '@/api/types';
import type { Dispatch, SetStateAction } from 'react';

interface ResponseSchemaTestPaneProps {
  inputs: ExternalApiInputDescriptor[];
  paramInputValues: Record<string, string>;
  onParamInputValuesChange: Dispatch<SetStateAction<Record<string, string>>>;
  testResult: ExternalApiTestResponse | null;
  isTesting: boolean;
  onRunTest: () => void;
  onUseResponseAsSchema: () => void;
  previewRequest?: RenderedDraftRequest;
}

export function ResponseSchemaTestPane({
  inputs,
  paramInputValues,
  onParamInputValuesChange,
  testResult,
  isTesting,
  onRunTest,
  onUseResponseAsSchema,
  previewRequest,
}: ResponseSchemaTestPaneProps) {
  const { t } = useTranslation('external-apis');
  const detailsRecord = testResult?.details as Record<string, unknown> | undefined;
  const requestPreview = getRequestPreview(detailsRecord, previewRequest);
  const responseBody = getExternalApiTestResponseBody(testResult);
  const responseText = getResponseText(detailsRecord, responseBody);
  const responseHeaders = getStringRecord(getObjectField(detailsRecord, 'responseHeaders'));
  const httpStatus = testResult?.httpStatus;
  const canUseResponseAsSchema =
    typeof httpStatus === 'number' &&
    httpStatus < 300 &&
    isExternalApiSchemaUsableBody(responseBody);

  return (
    <div className="grid h-full min-h-0 gap-6 xl:grid-cols-12">
      <div className="flex min-h-0 flex-col xl:col-span-4 xl:border-r xl:border-border-strong xl:pr-6">
        <h3 className="text-content-body mb-4 border-b border-border pb-2 text-sm font-medium">
          {t('sections.testInputs')}
        </h3>

        <div className="flex-1 overflow-y-auto pr-2">
          {inputs.length === 0 ? (
            <p className="text-content-subtle text-sm italic">{t('empty.params')}</p>
          ) : (
            <div className="space-y-4">
              {inputs.map((param) => (
                <div key={param.name}>
                  <label
                    htmlFor={`response-schema-test-input-${param.name}`}
                    className="text-content-muted mb-1 block text-xs"
                  >
                    {param.name} <span className="text-content-subtle">({param.type})</span>
                    {param.required ? <span className="text-destructive ml-1">*</span> : null}
                  </label>

                  {param.type === 'bool' ? (
                    <select
                      id={`response-schema-test-input-${param.name}`}
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
                      id={`response-schema-test-input-${param.name}`}
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
          <button
            type="button"
            onClick={onRunTest}
            disabled={isTesting}
            className="flex w-full items-center justify-center gap-2 rounded-lg bg-primary py-2 font-medium text-primary-foreground transition-colors hover:bg-primary/92 disabled:opacity-50"
          >
            <Play className="h-4 w-4" />
            {isTesting ? t('actions.testing') : t('actions.test')}
          </button>
        </div>
      </div>

      <div className="flex min-h-0 flex-col xl:col-span-8">
        <div className="mb-2 flex items-center justify-between">
          <span className="text-content-body text-sm font-medium">
            {t('sections.trafficInspection')}
          </span>
          {canUseResponseAsSchema ? (
            <button
              type="button"
              onClick={onUseResponseAsSchema}
              className="border-success-soft bg-success-soft text-success-soft-foreground hover:bg-success-soft/80 flex items-center gap-1 rounded border px-3 py-1.5 text-xs transition-colors"
            >
              {t('actions.useResponseAsSchema')}
            </button>
          ) : null}
        </div>

        <div className="fe-traffic-shell flex min-h-0 flex-1 flex-col overflow-hidden">
          {!testResult &&
          !requestPreview.method &&
          !requestPreview.url &&
          Object.keys(requestPreview.headers).length === 0 &&
          requestPreview.body == null ? (
            <div className="text-content-subtle flex flex-1 items-center justify-center text-sm italic">
              {t('empty.testPending')}
            </div>
          ) : (
            <div className="flex min-h-0 flex-1 flex-col">
              <TrafficSection
                title={t('sections.requestSent')}
                method={requestPreview.method}
                url={requestPreview.url}
                headers={requestPreview.headers}
                body={requestPreview.body}
                emptyText={t('empty.requestPreview')}
                request
              />
              <TrafficSection
                title={t('sections.responseReceived')}
                headers={responseHeaders}
                body={responseText}
                emptyText={t('empty.responsePreview')}
                statusBadge={typeof httpStatus === 'number' ? `STATUS ${httpStatus}` : undefined}
                statusOk={typeof httpStatus === 'number' ? httpStatus < 300 : false}
              />
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

interface TrafficSectionProps {
  title: string;
  headers: Record<string, string>;
  body: unknown;
  emptyText: string;
  method?: string;
  url?: string;
  request?: boolean;
  statusBadge?: string;
  statusOk?: boolean;
}

function TrafficSection({
  title,
  headers,
  body,
  emptyText,
  method,
  url,
  request = false,
  statusBadge,
  statusOk = false,
}: TrafficSectionProps) {
  const bodyText = formatPreview(body);
  const hasHeaders = Object.keys(headers).length > 0;

  return (
    <div
      className={`flex min-h-0 flex-1 flex-col overflow-y-auto ${
        request ? 'border-b-4 border-border-strong' : ''
      }`}
    >
      <div className="fe-traffic-header">
        <span className="fe-traffic-title">{title}</span>
        {!request && statusBadge ? (
          <span
            className={`rounded px-2 py-0.5 text-[10px] font-bold ${
              statusOk
                ? 'border-success-soft bg-success-soft text-success-soft-foreground'
                : 'border-danger-soft bg-danger-soft text-danger-soft-foreground'
            }`}
          >
            {statusBadge}
          </span>
        ) : null}
      </div>

      <div
        className={`flex-1 whitespace-pre-wrap p-4 font-mono text-xs ${
          request
            ? 'text-content-body'
            : statusOk
              ? 'text-success-soft-foreground'
              : 'text-danger-soft-foreground'
        }`}
      >
        {request && method && url ? (
          <div className="mb-2">
            <span className="text-syntax-template-valid font-bold">{method}</span> {url}
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
            className={
              request && hasHeaders
                ? 'text-success/70 mt-3 border-t border-border-strong pt-3'
                : undefined
            }
          >
            {bodyText}
          </div>
        ) : (
          <div className="text-content-subtle italic">{emptyText}</div>
        )}
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
  details: Record<string, unknown> | undefined,
  previewRequest?: RenderedDraftRequest,
): RequestPreview {
  const request = getObjectField(details, 'request');
  if (!request && previewRequest) {
    return {
      method: previewRequest.method,
      url: previewRequest.url,
      headers: previewRequest.headers,
      body: previewRequest.body,
    };
  }
  return {
    method: getStringField(request, 'method'),
    url: getStringField(request, 'url'),
    headers: getStringRecord(getObjectField(request, 'headers')),
    body: request ? request.body : undefined,
  };
}

function getResponseText(
  details: Record<string, unknown> | undefined,
  responseBody: unknown,
): string | undefined {
  const responseText = getStringField(details, 'responseText');
  if (responseText != null) {
    return responseText;
  }
  return responseBody === undefined ? undefined : formatPreview(responseBody);
}

function getStringField(
  value: Record<string, unknown> | undefined,
  key: string,
): string | undefined {
  const candidate = value?.[key];
  return typeof candidate === 'string' ? candidate : undefined;
}

function getObjectField(
  value: Record<string, unknown> | undefined,
  key: string,
): Record<string, unknown> | undefined {
  const candidate = value?.[key];
  if (!candidate || typeof candidate !== 'object' || Array.isArray(candidate)) {
    return undefined;
  }
  return candidate as Record<string, unknown>;
}

function getStringRecord(value: Record<string, unknown> | undefined): Record<string, string> {
  if (!value) {
    return {};
  }

  return Object.fromEntries(
    Object.entries(value)
      .filter(([, candidate]) => candidate != null)
      .map(([key, candidate]) => [key, String(candidate)]),
  );
}

function formatPreview(value: unknown): string {
  return formatExternalApiPreview(value);
}

import {
  AlertCircle,
  AlignLeft,
  FileJson,
  GitCommit,
  ListTree,
  Plus,
  Terminal,
  X,
} from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import {
  buildBodyEditorState,
  createVisualNodeDraft,
  inferJsonSchema,
  jsonSchemaToExample,
  visualNodesToJson,
  type BodyRootType,
  type ExternalApiInputDescriptor,
  type RenderedDraftRequest,
  type VisualNode,
} from './external-api-builder-utils';
import {
  getExternalApiTestResponseBody,
  isExternalApiSchemaUsableBody,
} from './external-api-test-utils';
import { ResponseSchemaTestPane } from './response-schema-test-pane';
import { ResponseSchemaVisualTree } from './response-schema-visual-tree';

import type { ExternalApiTestResponse } from '@/api/types';
import type { Dispatch, SetStateAction } from 'react';

import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/components/ui/dialog';

type ResponseSchemaModalTab = 'sample' | 'visual' | 'test';

interface ResponseSchemaModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  schema?: Record<string, unknown>;
  sampleResponseText?: string;
  inputs: ExternalApiInputDescriptor[];
  paramInputValues: Record<string, string>;
  onParamInputValuesChange: Dispatch<SetStateAction<Record<string, string>>>;
  testResult: ExternalApiTestResponse | null;
  isTesting: boolean;
  onRunTest: () => void;
  previewRequest?: RenderedDraftRequest;
  onApply: (next: { schema: Record<string, unknown>; sampleResponseText: string }) => void;
}

export function ResponseSchemaModal({
  open,
  onOpenChange,
  schema,
  sampleResponseText,
  inputs,
  paramInputValues,
  onParamInputValuesChange,
  testResult,
  isTesting,
  onRunTest,
  previewRequest,
  onApply,
}: ResponseSchemaModalProps) {
  const editorKey = JSON.stringify(schema ?? {}) + '|' + (sampleResponseText ?? '');

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-5xl gap-0 overflow-hidden border-0 bg-transparent p-0 shadow-none [&>button.absolute]:hidden">
        {open ? (
          <ResponseSchemaModalContent
            key={editorKey}
            onOpenChange={onOpenChange}
            schema={schema}
            sampleResponseText={sampleResponseText}
            inputs={inputs}
            paramInputValues={paramInputValues}
            onParamInputValuesChange={onParamInputValuesChange}
            testResult={testResult}
            isTesting={isTesting}
            onRunTest={onRunTest}
            previewRequest={previewRequest}
            onApply={onApply}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function ResponseSchemaModalContent({
  onOpenChange,
  schema,
  sampleResponseText,
  inputs,
  paramInputValues,
  onParamInputValuesChange,
  testResult,
  isTesting,
  onRunTest,
  previewRequest,
  onApply,
}: Omit<ResponseSchemaModalProps, 'open'>) {
  const { t } = useTranslation('external-apis');
  const initialSampleRaw = buildInitialSampleRaw(schema, sampleResponseText);
  const initialVisualState = buildVisualState(initialSampleRaw);
  const [activeTab, setActiveTab] = useState<ResponseSchemaModalTab>('sample');
  const [sampleRaw, setSampleRaw] = useState(initialSampleRaw);
  const [visualNodes, setVisualNodes] = useState<VisualNode[]>(initialVisualState.nodes);
  const [rootType, setRootType] = useState<BodyRootType>(initialVisualState.rootType);
  const [jsonError, setJsonError] = useState<string | null>(null);
  const responseBody = getExternalApiTestResponseBody(testResult);

  const syncVisualFromRaw = (nextRaw: string, notifyOnError: boolean) => {
    const parsed = parseJson(nextRaw);
    if (parsed == null || typeof parsed !== 'object') {
      const message = t('errors.schemaExampleObject');
      setJsonError(message);
      if (notifyOnError) {
        toast.error(message);
      }
      return false;
    }

    const nextState = buildBodyEditorState(parsed);
    setVisualNodes(nextState.bodyVisual);
    setRootType(nextState.rootType);
    setJsonError(null);
    return true;
  };

  const syncRawFromVisual = () => {
    const nextValue = visualNodesToJson(visualNodes, rootType);
    setSampleRaw(JSON.stringify(nextValue, null, 2));
    setJsonError(null);
  };

  const handleTabChange = (nextTab: ResponseSchemaModalTab) => {
    setJsonError(null);

    if (nextTab === activeTab) {
      return;
    }

    if (nextTab === 'visual' && !syncVisualFromRaw(sampleRaw, true)) {
      return;
    }

    if (activeTab === 'visual' && nextTab !== 'visual') {
      syncRawFromVisual();
    }

    setActiveTab(nextTab);
  };

  const handleFormatSample = () => {
    const parsed = parseJson(sampleRaw);
    if (parsed === undefined) {
      const message = t('errors.invalidJson');
      setJsonError(message);
      toast.error(message);
      return;
    }

    setSampleRaw(JSON.stringify(parsed, null, 2));
    setJsonError(null);
  };

  const applyFromSample = () => {
    const parsed = parseJson(sampleRaw);
    if (parsed === undefined) {
      const message = t('errors.invalidJson');
      setJsonError(message);
      toast.error(message);
      return;
    }

    onApply({
      schema: inferJsonSchema(parsed),
      sampleResponseText: JSON.stringify(parsed, null, 2),
    });
    onOpenChange(false);
  };

  const applyFromVisual = () => {
    const nextValue = visualNodesToJson(visualNodes, rootType);
    onApply({
      schema: inferJsonSchema(nextValue),
      sampleResponseText: JSON.stringify(nextValue, null, 2),
    });
    onOpenChange(false);
  };

  const applyFromTest = () => {
    if (responseBody === undefined) {
      return;
    }

    onApply({
      schema: inferJsonSchema(responseBody),
      sampleResponseText: JSON.stringify(responseBody, null, 2),
    });
    onOpenChange(false);
  };

  const loadTestResponseIntoDraft = () => {
    if (!isExternalApiSchemaUsableBody(responseBody)) {
      const message = t('errors.testResponseSchemaObject');
      setJsonError(message);
      toast.error(message);
      return;
    }

    const nextRaw = JSON.stringify(responseBody, null, 2);
    setSampleRaw(nextRaw);
    const nextState = buildBodyEditorState(responseBody);
    setVisualNodes(nextState.bodyVisual);
    setRootType(nextState.rootType);
    setJsonError(null);
    setActiveTab('sample');
  };

  const saveDisabled =
    activeTab === 'test'
      ? !isExternalApiSchemaUsableBody(responseBody)
      : activeTab === 'sample'
        ? sampleRaw.trim().length === 0
        : false;

  return (
    <div className="fe-panel-elevated flex max-h-[90vh] flex-col overflow-hidden">
      <div className="flex items-center justify-between border-b border-border-strong bg-surface-panel/80 px-6 py-4">
        <div className="min-w-0">
          <DialogTitle className="text-content-strong flex items-center gap-2 text-lg font-bold">
            <GitCommit className="text-accent-soft-foreground h-5 w-5" />
            {t('schemaModal.title')}
          </DialogTitle>
          <DialogDescription className="sr-only">{t('schemaModal.description')}</DialogDescription>
        </div>

        <button
          type="button"
          onClick={() => onOpenChange(false)}
          className="text-content-muted hover:text-content-strong transition-colors"
          aria-label={t('common:actions.cancel')}
        >
          <X className="h-5 w-5" />
        </button>
      </div>

      <div className="flex gap-3 border-b border-border-strong bg-surface-elevated px-6 pt-4">
        {[
          { id: 'sample', icon: FileJson, label: t('schemaModal.tabs.sample') },
          { id: 'visual', icon: ListTree, label: t('schemaModal.tabs.visual') },
          { id: 'test', icon: Terminal, label: t('schemaModal.tabs.test') },
        ].map((tab) => {
          const Icon = tab.icon;
          const active = activeTab === tab.id;

          return (
            <button
              key={tab.id}
              type="button"
              onClick={() => handleTabChange(tab.id as ResponseSchemaModalTab)}
              data-active={active}
              className="fe-tab-trigger pb-3"
            >
              <Icon className="mb-0.5 mr-1 inline-block h-4 w-4" />
              {tab.label}
            </button>
          );
        })}
      </div>

      <div
        data-testid="response-schema-modal-content"
        className="h-[560px] min-h-[560px] flex-1 overflow-y-auto bg-surface-elevated p-6"
      >
        {activeTab === 'sample' ? (
          <div className="flex h-full min-h-0 flex-col animate-in fade-in">
            <label htmlFor="response-schema-sample" className="sr-only">
              {t('fields.sampleResponse')}
            </label>

            <div className="mb-3 flex items-center justify-between gap-3">
              <p className="text-content-muted text-sm">{t('schemaModal.sampleHint')}</p>
              {jsonError ? (
                <span className="text-danger-soft-foreground flex items-center gap-1 text-xs">
                  <AlertCircle className="h-3 w-3" />
                  {jsonError}
                </span>
              ) : null}
            </div>

            <div className="group relative min-h-0 flex-1">
              <textarea
                id="response-schema-sample"
                value={sampleRaw}
                onChange={(event) => {
                  setSampleRaw(event.target.value);
                  setJsonError(null);
                }}
                className="fe-editor h-full min-h-0 w-full resize-none p-4 font-mono text-sm text-success-soft-foreground outline-none focus:border-ring"
                placeholder={t('placeholders.sampleResponse')}
                spellCheck={false}
              />
              <button
                type="button"
                aria-label={t('actions.formatJson')}
                title={t('actions.formatJson')}
                onClick={handleFormatSample}
                className="text-success-soft-foreground absolute bottom-4 right-6 rounded-lg border border-border bg-surface-panel p-2 opacity-0 shadow-lg transition-opacity hover:bg-surface-subtle focus:opacity-100 focus:outline-none focus:ring-2 focus:ring-ring group-hover:opacity-100"
              >
                <AlignLeft className="h-5 w-5" />
              </button>
            </div>
          </div>
        ) : null}

        {activeTab === 'visual' ? (
          <div className="flex h-full min-h-0 flex-col animate-in fade-in">
            <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
              <div className="flex items-center gap-2">
                <span className="text-content-muted text-sm font-medium">
                  {t('schemaModal.rootType')}
                </span>
                <select
                  id="response-schema-root-type"
                  value={rootType}
                  onChange={(event) => setRootType(event.target.value as BodyRootType)}
                  className="fe-editor px-2 py-1 pr-8 text-xs focus:outline-none"
                >
                  <option value="object">{t('schemaModal.rootTypes.object')}</option>
                  <option value="array">{t('schemaModal.rootTypes.array')}</option>
                </select>
              </div>

              <button
                type="button"
                onClick={() => setVisualNodes((current) => [...current, createVisualNodeDraft()])}
                className="bg-accent-soft text-accent-soft-foreground hover:bg-accent-soft/80 flex items-center gap-1 rounded border border-accent-soft px-3 py-1.5 text-xs transition-colors"
              >
                <Plus className="h-3 w-3" />
                {rootType === 'array'
                  ? t('schemaModal.addRootItem')
                  : t('schemaModal.addRootField')}
              </button>
            </div>

            <div className="fe-editor min-h-0 flex-1 overflow-y-auto p-4">
              {visualNodes.length === 0 ? (
                <p className="text-content-subtle mt-4 text-center text-sm italic">
                  {t('schemaModal.emptyVisual')}
                </p>
              ) : (
                <ResponseSchemaVisualTree
                  nodes={visualNodes}
                  rootType={rootType}
                  onChange={setVisualNodes}
                  addFieldLabel={t('actions.addField')}
                  addItemLabel={t('actions.addItem')}
                  stringValuePlaceholder={t('schemaModal.visualStringPlaceholder')}
                  numberValuePlaceholder="123"
                />
              )}
            </div>
          </div>
        ) : null}

        {activeTab === 'test' ? (
          <div className="h-full min-h-0 animate-in fade-in">
            <ResponseSchemaTestPane
              inputs={inputs}
              paramInputValues={paramInputValues}
              onParamInputValuesChange={onParamInputValuesChange}
              testResult={testResult}
              isTesting={isTesting}
              onRunTest={onRunTest}
              previewRequest={previewRequest}
              onUseResponseAsSchema={loadTestResponseIntoDraft}
            />
          </div>
        ) : null}
      </div>

      <div className="flex justify-end border-t border-border-strong bg-surface-panel/40 px-6 py-4">
        <button
          type="button"
          onClick={() => {
            if (activeTab === 'sample') {
              applyFromSample();
              return;
            }
            if (activeTab === 'visual') {
              applyFromVisual();
              return;
            }
            applyFromTest();
          }}
          disabled={saveDisabled}
          className="rounded-lg bg-primary px-5 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/92 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {t('actions.confirmSchema')}
        </button>
      </div>
    </div>
  );
}

function buildInitialSampleRaw(
  schema: Record<string, unknown> | undefined,
  sampleResponseText: string | undefined,
) {
  const trimmedSample = sampleResponseText?.trim();
  if (trimmedSample) {
    const parsed = parseJson(trimmedSample);
    if (parsed !== undefined) {
      return JSON.stringify(parsed, null, 2);
    }
    return trimmedSample;
  }

  if (!schema || Object.keys(schema).length === 0) {
    return JSON.stringify({}, null, 2);
  }

  const example = jsonSchemaToExample(schema);
  return JSON.stringify(example, null, 2);
}

function buildVisualState(rawValue: string) {
  const parsed = parseJson(rawValue);
  if (parsed == null || typeof parsed !== 'object') {
    return {
      nodes: [] as VisualNode[],
      rootType: 'object' as BodyRootType,
    };
  }

  const nextState = buildBodyEditorState(parsed);
  return {
    nodes: nextState.bodyVisual,
    rootType: nextState.rootType,
  };
}

function parseJson(value: string): unknown | undefined {
  try {
    return JSON.parse(value) as unknown;
  } catch {
    return undefined;
  }
}

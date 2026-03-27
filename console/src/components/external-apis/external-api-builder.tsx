import { useNavigate } from '@tanstack/react-router';
import {
  AlertCircle,
  AlignLeft,
  ArrowLeft,
  ArrowRight,
  Check,
  Code,
  DatabaseZap,
  FileJson,
  Globe,
  ListTree,
  Play,
  Plus,
  Save,
  Trash2,
} from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { BodyVisualEditor } from './body-visual-editor';
import {
  areDraftVariablesEqual,
  buildExternalApiPartsFromDraft,
  buildRequestConfigFromDraft,
  collectSecretKeysFromDraft,
  createDraftHeader,
  createExternalApiDraft,
  detectDraftVariables,
  draftVariablesToInputDescriptors,
  jsonToVisualNodes,
  parseParamInputValue,
  renderDraftRequest,
  visualNodesToJson,
  type BodyRootType,
  type DraftVariableConfig,
  type ExternalApiDraft,
} from './external-api-builder-utils';
import { ExternalApiTestPanel } from './external-api-test-panel';
import { getExternalApiTestResponseBody } from './external-api-test-utils';
import {
  HighlightedTemplateInput,
  HighlightedTemplateTextarea,
  renderHighlightedText,
} from './highlighted-template-field';
import { ResponseExpressionEditor } from './response-expression-editor';
import { ResponseSchemaModal } from './response-schema-modal';
import { VariablesManagerModal } from './variables-manager-modal';

import type {
  ExternalApi,
  ExternalApiRequestConfig,
  ExternalApiResponseValidation,
} from '@/api/types';
import type { Dispatch, SetStateAction } from 'react';

import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { PageHeader } from '@/components/shared/page-header';
import { Button } from '@/components/ui/button';
import { Switch } from '@/components/ui/switch';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { useUnsavedChanges } from '@/hooks/use-unsaved-changes';
import { getVisibleErrorMessage } from '@/lib/display-error';
import { slugifyResourceKey } from '@/lib/resource-key';
import { cn } from '@/lib/utils';
import {
  useCreateExternalApi,
  useDeleteExternalApi,
  useTestExternalApi,
  useUpdateExternalApi,
} from '@/mutations/external-api-mutations';

type BuilderStep = 'url' | 'request' | 'validation' | 'test';

interface ExternalApiBuilderProps {
  externalApi?: ExternalApi;
}

export function ExternalApiBuilder({ externalApi }: ExternalApiBuilderProps) {
  const { t } = useTranslation('external-apis');
  const navigate = useNavigate();
  const isEditing = Boolean(externalApi?.key);
  const createExternalApi = useCreateExternalApi();
  const updateExternalApi = useUpdateExternalApi();
  const deleteExternalApi = useDeleteExternalApi();
  const testExternalApi = useTestExternalApi();

  const [draft, setDraft] = useState<ExternalApiDraft>(() => createExternalApiDraft(externalApi));
  const [savedSnapshot, setSavedSnapshot] = useState(() =>
    serializeExternalApiPersistedSnapshot(
      buildExternalApiPersistedSnapshot(createExternalApiDraft(externalApi), isEditing),
    ),
  );
  const [activeStep, setActiveStep] = useState<BuilderStep>('url');
  const [bodyJsonError, setBodyJsonError] = useState<string | null>(null);
  const [variablesModalOpen, setVariablesModalOpen] = useState(false);
  const [schemaModalOpen, setSchemaModalOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [savePulseActive, setSavePulseActive] = useState(false);
  const hasTriggeredSavePulseRef = useRef(false);
  const steps = useMemo(
    () =>
      [
        { id: 'url', icon: Globe, label: t('steps.urlMethod') },
        { id: 'request', icon: DatabaseZap, label: t('steps.request') },
        { id: 'validation', icon: Check, label: t('steps.validation') },
        { id: 'test', icon: Play, label: t('steps.test') },
      ] satisfies { id: BuilderStep; icon: typeof Globe; label: string }[],
    [t],
  );

  const derivedKey = useMemo(() => slugifyResourceKey(draft.name), [draft.name]);
  const secretKeys = useMemo(() => collectSecretKeysFromDraft(draft), [draft]);
  const secretPayload = useMemo(
    () => filterFilledSecretPayload(draft.secretPayload),
    [draft.secretPayload],
  );
  const previewRequest = useMemo(() => renderDraftRequest(draft, draft.testInputs), [draft]);
  const { request, params, expressionVariables, responseValidation } = useMemo(
    () => buildExternalApiPartsFromDraft(draft),
    [draft],
  );
  const inputDescriptors = useMemo(
    () => draftVariablesToInputDescriptors(draft.variables),
    [draft.variables],
  );
  const lastTestResponseBody = getExternalApiTestResponseBody(draft.testResult);
  const hasSchema = hasResponseSchemaSample(draft.responseSchema);
  const schemaPreview = useMemo(
    () => JSON.stringify(responseValidation.body.schema ?? {}, null, 2),
    [responseValidation.body.schema],
  );
  const hasRequestBody = ['POST', 'PUT', 'PATCH'].includes(draft.method);
  const currentStepIndex = steps.findIndex((step) => step.id === activeStep);
  const previousStep = currentStepIndex > 0 ? steps[currentStepIndex - 1] : null;
  const nextStep = currentStepIndex < steps.length - 1 ? steps[currentStepIndex + 1] : null;
  const responseHeaderKeys = useMemo(() => {
    const rawHeaders = draft.testResult?.details?.responseHeaders;
    if (!rawHeaders || typeof rawHeaders !== 'object') {
      return [];
    }

    return Object.keys(rawHeaders as Record<string, unknown>).sort((left, right) =>
      left.localeCompare(right),
    );
  }, [draft.testResult?.details]);
  const urlVariables = useMemo(
    () =>
      Object.entries(draft.variables)
        .filter(([, config]) =>
          [...config.locations].some((location) => location.startsWith('url_')),
        )
        .sort(([left], [right]) => left.localeCompare(right)),
    [draft.variables],
  );
  const persistedSnapshot = useMemo(
    () =>
      serializeExternalApiPersistedSnapshot(
        buildExternalApiPersistedSnapshot(draft, isEditing, {
          request,
          params,
          expressionVariables,
          responseValidation,
        }),
      ),
    [draft, expressionVariables, isEditing, params, request, responseValidation],
  );
  const isDirty = persistedSnapshot !== savedSnapshot;
  const isSaving = createExternalApi.isPending || updateExternalApi.isPending;
  const saveEnabled = isDirty && !isSaving && bodyJsonError == null;
  const { handleBack, UnsavedDialog, markClean } = useUnsavedChanges({
    isDirty,
    backTo: '/settings/external-apis',
    blockNavigation: true,
  });

  useEffect(() => {
    if (!saveEnabled || hasTriggeredSavePulseRef.current) {
      return;
    }

    hasTriggeredSavePulseRef.current = true;
    setSavePulseActive(true);

    const timeoutId = window.setTimeout(() => {
      setSavePulseActive(false);
    }, 1600);

    return () => window.clearTimeout(timeoutId);
  }, [saveEnabled]);

  const syncSavedState = (saved: ExternalApi) => {
    const nextDraft = createExternalApiDraft(saved);
    setDraft(nextDraft);
    setSavedSnapshot(
      serializeExternalApiPersistedSnapshot(buildExternalApiPersistedSnapshot(nextDraft, true)),
    );
    setBodyJsonError(null);
  };

  const syncDetectedVariables = (nextDraft: ExternalApiDraft): ExternalApiDraft => {
    const nextVariables = detectDraftVariables(nextDraft);
    if (areDraftVariablesEqual(nextDraft.variables, nextVariables)) {
      return nextDraft;
    }
    return {
      ...nextDraft,
      variables: nextVariables,
    };
  };

  const updateDraft = <Key extends keyof ExternalApiDraft>(
    key: Key,
    value: ExternalApiDraft[Key],
  ) => {
    setDraft((current) => {
      const nextDraft = { ...current, [key]: value };
      return key === 'url' ||
        key === 'headers' ||
        key === 'bodyMode' ||
        key === 'bodyRaw' ||
        key === 'bodyVisual' ||
        key === 'method'
        ? syncDetectedVariables(nextDraft)
        : nextDraft;
    });
  };

  const updateVariableConfig = (
    name: string,
    field: keyof Pick<DraftVariableConfig, 'type' | 'required'>,
    value: DraftVariableConfig[typeof field],
  ) => {
    setDraft((current) => {
      const currentVariable = current.variables[name];
      if (!currentVariable) {
        return current;
      }

      return {
        ...current,
        variables: {
          ...current.variables,
          [name]: {
            ...currentVariable,
            [field]: value,
          },
        },
      };
    });
  };

  const addManualVariable = () => {
    setDraft((current) => {
      const name = buildNextManualVariableName(current.variables);
      return {
        ...current,
        variables: {
          ...current.variables,
          [name]: {
            origin: 'manual',
            type: 'any',
            required: false,
            locations: new Set(),
          },
        },
      };
    });
  };

  const renameManualVariable = (currentName: string, nextName: string) => {
    setDraft((current) => {
      const variable = current.variables[currentName];
      if (!variable || variable.origin !== 'manual' || currentName === nextName) {
        return current;
      }

      const { [currentName]: _removedVariable, ...nextVariables } = current.variables;
      nextVariables[nextName] = variable;

      const currentInputValue = current.testInputs[currentName];
      const { [currentName]: _removedInput, ...nextInputs } = current.testInputs;
      if (currentInputValue != null) {
        nextInputs[nextName] = currentInputValue;
      }

      return {
        ...current,
        variables: nextVariables,
        testInputs: nextInputs,
      };
    });
  };

  const removeManualVariable = (name: string) => {
    setDraft((current) => {
      const variable = current.variables[name];
      if (!variable || variable.origin !== 'manual') {
        return current;
      }

      const { [name]: _removedVariable, ...nextVariables } = current.variables;
      const { [name]: _removedInput, ...nextInputs } = current.testInputs;

      return {
        ...current,
        variables: nextVariables,
        testInputs: nextInputs,
      };
    });
  };

  const setTestInputs: Dispatch<SetStateAction<Record<string, string>>> = (nextValue) => {
    setDraft((current) => ({
      ...current,
      testInputs: typeof nextValue === 'function' ? nextValue(current.testInputs) : nextValue,
    }));
  };

  const handleFormatJson = (field: 'bodyRaw') => {
    try {
      const formatted = JSON.stringify(JSON.parse(draft[field]), null, 2);
      updateDraft(field, formatted);
      setBodyJsonError(null);
    } catch {
      setBodyJsonError(t('errors.invalidJson'));
    }
  };

  const handleBodyModeSwitch = (nextMode: ExternalApiDraft['bodyMode']) => {
    setBodyJsonError(null);

    if (nextMode === 'visual' && draft.bodyMode === 'json') {
      try {
        const parsed = JSON.parse(draft.bodyRaw || '{}') as unknown;
        if (parsed == null || typeof parsed !== 'object') {
          setBodyJsonError(t('errors.bodyObject'));
          return;
        }
        updateDraft('bodyRootType', Array.isArray(parsed) ? 'array' : 'object');
        updateDraft('bodyVisual', buildVisualNodes(parsed));
        updateDraft('bodyMode', nextMode);
        return;
      } catch {
        setBodyJsonError(t('errors.invalidJson'));
        return;
      }
    }

    if (nextMode === 'json' && draft.bodyMode === 'visual') {
      updateDraft(
        'bodyRaw',
        JSON.stringify(visualNodesToJson(draft.bodyVisual, draft.bodyRootType), null, 2),
      );
      updateDraft('bodyMode', nextMode);
      return;
    }

    updateDraft('bodyMode', nextMode);
  };

  const validateBeforeSubmit = ({
    requireResponseValidation = true,
  }: {
    requireResponseValidation?: boolean;
  } = {}): string | null => {
    if (!draft.name.trim()) return t('errors.nameRequired');
    if (!draft.url.trim()) return t('errors.urlRequired');

    if (['POST', 'PUT', 'PATCH'].includes(draft.method) && draft.bodyMode === 'json') {
      try {
        const parsed = JSON.parse(draft.bodyRaw || '{}') as unknown;
        if (parsed == null || typeof parsed !== 'object') {
          return t('errors.bodyObject');
        }
      } catch {
        return t('errors.invalidJson');
      }
    }

    if (
      requireResponseValidation &&
      (draft.validationMode === 'responseBody' || draft.validationMode === 'both')
    ) {
      if (!draft.expression.trim()) {
        return t('errors.expressionRequired');
      }

      try {
        const parsed = JSON.parse(draft.responseSchema || '{}') as unknown;
        if (parsed == null || typeof parsed !== 'object') {
          return t('errors.schemaExampleObject');
        }
      } catch {
        return t('errors.invalidJson');
      }
    }

    if (draft.httpCodeMode === 'custom') {
      const codes = draft.httpCodeCustom
        .split(',')
        .map((value) => Number(value.trim()))
        .filter((value) => !Number.isNaN(value));
      if (codes.length === 0) {
        return t('errors.httpCodesRequired');
      }
    }

    return null;
  };

  const handleSave = () => {
    const validationError = validateBeforeSubmit();
    if (validationError) {
      toast.error(validationError);
      return;
    }

    const payload = buildExternalApiSavePayload({
      draft,
      derivedKey,
      expressionVariables,
      isEditing,
      params,
      request,
      responseValidation,
      secretPayload,
    });

    if (isEditing && externalApi) {
      updateExternalApi.mutate(
        { key: externalApi.key, data: payload },
        {
          onSuccess: (saved) => {
            markClean();
            syncSavedState(saved);
            toast.success(t('messages.updated'));
            void navigate({
              to: '/settings/external-apis/$key',
              params: { key: saved.key },
            });
          },
          onError: (error) => toast.error(getVisibleErrorMessage(error, t('messages.saveError'))),
        },
      );
      return;
    }

    createExternalApi.mutate(payload, {
      onSuccess: (saved) => {
        markClean();
        toast.success(t('messages.created'));
        void navigate({
          to: '/settings/external-apis/$key',
          params: { key: saved.key },
        });
      },
      onError: (error) => toast.error(getVisibleErrorMessage(error, t('messages.saveError'))),
    });
  };

  const handleDelete = () => {
    if (!externalApi) {
      return;
    }

    deleteExternalApi.mutate(externalApi.key, {
      onSuccess: () => {
        markClean();
        toast.success(t('messages.deleted'));
        void navigate({ to: '/settings/external-apis' });
      },
      onError: (error) => toast.error(getVisibleErrorMessage(error, t('messages.deleteError'))),
    });
  };

  const runTest = ({
    requireResponseValidation = true,
    responseValidationOverride,
  }: {
    requireResponseValidation?: boolean;
    responseValidationOverride?: ExternalApiResponseValidation;
  } = {}) => {
    const validationError = validateBeforeSubmit({ requireResponseValidation });
    if (validationError) {
      toast.error(validationError);
      return;
    }

    const paramValues = Object.fromEntries(
      inputDescriptors
        .map(
          (param) =>
            [
              param.name,
              parseParamInputValue(draft.testInputs[param.name] ?? '', param.type),
            ] as const,
        )
        .filter(([, value]) => value !== undefined),
    );

    testExternalApi.mutate(
      {
        currentKey: externalApi?.key,
        key: derivedKey,
        name: draft.name,
        active: draft.active,
        request,
        params,
        expressionVariables,
        responseValidation: responseValidationOverride ?? responseValidation,
        secretPayload,
        replaceSecret: isEditing ? draft.replaceSecret : true,
        paramValues,
      },
      {
        onSuccess: (result) => {
          setDraft((current) => ({
            ...current,
            testResult: result,
          }));
          toast.success(result.ok ? t('messages.testPassed') : t('messages.testBlocked'));
        },
        onError: (error) => {
          setDraft((current) => ({
            ...current,
            testResult: null,
          }));
          toast.error(getVisibleErrorMessage(error, t('messages.testError')));
        },
      },
    );
  };

  const handleTest = () => {
    runTest();
  };

  const handleSchemaTest = () => {
    runTest({
      requireResponseValidation: false,
      responseValidationOverride: {
        mode: 'httpCode',
        http: { mode: 'any_2xx', codes: [] },
        body: {
          expression: '',
          schema: responseValidation.body.schema,
          sampleResponseText: responseValidation.body.sampleResponseText,
        },
      },
    });
  };

  const handleSchemaApply = ({
    sampleResponseText,
  }: {
    schema: Record<string, unknown>;
    sampleResponseText: string;
  }) => {
    try {
      const parsed = JSON.parse(sampleResponseText || '{}') as unknown;
      if (parsed == null || typeof parsed !== 'object') {
        toast.error(t('errors.schemaExampleObject'));
        return;
      }

      setDraft((current) => ({
        ...current,
        responseSchema: JSON.stringify(parsed, null, 2),
        responseSchemaRootType: Array.isArray(parsed) ? 'array' : 'object',
        responseSchemaVisual: buildVisualNodes(parsed),
      }));
    } catch {
      toast.error(t('errors.invalidJson'));
    }
  };

  const handleUseLastResponseAsSchema = () => {
    if (lastTestResponseBody === undefined) {
      return;
    }

    handleSchemaApply({
      schema: {},
      sampleResponseText: JSON.stringify(lastTestResponseBody, null, 2),
    });
    setActiveStep('validation');
  };

  return (
    <div className="flex h-[calc(100dvh-6.5rem)] min-h-0 flex-col gap-6">
      <PageHeader
        title={isEditing ? t('editTitle', { name: draft.name }) : t('createTitle')}
        description={t('subtitle')}
        actions={
          <div className="flex flex-wrap gap-2">
            <Button type="button" variant="outline" onClick={handleBack}>
              <ArrowLeft className="mr-2 h-4 w-4" />
              {t('actions.back')}
            </Button>
            {isEditing ? (
              <Button type="button" variant="outline" onClick={() => setDeleteOpen(true)}>
                <Trash2 className="mr-2 h-4 w-4" />
                {t('actions.delete')}
              </Button>
            ) : null}
            <Button
              type="button"
              onClick={handleSave}
              disabled={!saveEnabled}
              className={cn(
                savePulseActive &&
                  'animate-pulse shadow-lg shadow-primary/30 motion-reduce:animate-none',
              )}
            >
              <Save className="mr-2 h-4 w-4" />
              {t('actions.save')}
            </Button>
          </div>
        }
      />

      <div className="fe-panel-strong flex min-h-0 flex-1 flex-col p-6">
        <div className="mb-6 flex flex-wrap items-center justify-between gap-4">
          <div className="flex flex-wrap items-center gap-3">
            <div className="fe-tablist flex flex-wrap">
              {steps.map((step) => (
                <button
                  key={step.id}
                  type="button"
                  onClick={() => setActiveStep(step.id)}
                  data-active={activeStep === step.id}
                  className="fe-tab-trigger"
                >
                  <step.icon className="h-4 w-4" />
                  {step.label}
                </button>
              ))}
            </div>

            <div className="fe-tablist flex flex-wrap">
              <button
                type="button"
                onClick={() => previousStep && setActiveStep(previousStep.id)}
                disabled={!previousStep}
                className="fe-tab-trigger"
              >
                <ArrowLeft className="h-4 w-4" />
                {t('actions.previous')}
              </button>

              <button
                type="button"
                onClick={() => nextStep && setActiveStep(nextStep.id)}
                disabled={!nextStep}
                className="fe-tab-trigger"
              >
                {t('actions.next')}
                <ArrowRight className="h-4 w-4" />
              </button>
            </div>
          </div>

          <button
            type="button"
            onClick={() => setVariablesModalOpen(true)}
            className="bg-accent-soft text-accent-soft-foreground hover:bg-accent-soft/80 flex items-center gap-2 rounded-lg border border-accent-soft px-4 py-2 text-sm font-medium shadow-lg transition-colors"
          >
            <ListTree className="h-4 w-4" />
            {t('actions.variablesManager', { count: Object.keys(draft.variables).length })}
          </button>
        </div>

        <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
          {activeStep === 'url' ? (
            <div className="flex min-h-0 flex-1 flex-col">
              <div className="min-h-0 flex-1 space-y-6 overflow-y-auto pr-1">
                <div className="flex items-center gap-4">
                  <TooltipProvider delayDuration={120}>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="inline-flex">
                          <Switch
                            id="external-api-active"
                            aria-label={
                              draft.active ? t('fields.active') : t('fields.inactive')
                            }
                            checked={draft.active}
                            onCheckedChange={(checked) =>
                              updateDraft('active', checked)
                            }
                          />
                        </span>
                      </TooltipTrigger>
                      <TooltipContent>
                        {draft.active ? t('fields.active') : t('fields.inactive')}
                      </TooltipContent>
                    </Tooltip>
                  </TooltipProvider>
                </div>

                <div className="grid gap-4 md:grid-cols-[minmax(0,1.4fr)_minmax(260px,0.8fr)]">
                  <div>
                    <label className="text-content-muted mb-1 block text-sm font-medium">
                      {t('fields.name')}
                    </label>
                    <input
                      type="text"
                      value={draft.name}
                      onChange={(event) => updateDraft('name', event.target.value)}
                      className="w-full rounded-lg border border-border bg-background px-4 py-2 text-foreground focus:border-ring focus:outline-none"
                    />
                  </div>

                  <div>
                    <label className="text-content-muted mb-1 block text-sm font-medium">
                      {t('fields.key')}
                    </label>
                    <div
                      title={derivedKey || t('empty.generatedKey')}
                      className="text-content-body flex h-11 min-w-0 items-center overflow-hidden rounded-lg border border-border bg-background px-4 font-mono text-sm"
                    >
                      <span className="block min-w-0 truncate">
                        {derivedKey || t('empty.generatedKey')}
                      </span>
                    </div>
                  </div>
                </div>

                <p className="text-content-subtle -mt-2 text-xs">{t('hints.key')}</p>

                <div>
                  <label className="text-content-muted mb-1 block text-sm font-medium">
                    {t('fields.url')}
                  </label>
                  <div className="flex gap-2">
                    <select
                      value={draft.method}
                      onChange={(event) =>
                        updateDraft(
                          'method',
                          event.target.value as ExternalApiRequestConfig['method'],
                        )
                      }
                      className="w-32 rounded-lg border border-border bg-background px-4 py-2 pr-10 text-foreground focus:border-ring focus:outline-none"
                    >
                      {['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].map((method) => (
                        <option key={method} value={method}>
                          {method}
                        </option>
                      ))}
                    </select>
                    <HighlightedTemplateInput
                      value={draft.url}
                      onChange={(value) => updateDraft('url', value)}
                      placeholder={t('placeholders.urlTemplate')}
                      className="flex-1 rounded-lg border border-border bg-background focus-within:border-ring"
                    />
                  </div>
                  <p className="text-content-subtle mt-2 flex items-center gap-1 text-xs">
                    <AlertCircle className="h-3 w-3" />
                    {t('hints.url')}
                  </p>
                </div>

                <div className="fe-panel-subtle p-4">
                  <div className="mb-3">
                    <h3 className="text-content-body text-sm font-medium">
                      {t('sections.params')}
                    </h3>
                    <p className="text-content-subtle text-sm">{t('hints.urlDetectedParams')}</p>
                  </div>

                  {urlVariables.length === 0 ? (
                    <p className="text-content-subtle text-sm italic">{t('empty.params')}</p>
                  ) : (
                    <div className="space-y-3">
                      {urlVariables.map(([name, config]) => (
                        <div
                          key={name}
                          className="fe-editor flex flex-wrap items-center justify-between gap-3 px-4 py-3"
                        >
                          <div>
                            <div className="text-accent-soft-foreground font-mono text-sm">
                              {'{{'}
                              {name}
                              {'}}'}
                            </div>
                            <div className="mt-1 flex flex-wrap gap-1">
                              {[...config.locations]
                                .filter((location) => location.startsWith('url_'))
                                .sort()
                                .map((location) => (
                                  <span key={`${name}-${location}`} className="fe-inline-meta">
                                    {location.replace('_', ' ')}
                                  </span>
                                ))}
                            </div>
                          </div>

                          <div className="text-content-subtle text-xs">
                            {config.required
                              ? t('hints.forcedRequired')
                              : t('hints.optionalSupported')}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </div>
          ) : null}

          {activeStep === 'request' ? (
            <div className="flex min-h-0 flex-1 flex-col">
              <div className="flex min-h-0 flex-1 flex-col gap-8 overflow-y-auto pr-1">
                <section className="shrink-0">
                  <div className="mb-4 flex items-center justify-between border-b border-border pb-2">
                    <h3 className="text-content-body text-base font-medium">
                      {t('sections.headers')}
                    </h3>
                    <button
                      type="button"
                      onClick={() =>
                        updateDraft('headers', [...draft.headers, createDraftHeader()])
                      }
                      className="text-accent-soft-foreground hover:text-primary-foreground flex items-center gap-1 text-xs transition-colors"
                    >
                      <Plus className="h-4 w-4" />
                      {t('actions.addHeader')}
                    </button>
                  </div>

                  {draft.headers.length === 0 ? (
                    <p className="text-content-subtle text-sm italic">{t('empty.headers')}</p>
                  ) : (
                    <div className="space-y-3">
                      {draft.headers.map((header) => (
                        <div key={header.id} className="flex items-start gap-3">
                          <HighlightedTemplateInput
                            value={header.key}
                            onChange={(value) =>
                              updateDraft(
                                'headers',
                                draft.headers.map((candidate) =>
                                  candidate.id === header.id
                                    ? { ...candidate, key: value }
                                    : candidate,
                                ),
                              )
                            }
                            placeholder={t('placeholders.headerKey')}
                            className="h-10 w-1/3 rounded-lg border border-border bg-background focus-within:border-ring"
                          />
                          <HighlightedTemplateInput
                            value={header.value}
                            onChange={(value) =>
                              updateDraft(
                                'headers',
                                draft.headers.map((candidate) =>
                                  candidate.id === header.id ? { ...candidate, value } : candidate,
                                ),
                              )
                            }
                            placeholder={t('placeholders.headerValue')}
                            className="h-10 flex-1 rounded-lg border border-border bg-background focus-within:border-ring"
                          />
                          <button
                            type="button"
                            onClick={() =>
                              updateDraft(
                                'headers',
                                draft.headers.filter((candidate) => candidate.id !== header.id),
                              )
                            }
                            className="text-danger-soft-foreground hover:text-danger-soft-foreground/80 h-10 px-2 transition-colors"
                          >
                            <Trash2 className="h-4 w-4" />
                          </button>
                        </div>
                      ))}
                    </div>
                  )}
                </section>

                {hasRequestBody ? (
                  <section className="flex min-h-0 flex-1 flex-col">
                    <div className="mb-4 flex items-center justify-between border-b border-border pb-2">
                      <h3 className="text-content-body text-base font-medium">
                        {t('sections.body')}
                      </h3>

                      <div className="fe-tablist gap-4">
                        {bodyJsonError ? (
                          <span className="text-danger-soft-foreground flex items-center gap-1 text-xs">
                            <AlertCircle className="h-3 w-3" />
                            {bodyJsonError}
                          </span>
                        ) : null}

                        <div className="flex gap-1">
                          <button
                            type="button"
                            onClick={() => handleBodyModeSwitch('visual')}
                            className={`rounded-md px-3 py-1 text-xs font-medium ${
                              draft.bodyMode === 'visual'
                                ? 'bg-surface-subtle text-content-strong'
                                : 'text-content-muted hover:text-content-strong'
                            }`}
                          >
                            {t('bodyModes.visual')}
                          </button>
                          <button
                            type="button"
                            onClick={() => handleBodyModeSwitch('json')}
                            className={`rounded-md px-3 py-1 text-xs font-medium ${
                              draft.bodyMode === 'json'
                                ? 'bg-surface-subtle text-content-strong'
                                : 'text-content-muted hover:text-content-strong'
                            }`}
                          >
                            {t('bodyModes.raw')}
                          </button>
                        </div>
                      </div>
                    </div>

                    {draft.bodyMode === 'json' ? (
                      <p className="text-content-subtle mb-2 text-xs">{t('hints.body')}</p>
                    ) : (
                      <div className="mb-2 flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <span className="text-content-muted text-xs font-medium uppercase">
                            {t('sections.bodyStructure')}
                          </span>
                          <select
                            value={draft.bodyRootType}
                            onChange={(event) =>
                              updateDraft('bodyRootType', event.target.value as BodyRootType)
                            }
                            className="rounded border border-border bg-background px-2 py-1 pr-8 text-[10px] text-content-body focus:outline-none"
                          >
                            <option value="object">{t('bodyRootTypes.object')}</option>
                            <option value="array">{t('bodyRootTypes.array')}</option>
                          </select>
                        </div>
                        <button
                          type="button"
                          onClick={() =>
                            updateDraft('bodyVisual', [
                              ...draft.bodyVisual,
                              {
                                id: `${Date.now()}`,
                                key: '',
                                type: 'string',
                                value: '',
                                children: [],
                              },
                            ])
                          }
                          className="text-accent-soft-foreground hover:text-primary-foreground flex items-center gap-1 text-xs transition-colors"
                        >
                          <Plus className="h-3 w-3" />
                          {draft.bodyRootType === 'array'
                            ? t('bodyActions.addRootItem')
                            : t('bodyActions.addRootField')}
                        </button>
                      </div>
                    )}

                    <div className="grid min-h-[400px] flex-1 gap-6 xl:grid-cols-4 xl:items-stretch">
                      <div className="min-h-0 xl:col-span-3">
                        {draft.bodyMode === 'json' ? (
                          <div className="group relative h-full min-h-[400px]">
                            <HighlightedTemplateTextarea
                              value={draft.bodyRaw}
                              onChange={(value) => {
                                updateDraft('bodyRaw', value);
                                setBodyJsonError(null);
                              }}
                              placeholder={t('placeholders.requestBody')}
                              className="h-full min-h-[400px] w-full rounded-lg border border-border bg-background focus-within:border-ring"
                            />
                            <button
                              type="button"
                              onClick={() => handleFormatJson('bodyRaw')}
                              title={t('actions.formatJson')}
                              className="text-success-soft-foreground absolute bottom-4 right-6 rounded-lg border border-border bg-surface-panel/80 p-2 opacity-0 shadow-lg transition-opacity hover:bg-surface-subtle group-hover:opacity-100"
                            >
                              <AlignLeft className="h-5 w-5" />
                            </button>
                          </div>
                        ) : (
                          <div className="grid h-full min-h-[400px] grid-cols-12 gap-6">
                            <div className="col-span-12 min-h-0 lg:col-span-8">
                              <div className="h-full min-h-[400px] overflow-y-auto pr-2 pb-4 lg:min-h-0">
                                {draft.bodyVisual.length === 0 ? (
                                  <p className="text-content-subtle mt-4 text-center text-sm italic">
                                    {t('bodyActions.empty')}
                                  </p>
                                ) : (
                                  <BodyVisualEditor
                                    nodes={draft.bodyVisual}
                                    rootType={draft.bodyRootType}
                                    onChange={(nodes) => updateDraft('bodyVisual', nodes)}
                                    variant="dark"
                                    showHeader={false}
                                    addFieldLabel={t('actions.addField')}
                                    addItemLabel={t('actions.addItem')}
                                  />
                                )}
                              </div>
                            </div>

                            <div className="fe-editor col-span-12 flex min-h-[400px] flex-col p-4 lg:col-span-4 lg:min-h-0 lg:h-full">
                              <span className="text-content-muted mb-2 text-xs font-medium uppercase">
                                {t('sections.bodyPreview')}
                              </span>
                              <pre className="flex-1 overflow-auto font-mono text-xs">
                                {renderHighlightedText(
                                  JSON.stringify(
                                    buildRequestConfigFromDraft(draft).bodyTemplate ?? {},
                                    null,
                                    2,
                                  ),
                                  'text-success-soft-foreground',
                                )}
                              </pre>
                            </div>
                          </div>
                        )}
                      </div>

                      <div className="min-h-0 xl:col-span-1">
                        <div className="fe-panel-subtle flex h-full min-h-[400px] flex-col p-4 xl:min-h-0">
                          <div className="mb-3 flex items-center justify-between gap-4">
                            <div>
                              <h3 className="text-content-body text-sm font-semibold">
                                {t('sections.secrets')}
                              </h3>
                              <p className="text-content-subtle text-sm">{t('hints.secrets')}</p>
                            </div>
                            {externalApi?.hasSecrets ? (
                              <label className="text-content-body flex items-center gap-2 text-sm">
                                <input
                                  type="checkbox"
                                  checked={draft.replaceSecret}
                                  onChange={(event) =>
                                    updateDraft('replaceSecret', event.target.checked)
                                  }
                                />
                                {t('fields.replaceSecrets')}
                              </label>
                            ) : null}
                          </div>

                          {secretKeys.length === 0 ? (
                            <div className="flex flex-1 items-center justify-center">
                              <p className="text-content-subtle text-center text-sm italic">
                                {t('empty.secrets')}
                              </p>
                            </div>
                          ) : (
                            <div className="flex-1 space-y-3 overflow-y-auto pr-1">
                              {secretKeys.map((secretKey) => (
                                <div key={secretKey} className="space-y-2">
                                  <label className="text-content-body block text-sm">{`{{secret.${secretKey}}}`}</label>
                                  <input
                                    type="password"
                                    value={draft.secretPayload[secretKey] ?? ''}
                                    onChange={(event) =>
                                      updateDraft('secretPayload', {
                                        ...draft.secretPayload,
                                        [secretKey]: event.target.value,
                                      })
                                    }
                                    placeholder={
                                      externalApi?.hasSecrets
                                        ? '••••••••'
                                        : t('placeholders.secret')
                                    }
                                    className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:border-ring focus:outline-none"
                                  />
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  </section>
                ) : (
                  <section>
                    <div className="fe-panel-subtle p-4">
                      <div className="mb-3 flex items-center justify-between gap-4">
                        <div>
                          <h3 className="text-content-body text-sm font-semibold">
                            {t('sections.secrets')}
                          </h3>
                          <p className="text-content-subtle text-sm">{t('hints.secrets')}</p>
                        </div>
                        {externalApi?.hasSecrets ? (
                          <label className="text-content-body flex items-center gap-2 text-sm">
                            <input
                              type="checkbox"
                              checked={draft.replaceSecret}
                              onChange={(event) =>
                                updateDraft('replaceSecret', event.target.checked)
                              }
                            />
                            {t('fields.replaceSecrets')}
                          </label>
                        ) : null}
                      </div>

                      {secretKeys.length === 0 ? (
                        <p className="text-content-subtle text-sm italic">{t('empty.secrets')}</p>
                      ) : (
                        <div className="space-y-3">
                          {secretKeys.map((secretKey) => (
                            <div key={secretKey} className="space-y-2">
                              <label className="text-content-body block text-sm">{`{{secret.${secretKey}}}`}</label>
                              <input
                                type="password"
                                value={draft.secretPayload[secretKey] ?? ''}
                                onChange={(event) =>
                                  updateDraft('secretPayload', {
                                    ...draft.secretPayload,
                                    [secretKey]: event.target.value,
                                  })
                                }
                                placeholder={
                                  externalApi?.hasSecrets ? '••••••••' : t('placeholders.secret')
                                }
                                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:border-ring focus:outline-none"
                              />
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  </section>
                )}
              </div>
            </div>
          ) : null}

          {activeStep === 'validation' ? (
            <div className="flex min-h-0 flex-1 flex-col">
              <div className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto pr-1">
                <div className="shrink-0 rounded-lg border border-accent-soft bg-accent-soft p-4">
                  <p className="text-accent-soft-foreground text-sm">{t('validationIntro')}</p>
                </div>

                <div className="fe-panel-subtle shrink-0 p-4">
                  <label className="text-content-body mb-2 block text-sm font-medium">
                    {t('fields.httpValidation')}
                  </label>
                  <div className="flex items-center gap-4">
                    <select
                      value={draft.httpCodeMode}
                      onChange={(event) =>
                        updateDraft(
                          'httpCodeMode',
                          event.target.value as ExternalApiDraft['httpCodeMode'],
                        )
                      }
                      className="w-full rounded-lg border border-border bg-background px-4 py-2.5 pr-10 text-sm text-content-body focus:border-ring focus:outline-none md:w-1/3"
                    >
                      <option value="2xx">{t('httpModes.2xx')}</option>
                      <option value="custom">{t('httpModes.custom')}</option>
                    </select>

                    {draft.httpCodeMode === 'custom' ? (
                      <input
                        type="text"
                        value={draft.httpCodeCustom}
                        onChange={(event) => updateDraft('httpCodeCustom', event.target.value)}
                        className="flex-1 rounded-lg border border-border bg-background px-4 py-2.5 font-mono text-sm text-foreground focus:border-ring focus:outline-none"
                        placeholder={t('placeholders.httpCodes')}
                      />
                    ) : null}
                  </div>
                  {draft.httpCodeMode === '2xx' ? (
                    <p className="text-content-subtle mt-2 ml-1 text-xs">{t('hints.http2xx')}</p>
                  ) : null}
                </div>

                {draft.validationMode === 'responseBody' || draft.validationMode === 'both' ? (
                  <div className="fe-panel-subtle flex min-h-0 flex-1 flex-col p-4">
                    <div className="mb-4 flex items-center justify-between">
                      <label className="text-content-body block text-sm font-medium">
                        {t('sections.responseLogic')}
                      </label>
                      <div className="flex gap-2">
                        {[
                          { id: 'httpCode', label: t('responseModes.httpCode') },
                          { id: 'responseBody', label: t('responseModes.responseBody') },
                          { id: 'both', label: t('responseModes.both') },
                        ].map((mode) => (
                          <button
                            key={mode.id}
                            type="button"
                            onClick={() =>
                              updateDraft(
                                'validationMode',
                                mode.id as ExternalApiDraft['validationMode'],
                              )
                            }
                            className={`rounded-md px-3 py-1 text-xs font-medium ${
                              draft.validationMode === mode.id
                                ? 'bg-surface-subtle text-content-strong'
                                : 'text-content-muted hover:text-content-strong'
                            }`}
                          >
                            {mode.label}
                          </button>
                        ))}
                      </div>
                    </div>

                    <div className="grid h-full min-h-[400px] flex-1 grid-cols-12 gap-6 lg:items-stretch">
                      {hasSchema ? (
                        <div className="fe-editor group relative col-span-12 flex min-h-[400px] flex-col overflow-hidden lg:col-span-4 lg:min-h-0 lg:h-full">
                          <div className="fe-traffic-header absolute top-0 z-10 w-full">
                            <span className="fe-traffic-title flex items-center gap-1.5">
                              <FileJson className="h-3 w-3" />
                              {t('schemaCard.configured')}
                            </span>
                            <button
                              type="button"
                              onClick={() => setSchemaModalOpen(true)}
                              aria-label={t('actions.editSchema')}
                              title={t('actions.editSchema')}
                              className="bg-accent-soft text-accent-soft-foreground hover:bg-accent-soft/80 rounded border border-accent-soft p-1.5 transition-colors"
                            >
                              <FileJson className="h-3.5 w-3.5" />
                            </button>
                          </div>
                          <pre className="text-content-muted min-h-0 flex-1 overflow-y-auto whitespace-pre-wrap p-4 pt-12 font-mono text-[10px]">
                            {schemaPreview}
                          </pre>
                        </div>
                      ) : (
                        <div className="fe-empty-surface col-span-12 flex min-h-[400px] flex-col items-center justify-center p-4 text-center lg:col-span-4 lg:min-h-0 lg:h-full">
                          <FileJson className="text-content-subtle mb-3 h-10 w-10" />
                          <h4 className="text-content-body mb-1 text-sm font-medium">
                            {t('schemaModal.emptySchemaTitle')}
                          </h4>
                          <p className="text-content-subtle mb-4 px-2 text-[11px]">
                            {t('schemaModal.emptySchemaDescription')}
                          </p>
                          <button
                            type="button"
                            onClick={() => setSchemaModalOpen(true)}
                            className="rounded-lg border border-border bg-surface-panel px-4 py-2 text-xs font-medium text-content-body transition-colors hover:bg-surface-subtle"
                          >
                            {t('actions.configureSchema')}
                          </button>
                        </div>
                      )}

                      <div className="col-span-12 lg:col-span-8">
                        <div className="fe-editor flex min-h-[400px] flex-col overflow-hidden lg:min-h-0 lg:h-full">
                          <div className="fe-traffic-header">
                            <label className="fe-traffic-title flex items-center gap-1.5">
                              <Code className="text-accent-soft-foreground h-3.5 w-3.5" />
                              {t('fields.expression')}
                            </label>
                          </div>
                          <div className="min-h-0 flex-1 p-3">
                            <ResponseExpressionEditor
                              value={draft.expression}
                              onChange={(value) => updateDraft('expression', value)}
                              sampleResponseText={draft.responseSchema}
                              schema={responseValidation.body.schema}
                              variables={draft.variables}
                              responseHeaderKeys={responseHeaderKeys}
                              className="min-h-0"
                            />
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className="fe-panel-subtle p-4">
                    <div className="flex gap-2">
                      {[
                        { id: 'httpCode', label: t('responseModes.httpCode') },
                        { id: 'responseBody', label: t('responseModes.responseBody') },
                        { id: 'both', label: t('responseModes.both') },
                      ].map((mode) => (
                        <button
                          key={mode.id}
                          type="button"
                          onClick={() =>
                            updateDraft(
                              'validationMode',
                              mode.id as ExternalApiDraft['validationMode'],
                            )
                          }
                          className={`rounded-md px-3 py-1 text-xs font-medium ${
                            draft.validationMode === mode.id
                              ? 'bg-surface-subtle text-content-strong'
                              : 'text-content-muted hover:text-content-strong'
                          }`}
                        >
                          {mode.label}
                        </button>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            </div>
          ) : null}

          {activeStep === 'test' ? (
            <ExternalApiTestPanel
              inputs={inputDescriptors}
              paramInputValues={draft.testInputs}
              onParamInputValuesChange={setTestInputs}
              testResult={draft.testResult}
              isTesting={testExternalApi.isPending}
              onRunTest={handleTest}
              onUseResponseAsSchema={
                lastTestResponseBody !== undefined ? handleUseLastResponseAsSchema : undefined
              }
              previewRequest={previewRequest}
              className="min-h-0 flex-1"
            />
          ) : null}
        </div>
      </div>

      <VariablesManagerModal
        open={variablesModalOpen}
        onOpenChange={setVariablesModalOpen}
        variables={draft.variables}
        onAddVariable={addManualVariable}
        onRenameVariable={renameManualVariable}
        onRemoveVariable={removeManualVariable}
        onTypeChange={(name, type) => updateVariableConfig(name, 'type', type)}
        onRequiredChange={(name, required) => updateVariableConfig(name, 'required', required)}
      />

      <ResponseSchemaModal
        open={schemaModalOpen}
        onOpenChange={setSchemaModalOpen}
        schema={responseValidation.body.schema}
        sampleResponseText={draft.responseSchema}
        inputs={inputDescriptors}
        paramInputValues={draft.testInputs}
        onParamInputValuesChange={setTestInputs}
        testResult={draft.testResult}
        isTesting={testExternalApi.isPending}
        onRunTest={handleSchemaTest}
        previewRequest={previewRequest}
        onApply={handleSchemaApply}
      />

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t('delete.title')}
        description={t('delete.description', { key: externalApi?.key ?? derivedKey })}
        variant="destructive"
        onConfirm={handleDelete}
        loading={deleteExternalApi.isPending}
      />

      <UnsavedDialog />
    </div>
  );
}

function buildNextManualVariableName(variables: Record<string, DraftVariableConfig>) {
  let index = 1;
  while (variables[`manual_var_${index}`]) {
    index += 1;
  }
  return `manual_var_${index}`;
}

function hasResponseSchemaSample(value: string) {
  try {
    const parsed = JSON.parse(value) as unknown;
    if (Array.isArray(parsed)) {
      return parsed.length > 0;
    }
    if (parsed && typeof parsed === 'object') {
      return Object.keys(parsed as Record<string, unknown>).length > 0;
    }
  } catch {
    return false;
  }

  return false;
}

function buildVisualNodes(value: unknown) {
  if (value == null || typeof value !== 'object') {
    return [];
  }
  return jsonToVisualNodes(value);
}

function filterFilledSecretPayload(secretPayload: Record<string, string>) {
  return Object.fromEntries(
    Object.entries(secretPayload)
      .filter(([, value]) => value.trim().length > 0)
      .sort(([left], [right]) => left.localeCompare(right)),
  );
}

function buildExternalApiPersistedSnapshot(
  draft: ExternalApiDraft,
  isEditing: boolean,
  parts = buildExternalApiPartsFromDraft(draft),
) {
  return {
    name: draft.name,
    active: draft.active,
    request: parts.request,
    params: parts.params,
    expressionVariables: parts.expressionVariables,
    responseValidation: parts.responseValidation,
    secretPayload: filterFilledSecretPayload(draft.secretPayload),
    replaceSecret: isEditing ? draft.replaceSecret : true,
  };
}

function serializeExternalApiPersistedSnapshot(
  snapshot: ReturnType<typeof buildExternalApiPersistedSnapshot>,
) {
  return JSON.stringify(snapshot);
}

function buildExternalApiSavePayload({
  draft,
  derivedKey,
  expressionVariables,
  isEditing,
  params,
  request,
  responseValidation,
  secretPayload,
}: {
  draft: ExternalApiDraft;
  derivedKey: string;
  expressionVariables: ReturnType<typeof buildExternalApiPartsFromDraft>['expressionVariables'];
  isEditing: boolean;
  params: ReturnType<typeof buildExternalApiPartsFromDraft>['params'];
  request: ReturnType<typeof buildExternalApiPartsFromDraft>['request'];
  responseValidation: ReturnType<typeof buildExternalApiPartsFromDraft>['responseValidation'];
  secretPayload: Record<string, string>;
}) {
  return {
    key: derivedKey,
    name: draft.name,
    active: draft.active,
    request,
    params,
    expressionVariables,
    responseValidation,
    secretPayload,
    replaceSecret: isEditing ? draft.replaceSecret : true,
  };
}

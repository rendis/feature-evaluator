import { useMutation, useQuery } from '@tanstack/react-query';
import {
  ChevronDown,
  Code2,
  Database,
  GitBranch,
  Globe,
  Layers,
  Loader2,
  Plus,
  TestTube2,
  Trash2,
  Waypoints,
} from 'lucide-react';
import { useEffect, useMemo, useState, type ReactNode } from 'react';

import {
  emptyGroup,
  emptyExternalApiCondition,
  emptySegmentCondition,
  emptySegmentFieldOp,
  emptyStaticCondition,
  extractBuilderRoot,
  getOperatorOptions,
  isBuilderGroup,
  isGroupComplete,
  normalizeStaticCondition,
  serializeConditionTree,
  withBuilderMetadata,
  withoutBuilderMetadata,
} from './conditions-builder-types';
import { getVisibleErrorMessage } from '@/lib/display-error';

import type {
  BuilderCondition,
  BuilderConditionKind,
  BuilderConnector,
  BuilderExternalApiCondition,
  BuilderExternalApiParamMapping,
  BuilderFieldCategory,
  BuilderFieldType,
  BuilderGroup,
  BuilderInputRef,
  BuilderOperator,
  BuilderSegmentCondition,
  BuilderSegmentFieldOp,
  BuilderStaticCondition,
} from './conditions-builder-types';
import type {
  ExternalApi,
  ExternalApiBinding,
  Feature,
  FeatureExpressionField,
  FeatureExpressionSchema,
  FeatureExpressionTestResponse,
  InputContract,
  SourceBindings,
} from '@/api/types';

import { expressionApi } from '@/api/expression';
import { flattenSchemaFields } from '@/components/segments/segment-data-utils';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { cn } from '@/lib/utils';
import { expressionQueries } from '@/queries/expression-queries';
import { externalApiQueries } from '@/queries/external-api-queries';
import { segmentQueries } from '@/queries/segment-queries';

interface RuleConditionsCanvasProps {
  feature: Feature;
  initialExpression: string;
  initialMetadata: Record<string, unknown>;
  initialSourceBindings: SourceBindings;
  initialExternalApiBindings?: ExternalApiBinding[];
  onChange: (value: RuleConditionsValue) => void;
}

export interface RuleConditionsValue {
  expression: string;
  metadata: Record<string, unknown>;
  sourceBindings: SourceBindings;
  externalApiBindings: ExternalApiBinding[];
}

interface CatalogFieldOption {
  category: BuilderFieldCategory;
  description?: string;
  detail: string;
  example?: unknown;
  label: string;
  path: string;
  type: BuilderFieldType;
}

interface SearchOption {
  description?: string;
  detail?: string;
  inputRef?: BuilderInputRef;
  fieldType?: BuilderFieldType;
  group?: string;
  keywords?: string;
  label: string;
  value: string;
}

interface SegmentOption {
  key: string;
  label: string;
  recordKeyPath?: string;
}

const EMPTY_EXTERNAL_API_BINDINGS: ExternalApiBinding[] = [];

const CONNECTOR_LABELS: Record<BuilderConnector, string> = {
  and: 'AND',
  or: 'OR',
};

export function RuleConditionsCanvas({
  feature,
  initialExpression,
  initialMetadata,
  initialSourceBindings,
  onChange,
}: RuleConditionsCanvasProps) {
  const inputContract = useMemo<InputContract>(
    () => ({
      headers: feature.inputContract?.headers ?? [],
      requestBodyExample: feature.inputContract?.requestBodyExample,
      requestBodySchema: feature.inputContract?.requestBodySchema,
    }),
    [feature.inputContract],
  );
  const [metadataSeed] = useState(() => withoutBuilderMetadata(initialMetadata));
  const [initialBuilderRoot] = useState<BuilderGroup | null>(() =>
    extractBuilderRoot(initialMetadata),
  );
  const [initialExpressionValue] = useState(initialExpression);
  const [builderRoot, setBuilderRoot] = useState<BuilderGroup>(
    () => initialBuilderRoot ?? emptyGroup(),
  );
  const [mode, setMode] = useState<'guided' | 'advanced'>(() => {
    if (initialBuilderRoot) {
      return 'guided';
    }
    return initialExpressionValue.trim() ? 'advanced' : 'guided';
  });
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const [advancedDraft, setAdvancedDraft] = useState(initialExpressionValue);
  const [advancedExpression, setAdvancedExpression] = useState(initialExpressionValue);
  const [manualSourceBindings, setManualSourceBindings] =
    useState<SourceBindings>(initialSourceBindings);
  const [scenarioHeaders, setScenarioHeaders] = useState<Record<string, string>>(() =>
    Object.fromEntries(inputContract.headers.map((header) => [header.headerName, ''])),
  );
  const [requestBodyText, setRequestBodyText] = useState(() =>
    JSON.stringify(inputContract.requestBodyExample ?? {}, null, 2),
  );
  const [scenarioError, setScenarioError] = useState<string | null>(null);
  const [lastTestResult, setLastTestResult] = useState<FeatureExpressionTestResponse | null>(null);

  const { data: featureSchemaData } = useQuery(expressionQueries.featureSchema(feature.key));
  const { data: segmentListData } = useQuery(segmentQueries.list({ page: 1, pageSize: 200 }));
  const { data: externalApiListData } = useQuery(externalApiQueries.list());

  const featureSchema = useMemo(
    () => featureSchemaData ?? buildFallbackFeatureSchema(inputContract),
    [featureSchemaData, inputContract],
  );
  const inputFields = useMemo(() => toCatalogFieldOptions(featureSchema), [featureSchema]);
  const segmentOptions = useMemo<SegmentOption[]>(
    () =>
      (segmentListData?.data ?? []).map((segment) => ({
        key: segment.key,
        label: segment.name || segment.key,
        recordKeyPath: segment.recordKeyPath,
      })),
    [segmentListData],
  );
  const externalApis = useMemo<ExternalApi[]>(
    () => externalApiListData ?? [],
    [externalApiListData],
  );

  const guidedResult = useMemo(() => serializeConditionTree(builderRoot), [builderRoot]);
  const currentExpression = mode === 'guided' ? guidedResult.expression : advancedExpression;
  const currentSourceBindings =
    mode === 'guided' ? guidedResult.sourceBindings : manualSourceBindings;
  const currentExternalApiBindings =
    mode === 'guided' ? guidedResult.externalApiBindings : EMPTY_EXTERNAL_API_BINDINGS;
  const currentMetadata = useMemo(
    () => (mode === 'guided' ? withBuilderMetadata(metadataSeed, builderRoot) : metadataSeed),
    [builderRoot, metadataSeed, mode],
  );

  useEffect(() => {
    onChange({
      expression: currentExpression,
      metadata: currentMetadata,
      sourceBindings: currentSourceBindings,
      externalApiBindings: currentExternalApiBindings,
    });
  }, [
    currentExpression,
    currentMetadata,
    currentSourceBindings,
    currentExternalApiBindings,
    onChange,
  ]);

  const canReturnToGuided = initialBuilderRoot != null || !initialExpressionValue.trim();

  const testExpression = useMutation({
    mutationFn: async () => {
      let requestBody: Record<string, unknown> = {};
      if (requestBodyText.trim()) {
        const parsed = JSON.parse(requestBodyText) as unknown;
        if (!isObjectRecord(parsed)) {
          throw new Error('El request body debe ser un objeto JSON.');
        }
        requestBody = parsed;
      }

      return expressionApi.featureTest(feature.key, {
        expression: currentExpression,
        sourceBindings: currentSourceBindings,
        externalApiBindings: currentExternalApiBindings,
        scenario: {
          headers: cleanScenarioHeaders(scenarioHeaders),
          requestBody,
        },
      });
    },
    onMutate: () => setScenarioError(null),
    onSuccess: (data) => setLastTestResult(data),
    onError: (error) => {
      setScenarioError(getVisibleErrorMessage(error, 'No se pudo probar la regla.'));
    },
  });

  const handleBuilderChange = (next: BuilderGroup) => {
    setBuilderRoot(next);
  };

  const openAdvancedEditor = () => {
    setAdvancedDraft(currentExpression);
    setAdvancedOpen(true);
  };

  const saveAdvancedEditor = () => {
    setAdvancedExpression(advancedDraft);
    setManualSourceBindings(currentSourceBindings);
    setMode('advanced');
    setAdvancedOpen(false);
  };

  const returnToGuided = () => {
    setMode('guided');
  };

  return (
    <div className="space-y-5">
      <div className="rounded-[28px] border border-border/80 bg-card/95 p-5 shadow-sm">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="space-y-3">
            <Badge variant="secondary" className="rounded-full px-3 py-1 text-[11px] uppercase">
              Condicion de activacion
            </Badge>
            <div className="space-y-1">
              <h4 className="text-lg font-semibold text-foreground">
                La regla aplica cuando se cumple esta logica
              </h4>
              <p className="max-w-2xl text-sm leading-6 text-muted-foreground">
                Construye la condicion como una frase guiada. Puedes comparar datos del request,
                consultar APIs externas o campos de un segmento.
              </p>
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            {mode === 'advanced' && canReturnToGuided ? (
              <Button type="button" variant="outline" onClick={returnToGuided}>
                Volver al builder
              </Button>
            ) : null}
            <Button type="button" variant="outline" onClick={openAdvancedEditor}>
              <Code2 className="mr-2 h-4 w-4" />
              Avanzado
            </Button>
          </div>
        </div>

        <div className="mt-5 rounded-2xl border border-border/70 bg-card/85 p-4">
          <div className="mb-3 flex items-center gap-2 text-sm font-medium">
            <Waypoints className="h-4 w-4 text-muted-foreground" />
            Vista previa de la expresion
          </div>
          <pre className="fe-editor overflow-x-auto whitespace-pre-wrap break-words px-4 py-3 font-mono text-xs leading-6 text-content-body">
            {currentExpression || '// Agrega una condicion para generar la expresion'}
          </pre>
          {currentSourceBindings.segments.length > 0 ? (
            <div className="mt-3 flex flex-wrap gap-2">
              {currentSourceBindings.segments.map((binding) => (
                <Badge
                  key={`${binding.segmentKey}:${binding.lookupPath}`}
                  variant="outline"
                  className="rounded-full px-2 py-0.5 text-[10px]"
                >
                  <Database className="mr-1 h-3 w-3" />
                  {binding.segmentKey} → {binding.lookupPath}
                </Badge>
              ))}
            </div>
          ) : null}
          {currentExternalApiBindings.length > 0 ? (
            <div className="mt-3 flex flex-wrap gap-2">
              {currentExternalApiBindings.map((binding) => (
                <Badge
                  key={binding.externalApiKey}
                  variant="outline"
                  className="rounded-full px-2 py-0.5 text-[10px]"
                >
                  <Globe className="mr-1 h-3 w-3" />
                  {binding.externalApiKey}
                </Badge>
              ))}
            </div>
          ) : null}
        </div>

        {mode === 'guided' ? (
          <div className="mt-5">
            <ConditionGroupEditor
              depth={0}
              group={builderRoot}
              inputFields={inputFields}
              segments={segmentOptions}
              externalApis={externalApis}
              onAddCondition={(groupId, kind) =>
                handleBuilderChange(addNodeToGroup(builderRoot, groupId, createCondition(kind)))
              }
              onAddGroup={(groupId) =>
                handleBuilderChange(addNodeToGroup(builderRoot, groupId, emptyGroup()))
              }
              onConditionChange={(conditionId, updater) =>
                handleBuilderChange(updateNode(builderRoot, conditionId, updater))
              }
              onConnectorChange={(groupId, connector) =>
                handleBuilderChange(
                  updateNode(builderRoot, groupId, (node) =>
                    node.kind === 'group' ? { ...node, connector } : node,
                  ),
                )
              }
              onRemove={(nodeId) => handleBuilderChange(removeNode(builderRoot, nodeId))}
            />
          </div>
        ) : (
          <div className="mt-5 rounded-2xl border border-warning/25 bg-warning/5 p-4">
            <div className="mb-2 text-sm font-semibold text-warning">Modo avanzado</div>
            <textarea
              value={advancedExpression}
              onChange={(event) => setAdvancedExpression(event.target.value)}
              rows={10}
              className="mt-4 flex min-h-[220px] w-full rounded-2xl border border-warning/25 bg-background px-4 py-3 font-mono text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            />
            {currentSourceBindings.segments.length > 0 ? (
              <p className="mt-3 text-xs leading-5 text-muted-foreground">
                Las fuentes conectadas se conservan tal como estan. Si cambias namespaces en el
                codigo, revisa tambien los segmentos resueltos.
              </p>
            ) : null}
          </div>
        )}
      </div>

      <div className="rounded-[28px] border border-border/80 bg-card/95 p-5 shadow-sm">
        <div className="mb-5 flex items-start justify-between gap-4">
          <div className="space-y-1">
            <div className="flex items-center gap-2 text-sm font-semibold">
              <TestTube2 className="h-4 w-4 text-muted-foreground" />
              Simular evaluacion
            </div>
            <p className="max-w-2xl text-sm text-muted-foreground">
              Prueba la regla con datos de ejemplo para ver si matchea.
            </p>
          </div>
          <Button
            type="button"
            size="sm"
            onClick={() => void testExpression.mutateAsync()}
            disabled={
              testExpression.isPending ||
              !currentExpression.trim() ||
              (mode === 'guided' && !isGroupComplete(builderRoot))
            }
          >
            {testExpression.isPending ? (
              <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
            ) : (
              <TestTube2 className="mr-1.5 h-3.5 w-3.5" />
            )}
            Probar regla
          </Button>
        </div>

        <div className="grid gap-4 lg:grid-cols-2">
          <div className="space-y-4">
            <ScenarioHeadersCard
              inputContract={inputContract}
              headers={scenarioHeaders}
              onChange={setScenarioHeaders}
            />
            <RequestBodyCard requestBodyText={requestBodyText} onChange={setRequestBodyText} />
          </div>
          <div className="flex flex-col gap-4">
            {scenarioError ? (
              <div className="rounded-2xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
                {scenarioError}
              </div>
            ) : null}
            <SimulationResultCard
              isPending={testExpression.isPending}
              result={lastTestResult}
              sourceBindings={currentSourceBindings}
            />
          </div>
        </div>
      </div>

      <Dialog open={advancedOpen} onOpenChange={setAdvancedOpen}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>Editar expresion manualmente</DialogTitle>
            <DialogDescription>
              Usa este modo solo si necesitas un caso que el builder guiado no cubre. Al guardar,
              esta regla pasa a modo avanzado.
            </DialogDescription>
          </DialogHeader>
          <textarea
            value={advancedDraft}
            onChange={(event) => setAdvancedDraft(event.target.value)}
            rows={14}
            className="flex min-h-[300px] w-full rounded-2xl border border-input bg-background px-4 py-3 font-mono text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setAdvancedOpen(false)}>
              Cancelar
            </Button>
            <Button type="button" onClick={saveAdvancedEditor}>
              Guardar en avanzado
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Condition Group Editor
// ---------------------------------------------------------------------------

interface ConditionGroupEditorProps {
  depth: number;
  group: BuilderGroup;
  inputFields: CatalogFieldOption[];
  segments: SegmentOption[];
  externalApis: ExternalApi[];
  onAddCondition: (groupId: string, kind: BuilderConditionKind) => void;
  onAddGroup: (groupId: string) => void;
  onConditionChange: (
    conditionId: string,
    updater: (condition: BuilderCondition | BuilderGroup) => BuilderCondition | BuilderGroup,
  ) => void;
  onConnectorChange: (groupId: string, connector: BuilderConnector) => void;
  onRemove: (nodeId: string) => void;
}

function ConditionGroupEditor({
  depth,
  group,
  inputFields,
  segments,
  externalApis,
  onAddCondition,
  onAddGroup,
  onConditionChange,
  onConnectorChange,
  onRemove,
}: ConditionGroupEditorProps) {
  const isRoot = depth === 0;

  return (
    <div className={cn('relative space-y-2', depth > 0 && 'mt-2')}>
      {depth > 0 ? (
        <div className="flex items-center gap-3 mb-2">
          <ConnectorPill
            connector={group.connector}
            onChange={(connector) => onConnectorChange(group.id, connector)}
          />
          <div className="h-px bg-border/40 flex-1" />
          <Button
            type="button"
            variant="ghost"
            className="h-7 px-2 text-xs text-destructive hover:bg-destructive/10"
            onClick={() => onRemove(group.id)}
          >
            <Trash2 className="mr-1.5 h-3 w-3" />
            Eliminar grupo
          </Button>
        </div>
      ) : (
        <div className="mb-4 flex items-center justify-between gap-3">
          <div className="flex items-center gap-2 text-sm font-semibold">
            <GitBranch className="h-4 w-4 text-primary" />
            Condiciones
          </div>
          <ConnectorPill
            connector={group.connector}
            onChange={(connector) => onConnectorChange(group.id, connector)}
          />
        </div>
      )}

      <div className={cn('space-y-2', depth > 0 && 'border-l-2 border-border/40 pl-4 ml-3')}>
        {group.items.map((item) =>
          isBuilderGroup(item) ? (
            <ConditionGroupEditor
              key={item.id}
              depth={depth + 1}
              group={item}
              inputFields={inputFields}
              segments={segments}
              externalApis={externalApis}
              onAddCondition={onAddCondition}
              onAddGroup={onAddGroup}
              onConditionChange={onConditionChange}
              onConnectorChange={onConnectorChange}
              onRemove={onRemove}
            />
          ) : (
            <ConditionCardDispatcher
              key={item.id}
              condition={item}
              inputFields={inputFields}
              segments={segments}
              externalApis={externalApis}
              onChange={(next) => onConditionChange(item.id, () => next)}
              onRemove={() => onRemove(item.id)}
            />
          ),
        )}
      </div>

      {group.items.length === 0 ? (
        <div className="rounded-[16px] border border-dashed border-border/60 bg-muted/10 px-4 py-4 text-center">
          <span className="text-sm text-muted-foreground tracking-tight">
            Regla vacia. Agrega una condicion para empezar.
          </span>
        </div>
      ) : null}

      <div className={cn('flex flex-wrap items-center gap-1.5', !isRoot && 'pt-2')}>
        <ConditionTypePicker onSelect={(kind) => onAddCondition(group.id, kind)} />
        <Button
          type="button"
          variant="ghost"
          className="h-8 rounded-lg text-xs text-muted-foreground hover:bg-muted/50 hover:text-foreground"
          onClick={() => onAddGroup(group.id)}
        >
          <Plus className="mr-1.5 h-3.5 w-3.5" />
          Agregar grupo
        </Button>
        {depth > 0 && (
          <Button
            type="button"
            variant="ghost"
            className="ml-auto h-8 rounded-lg text-xs text-destructive hover:bg-destructive/10 hover:text-destructive"
            onClick={() => onRemove(group.id)}
          >
            <Trash2 className="mr-1.5 h-3 w-3" />
            Eliminar grupo
          </Button>
        )}
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Condition Type Picker (Add condition button with kind selection)
// ---------------------------------------------------------------------------

function ConditionTypePicker({ onSelect }: { onSelect: (kind: BuilderConditionKind) => void }) {
  const [open, setOpen] = useState(false);

  const options: {
    kind: BuilderConditionKind;
    label: string;
    description: string;
    icon: typeof Plus;
  }[] = [
    {
      kind: 'static',
      label: 'Estatica',
      description: 'Compara un dato del request con un valor fijo',
      icon: Layers,
    },
    {
      kind: 'externalApi',
      label: 'API externa',
      description: 'Consulta una API configurada en el workspace',
      icon: Globe,
    },
    {
      kind: 'segment',
      label: 'Segmento',
      description: 'Compara campos de un registro del segmento',
      icon: Database,
    },
  ];

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          className="h-8 rounded-lg text-xs text-muted-foreground hover:bg-muted/50 hover:text-foreground"
        >
          <Plus className="mr-1.5 h-3.5 w-3.5" />
          Agregar condicion
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-[280px] p-1">
        <div className="space-y-0.5">
          {options.map((opt) => (
            <button
              key={opt.kind}
              type="button"
              className="flex w-full items-start gap-3 rounded-lg px-3 py-2.5 text-left transition-colors hover:bg-muted/50"
              onClick={() => {
                onSelect(opt.kind);
                setOpen(false);
              }}
            >
              <opt.icon className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
              <div className="min-w-0">
                <div className="text-sm font-medium">{opt.label}</div>
                <div className="text-xs text-muted-foreground">{opt.description}</div>
              </div>
            </button>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
}

// ---------------------------------------------------------------------------
// Condition Card Dispatcher
// ---------------------------------------------------------------------------

function ConditionCardDispatcher({
  condition,
  inputFields,
  segments,
  externalApis,
  onChange,
  onRemove,
}: {
  condition: BuilderCondition;
  inputFields: CatalogFieldOption[];
  segments: SegmentOption[];
  externalApis: ExternalApi[];
  onChange: (value: BuilderCondition) => void;
  onRemove: () => void;
}) {
  switch (condition.conditionKind) {
    case 'static':
      return (
        <StaticConditionCard
          condition={condition}
          inputFields={inputFields}
          onChange={onChange}
          onRemove={onRemove}
        />
      );
    case 'externalApi':
      return (
        <ExternalApiConditionCard
          condition={condition}
          inputFields={inputFields}
          externalApis={externalApis}
          onChange={onChange}
          onRemove={onRemove}
        />
      );
    case 'segment':
      return (
        <SegmentConditionCard
          condition={condition}
          inputFields={inputFields}
          segments={segments}
          onChange={onChange}
          onRemove={onRemove}
        />
      );
    default:
      return null;
  }
}

// ---------------------------------------------------------------------------
// Static Condition Card
// ---------------------------------------------------------------------------

function StaticConditionCard({
  condition,
  inputFields,
  onChange,
  onRemove,
}: {
  condition: BuilderStaticCondition;
  inputFields: CatalogFieldOption[];
  onChange: (value: BuilderCondition) => void;
  onRemove: () => void;
}) {
  const comparisonOptions = useMemo(() => buildInputFieldSearchOptions(inputFields), [inputFields]);
  const leftType = condition.left?.type ?? 'string';
  const operatorOptions = getOperatorOptions(leftType);
  const isListOp = condition.operator === 'in' || condition.operator === 'not in';

  return (
    <ConditionCardWrapper kind="static" onRemove={onRemove}>
      <SearchPicker
        className="min-w-[240px]"
        emptyLabel="No hay campos disponibles"
        options={comparisonOptions}
        placeholder="Dato de la peticion..."
        value={condition.left ? `input:${condition.left.path}` : undefined}
        onSelect={(option) => {
          if (!option.inputRef) return;
          onChange(normalizeStaticCondition({ ...condition, left: option.inputRef }));
        }}
      />
      <Select
        value={condition.operator}
        onValueChange={(value) =>
          onChange(normalizeStaticCondition({ ...condition, operator: value as BuilderOperator }))
        }
      >
        <SelectTrigger className="w-[120px] rounded-lg bg-background h-9 border-border/60 shadow-sm font-mono text-xs">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {operatorOptions.map((op) => (
            <SelectItem key={op} value={op}>
              {op}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <LiteralValueEditor
        isListOperator={isListOp}
        type={leftType}
        value={condition.rightLiteral}
        onChange={(rightLiteral) =>
          onChange(normalizeStaticCondition({ ...condition, rightLiteral }))
        }
      />
    </ConditionCardWrapper>
  );
}

// ---------------------------------------------------------------------------
// External API Condition Card
// ---------------------------------------------------------------------------

function ExternalApiConditionCard({
  condition,
  inputFields,
  externalApis,
  onChange,
  onRemove,
}: {
  condition: BuilderExternalApiCondition;
  inputFields: CatalogFieldOption[];
  externalApis: ExternalApi[];
  onChange: (value: BuilderCondition) => void;
  onRemove: () => void;
}) {
  const comparisonOptions = useMemo(() => buildInputFieldSearchOptions(inputFields), [inputFields]);
  const apiOptions: SearchOption[] = useMemo(
    () =>
      externalApis
        .filter((api) => api.active)
        .map((api) => ({
          value: api.key,
          label: api.name || api.key,
          detail: api.key,
          keywords: `${api.name} ${api.key}`,
        })),
    [externalApis],
  );

  const handleApiSelect = (option: SearchOption) => {
    const api = externalApis.find((a) => a.key === option.value);
    if (!api) return;
    const mappings: BuilderExternalApiParamMapping[] = (api.params ?? []).map((p) => ({
      paramName: p.name,
      paramType: p.type,
      required: p.required,
      mode: 'input',
      inputRef: null,
      literalValue: '',
    }));
    onChange({
      ...condition,
      externalApiKey: api.key,
      externalApiName: api.name,
      paramMappings: mappings,
    });
  };

  const updateMapping = (index: number, partial: Partial<BuilderExternalApiParamMapping>) => {
    const next = [...condition.paramMappings];
    next[index] = { ...next[index], ...partial };
    onChange({ ...condition, paramMappings: next });
  };

  return (
    <ConditionCardWrapper kind="externalApi" onRemove={onRemove}>
      <div className="flex w-full flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <SearchPicker
            className="min-w-[240px]"
            emptyLabel="No hay APIs configuradas"
            options={apiOptions}
            placeholder="Seleccionar API..."
            value={condition.externalApiKey || undefined}
            onSelect={handleApiSelect}
          />
          <div className="flex items-center gap-2 ml-auto">
            <label className="text-xs text-muted-foreground">Negar</label>
            <Switch
              checked={condition.negate}
              onCheckedChange={(negate) => onChange({ ...condition, negate })}
            />
          </div>
        </div>

        {condition.paramMappings.length > 0 ? (
          <div className="rounded-xl border border-border/60 bg-muted/5 p-3 space-y-2">
            <div className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
              Parametros
            </div>
            {condition.paramMappings.map((mapping, idx) => (
              <div key={mapping.paramName} className="flex flex-wrap items-center gap-2">
                <span className="min-w-[120px] text-sm font-medium">
                  {mapping.paramName}
                  <span className="ml-1 text-xs text-muted-foreground">
                    ({mapping.paramType}
                    {mapping.required ? ', req' : ''})
                  </span>
                </span>
                <Select
                  value={mapping.mode}
                  onValueChange={(mode: string) =>
                    updateMapping(idx, { mode: mode as 'input' | 'literal' })
                  }
                >
                  <SelectTrigger className="w-[100px] h-8 text-xs rounded-lg border-border/60">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="input">Input</SelectItem>
                    <SelectItem value="literal">Literal</SelectItem>
                  </SelectContent>
                </Select>
                {mapping.mode === 'input' ? (
                  <SearchPicker
                    className="min-w-[200px] flex-1"
                    emptyLabel="No hay campos"
                    options={comparisonOptions}
                    placeholder="Campo del request..."
                    value={mapping.inputRef ? `input:${mapping.inputRef.path}` : undefined}
                    onSelect={(option) => {
                      if (!option.inputRef) return;
                      updateMapping(idx, { inputRef: option.inputRef });
                    }}
                  />
                ) : (
                  <Input
                    value={mapping.literalValue}
                    onChange={(e) => updateMapping(idx, { literalValue: e.target.value })}
                    placeholder="Valor..."
                    className="h-8 min-w-[200px] max-w-sm flex-1 rounded-lg text-sm border-border/60"
                  />
                )}
              </div>
            ))}
          </div>
        ) : null}
      </div>
    </ConditionCardWrapper>
  );
}

// ---------------------------------------------------------------------------
// Segment Condition Card
// ---------------------------------------------------------------------------

function SegmentConditionCard({
  condition,
  inputFields,
  segments,
  onChange,
  onRemove,
}: {
  condition: BuilderSegmentCondition;
  inputFields: CatalogFieldOption[];
  segments: SegmentOption[];
  onChange: (value: BuilderCondition) => void;
  onRemove: () => void;
}) {
  const comparisonOptions = useMemo(() => buildInputFieldSearchOptions(inputFields), [inputFields]);
  const segmentPickerOptions = useMemo(() => buildSegmentSearchOptions(segments), [segments]);

  const { data: segmentSchemaData } = useQuery({
    ...segmentQueries.schema(condition.segmentKey),
    enabled: condition.segmentKey.trim().length > 0,
  });
  const segmentFieldOptions = useMemo(
    () => buildSegmentFieldSearchOptions(segmentSchemaData?.schema, condition.segmentKey),
    [segmentSchemaData?.schema, condition.segmentKey],
  );

  const handleSegmentSelect = (option: SearchOption) => {
    const seg = segments.find((s) => s.key === option.value);
    if (!seg) return;

    // Auto-resolve lookupInputRef from the segment's recordKeyPath
    let autoLookup: BuilderInputRef | null = null;
    if (seg.recordKeyPath) {
      const match = inputFields.find(
        (f) => f.path === seg.recordKeyPath || f.path.endsWith(`.${seg.recordKeyPath}`),
      );
      if (match) {
        autoLookup = {
          refKind: 'input',
          category: match.category,
          path: match.path,
          label: match.label,
          type: match.type,
        };
      }
    }

    onChange({
      ...condition,
      segmentKey: seg.key,
      segmentName: seg.label,
      lookupInputRef: autoLookup,
      fieldOps: [],
    });
  };

  const addFieldOp = () => {
    onChange({ ...condition, fieldOps: [...condition.fieldOps, emptySegmentFieldOp()] });
  };

  const updateFieldOp = (index: number, next: BuilderSegmentFieldOp) => {
    const ops = [...condition.fieldOps];
    ops[index] = next;
    onChange({ ...condition, fieldOps: ops });
  };

  const removeFieldOp = (index: number) => {
    onChange({ ...condition, fieldOps: condition.fieldOps.filter((_, i) => i !== index) });
  };

  return (
    <ConditionCardWrapper kind="segment" onRemove={onRemove}>
      <div className="flex w-full flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <SearchPicker
            className="min-w-[200px]"
            emptyLabel="No hay segmentos"
            options={segmentPickerOptions}
            placeholder="Segmento..."
            value={condition.segmentKey || undefined}
            onSelect={handleSegmentSelect}
          />
          {condition.lookupInputRef ? (
            <span className="text-muted-foreground text-xs">
              via{' '}
              <span className="font-mono text-foreground/80">{condition.lookupInputRef.path}</span>
            </span>
          ) : condition.segmentKey ? (
            <SearchPicker
              className="min-w-[200px]"
              emptyLabel="No hay campos"
              options={comparisonOptions}
              placeholder="Campo de busqueda..."
              value={undefined}
              onSelect={(option) => {
                if (!option.inputRef) return;
                onChange({ ...condition, lookupInputRef: option.inputRef });
              }}
            />
          ) : null}
        </div>

        {condition.segmentKey && condition.lookupInputRef ? (
          <div className="rounded-xl border border-border/60 bg-muted/5 p-3 space-y-2">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                  Operaciones sobre campos
                </span>
                <ConnectorPill
                  connector={condition.fieldOpsConnector}
                  onChange={(fieldOpsConnector) => onChange({ ...condition, fieldOpsConnector })}
                />
              </div>
            </div>

            {condition.fieldOps.map((op, idx) => (
              <SegmentFieldOpRow
                key={op.id}
                op={op}
                comparisonOptions={comparisonOptions}
                segmentFieldOptions={segmentFieldOptions}
                onChange={(next) => updateFieldOp(idx, next)}
                onRemove={() => removeFieldOp(idx)}
              />
            ))}

            <Button
              type="button"
              variant="ghost"
              className="h-7 rounded-lg text-xs text-muted-foreground hover:bg-muted/50 hover:text-foreground"
              onClick={addFieldOp}
            >
              <Plus className="mr-1.5 h-3 w-3" />
              Agregar operacion
            </Button>
          </div>
        ) : null}
      </div>
    </ConditionCardWrapper>
  );
}

function SegmentFieldOpRow({
  op,
  comparisonOptions,
  segmentFieldOptions,
  onChange,
  onRemove,
}: {
  op: BuilderSegmentFieldOp;
  comparisonOptions: SearchOption[];
  segmentFieldOptions: SearchOption[];
  onChange: (next: BuilderSegmentFieldOp) => void;
  onRemove: () => void;
}) {
  const operatorOptions = getOperatorOptions(op.fieldType);
  const isListOp = op.operator === 'in' || op.operator === 'not in';

  return (
    <div className="group/op flex flex-wrap items-center gap-2">
      <SearchPicker
        className="min-w-[160px]"
        emptyLabel="Sin campos"
        options={segmentFieldOptions}
        placeholder="Campo..."
        value={op.fieldPath ? `field:${op.fieldPath}` : undefined}
        onSelect={(option) => {
          onChange({
            ...op,
            fieldPath: option.detail ?? option.value,
            fieldLabel: option.label,
            fieldType: option.fieldType ?? 'string',
            operator: getOperatorOptions(option.fieldType ?? 'string')[0],
          });
        }}
      />
      <Select
        value={op.operator}
        onValueChange={(value) => onChange({ ...op, operator: value as BuilderOperator })}
      >
        <SelectTrigger className="w-[100px] rounded-lg bg-background h-8 border-border/60 shadow-sm font-mono text-xs">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {operatorOptions.map((oper) => (
            <SelectItem key={oper} value={oper}>
              {oper}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <div className="flex items-center gap-1.5">
        <div className="inline-flex shrink-0 rounded-lg border border-border/60 bg-muted/10 p-0.5">
          <button
            type="button"
            className={cn(
              'rounded-md px-2 py-0.5 text-[10px] font-semibold tracking-wide transition-colors',
              op.rightMode === 'literal'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:bg-muted/40 hover:text-foreground',
            )}
            onClick={() => onChange({ ...op, rightMode: 'literal' })}
          >
            Literal
          </button>
          <button
            type="button"
            className={cn(
              'rounded-md px-2 py-0.5 text-[10px] font-semibold tracking-wide transition-colors',
              op.rightMode === 'input'
                ? 'bg-background text-foreground shadow-sm'
                : 'text-muted-foreground hover:bg-muted/40 hover:text-foreground',
            )}
            onClick={() => onChange({ ...op, rightMode: 'input' })}
          >
            Input
          </button>
        </div>

        {op.rightMode === 'literal' ? (
          <LiteralValueEditor
            isListOperator={isListOp}
            type={op.fieldType}
            value={op.rightLiteral}
            onChange={(rightLiteral) => onChange({ ...op, rightLiteral })}
          />
        ) : (
          <SearchPicker
            className="min-w-[180px]"
            emptyLabel="No hay campos"
            options={comparisonOptions}
            placeholder="Input..."
            value={op.rightInputRef ? `input:${op.rightInputRef.path}` : undefined}
            onSelect={(option) => {
              if (!option.inputRef) return;
              onChange({ ...op, rightInputRef: option.inputRef });
            }}
          />
        )}
      </div>

      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="h-7 w-7 text-muted-foreground opacity-0 transition-opacity hover:bg-destructive/10 hover:text-destructive group-hover/op:opacity-100"
        onClick={onRemove}
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Condition Card Wrapper (styled by kind)
// ---------------------------------------------------------------------------

const KIND_STYLES: Record<
  BuilderConditionKind,
  { border: string; badge: string; badgeLabel: string }
> = {
  static: {
    border: 'border-border/60',
    badge: 'bg-muted text-muted-foreground',
    badgeLabel: 'Estatica',
  },
  externalApi: {
    border: 'border-blue-500/30',
    badge: 'bg-blue-50 text-blue-700 dark:bg-blue-950 dark:text-blue-300',
    badgeLabel: 'API externa',
  },
  segment: {
    border: 'border-emerald-500/30',
    badge: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300',
    badgeLabel: 'Segmento',
  },
};

function ConditionCardWrapper({
  children,
  kind,
  onRemove,
}: {
  children: ReactNode;
  kind: BuilderConditionKind;
  onRemove: () => void;
}) {
  const styles = KIND_STYLES[kind];
  return (
    <div
      className={cn(
        'group relative flex w-full items-start gap-3 rounded-[16px] border bg-background py-2.5 pl-3 pr-10 shadow-sm transition-colors hover:border-border',
        styles.border,
      )}
    >
      <Badge
        className={cn(
          'mt-[9px] shrink-0 rounded-full px-2 py-0.5 text-[9px] uppercase font-bold tracking-wider border-0',
          styles.badge,
        )}
      >
        {styles.badgeLabel}
      </Badge>
      <div className="flex flex-1 flex-wrap items-start gap-2">{children}</div>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="absolute right-2 top-1/2 h-8 w-8 -translate-y-1/2 text-muted-foreground opacity-0 transition-opacity hover:bg-destructive/10 hover:text-destructive group-hover:opacity-100"
        onClick={onRemove}
      >
        <Trash2 className="h-4 w-4" />
        <span className="sr-only">Eliminar condicion</span>
      </Button>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Shared UI: LiteralValueEditor
// ---------------------------------------------------------------------------

function LiteralValueEditor({
  isListOperator,
  onChange,
  type,
  value,
}: {
  isListOperator: boolean;
  onChange: (value: string) => void;
  type: BuilderFieldType;
  value: string;
}) {
  if (type === 'boolean') {
    return (
      <Select value={value === 'false' ? 'false' : 'true'} onValueChange={onChange}>
        <SelectTrigger className="w-[150px] rounded-lg bg-background h-9 border-border/60 shadow-sm font-medium text-sm">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="true">true</SelectItem>
          <SelectItem value="false">false</SelectItem>
        </SelectContent>
      </Select>
    );
  }

  return (
    <Input
      value={value}
      onChange={(event) => onChange(event.target.value)}
      type={type === 'number' && !isListOperator ? 'number' : 'text'}
      placeholder={isListOperator ? 'valor 1, valor 2, valor 3' : 'valor'}
      className="h-9 min-w-[220px] max-w-sm flex-1 rounded-lg bg-background border-border/60 shadow-sm transition-colors hover:border-border text-sm"
    />
  );
}

// ---------------------------------------------------------------------------
// Scenario & Simulation Cards
// ---------------------------------------------------------------------------

function ScenarioHeadersCard({
  inputContract,
  headers,
  onChange,
}: {
  inputContract: InputContract;
  headers: Record<string, string>;
  onChange: React.Dispatch<React.SetStateAction<Record<string, string>>>;
}) {
  return (
    <div className="rounded-[24px] border border-border/70 bg-muted/10 p-4">
      <div className="mb-3 flex items-center gap-2 text-sm font-semibold">
        <Database className="h-4 w-4 text-muted-foreground" />
        Headers esperados
      </div>
      {inputContract.headers.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          Este feature no tiene headers configurados. Puedes seguir probando solo con request body y
          campos derivados.
        </p>
      ) : (
        <div className="space-y-3">
          {inputContract.headers.map((header) => (
            <div key={header.headerName} className="space-y-1">
              <label className="text-sm font-medium text-foreground">
                {header.label || header.headerName}
              </label>
              <Input
                value={headers[header.headerName] ?? ''}
                onChange={(event) =>
                  onChange((current) => ({
                    ...current,
                    [header.headerName]: event.target.value,
                  }))
                }
                placeholder={`${header.headerName}${header.required ? ' (requerido)' : ''}`}
                className="rounded-2xl bg-background"
              />
              <p className="text-xs text-muted-foreground">
                {header.expressionKey} · {humanizeSchemaType(header.type)}
              </p>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function RequestBodyCard({
  onChange,
  requestBodyText,
}: {
  onChange: (value: string) => void;
  requestBodyText: string;
}) {
  return (
    <div className="rounded-[24px] border border-border/70 bg-muted/10 p-4">
      <div className="mb-3 flex items-center gap-2 text-sm font-semibold">
        <Code2 className="h-4 w-4 text-muted-foreground" />
        Request body
      </div>
      <textarea
        value={requestBodyText}
        onChange={(event) => onChange(event.target.value)}
        rows={12}
        className="flex min-h-[260px] w-full rounded-2xl border border-input bg-background px-4 py-3 font-mono text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      />
    </div>
  );
}

function SimulationResultCard({
  isPending,
  result,
  sourceBindings,
}: {
  isPending: boolean;
  result?: FeatureExpressionTestResponse | null;
  sourceBindings: SourceBindings;
}) {
  return (
    <div className="flex min-h-0 flex-1 flex-col rounded-[24px] border border-border/70 bg-muted/10 p-4">
      <div className="mb-3 flex items-center gap-2 text-sm font-semibold">
        <Waypoints className="h-4 w-4 text-muted-foreground" />
        Resultado
        {isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground" /> : null}
      </div>

      {result ? (
        <div
          className={cn(
            'space-y-4 transition-opacity duration-150',
            isPending && 'pointer-events-none opacity-40',
          )}
        >
          <div className="flex flex-wrap items-center gap-2">
            <Badge
              variant={result.matched ? 'success' : 'outline'}
              className="rounded-full px-3 py-1"
            >
              {result.matched ? 'Hizo match' : 'No hizo match'}
            </Badge>
            {result.explanation ? (
              <span className="text-sm text-muted-foreground">{result.explanation}</span>
            ) : null}
          </div>

          <ResultBlock title="Valor evaluado" value={result.result} />
          <ResultBlock title="Campos procesados resueltos" value={result.derived ?? {}} />
          <ResolvedSourcesBlock
            resolvedSources={result.resolvedSources ?? []}
            sourceBindings={sourceBindings}
          />
          {result.error ? (
            <div className="rounded-2xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
              {result.error}
            </div>
          ) : null}
        </div>
      ) : isPending ? (
        <div className="flex flex-1 items-center justify-center gap-2 text-sm text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          Probando expresion...
        </div>
      ) : (
        <p className="flex-1 text-sm text-muted-foreground">
          Ejecuta una simulacion para ver si la regla matchea y que datos termino usando.
        </p>
      )}
    </div>
  );
}

function ResolvedSourcesBlock({
  resolvedSources,
  sourceBindings,
}: {
  resolvedSources: NonNullable<FeatureExpressionTestResponse['resolvedSources']>;
  sourceBindings: SourceBindings;
}) {
  const visibleSources = resolvedSources.length > 0 ? resolvedSources : sourceBindings.segments;

  return (
    <div className="space-y-2">
      <div className="text-sm font-medium">Sources resueltos</div>
      {visibleSources.length === 0 ? (
        <p className="text-sm text-muted-foreground">Esta regla no usa segmentos conectados.</p>
      ) : (
        <div className="space-y-2">
          {visibleSources.map((source) => {
            const isResolved = 'found' in source;
            const data = 'data' in source ? source.data : undefined;
            const found = 'found' in source ? (source as Record<string, unknown>).found : undefined;

            return (
              <div
                key={`${source.segmentKey}:${source.lookupPath}`}
                className="rounded-2xl border border-border/60 bg-background px-4 py-3"
              >
                <div className="flex flex-wrap items-center gap-2">
                  <Badge
                    variant={isResolved && found ? 'success' : 'outline'}
                    className="rounded-full px-3 py-1"
                  >
                    {source.segmentKey}
                  </Badge>
                  <span className="text-sm text-muted-foreground">lookup: {source.lookupPath}</span>
                  {isResolved ? (
                    <span className="text-sm text-muted-foreground">
                      {found ? 'registro encontrado' : 'sin registro'}
                    </span>
                  ) : null}
                </div>
                {isResolved && data ? (
                  <pre className="fe-editor mt-3 overflow-x-auto whitespace-pre-wrap break-words px-4 py-3 font-mono text-xs leading-6 text-content-body">
                    {JSON.stringify(data, null, 2)}
                  </pre>
                ) : null}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

function ResultBlock({ title, value }: { title: string; value: unknown }) {
  return (
    <div className="space-y-2">
      <div className="text-sm font-medium">{title}</div>
      <pre className="fe-editor overflow-x-auto whitespace-pre-wrap break-words px-4 py-3 font-mono text-xs leading-6 text-content-body">
        {JSON.stringify(value, null, 2)}
      </pre>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Shared UI: ConnectorPill & SearchPicker
// ---------------------------------------------------------------------------

function ConnectorPill({
  connector,
  onChange,
}: {
  connector: BuilderConnector;
  onChange: (connector: BuilderConnector) => void;
}) {
  return (
    <div className="inline-flex rounded-lg border border-border/60 bg-background p-0.5 shadow-sm">
      {(['and', 'or'] as BuilderConnector[]).map((option) => (
        <button
          key={option}
          type="button"
          onClick={() => onChange(option)}
          className={cn(
            'rounded-md px-2.5 py-1 text-[11px] font-semibold tracking-wide transition-colors',
            connector === option
              ? option === 'and'
                ? 'bg-accent-soft text-accent-soft-foreground'
                : 'bg-warning-soft text-warning-soft-foreground'
              : 'text-muted-foreground hover:text-foreground hover:bg-muted/50',
          )}
        >
          {CONNECTOR_LABELS[option]}
        </button>
      ))}
    </div>
  );
}

function SearchPicker({
  className,
  emptyLabel,
  onSelect,
  options,
  placeholder,
  value,
}: {
  className?: string;
  emptyLabel: string;
  onSelect: (option: SearchOption) => void;
  options: SearchOption[];
  placeholder: string;
  value?: string;
}) {
  const [open, setOpen] = useState(false);
  const selected = options.find((option) => option.value === value);
  const grouped = groupSearchOptions(options);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <button
          type="button"
          className={cn(
            'flex h-9 items-center justify-between gap-2 rounded-lg border border-border/60 bg-background pl-3 pr-2 text-left text-sm text-foreground shadow-sm transition-colors hover:border-border',
            className,
          )}
        >
          <span className="min-w-0 flex-1 truncate font-medium">
            {selected ? (
              selected.label
            ) : (
              <span className="text-muted-foreground font-normal">{placeholder}</span>
            )}
          </span>
          <ChevronDown className="h-4 w-4 shrink-0 text-muted-foreground opacity-50" />
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-[340px] p-0">
        <Command>
          <CommandInput placeholder="Buscar..." />
          <CommandList>
            <CommandEmpty>{emptyLabel}</CommandEmpty>
            {grouped.map(([group, groupOptions]) => (
              <CommandGroup key={group} heading={group === 'default' ? undefined : group}>
                {groupOptions.map((option) => (
                  <CommandItem
                    key={option.value}
                    value={`${option.label} ${option.detail ?? ''} ${option.keywords ?? ''}`}
                    onSelect={() => {
                      onSelect(option);
                      setOpen(false);
                    }}
                  >
                    <div className="min-w-0">
                      <div className="truncate font-medium">{option.label}</div>
                      {option.detail ? (
                        <div className="truncate text-xs text-muted-foreground">
                          {option.detail}
                        </div>
                      ) : null}
                      {option.description ? (
                        <div className="truncate text-xs text-muted-foreground">
                          {option.description}
                        </div>
                      ) : null}
                    </div>
                  </CommandItem>
                ))}
              </CommandGroup>
            ))}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

// ---------------------------------------------------------------------------
// Tree mutation helpers
// ---------------------------------------------------------------------------

function createCondition(kind: BuilderConditionKind): BuilderCondition {
  switch (kind) {
    case 'static':
      return emptyStaticCondition();
    case 'externalApi':
      return emptyExternalApiCondition();
    case 'segment':
      return emptySegmentCondition();
  }
}

function addNodeToGroup(
  group: BuilderGroup,
  groupId: string,
  node: BuilderCondition | BuilderGroup,
): BuilderGroup {
  if (group.id === groupId) {
    return {
      ...group,
      items: [...group.items, node],
    };
  }

  return {
    ...group,
    items: group.items.map((item): BuilderCondition | BuilderGroup =>
      isBuilderGroup(item) ? addNodeToGroup(item, groupId, node) : item,
    ),
  };
}

function updateNode(
  group: BuilderGroup,
  nodeId: string,
  updater: (node: BuilderCondition | BuilderGroup) => BuilderCondition | BuilderGroup,
): BuilderGroup {
  if (group.id === nodeId) {
    const updated = updater(group);
    return updated.kind === 'group' ? updated : group;
  }

  return {
    ...group,
    items: group.items.map((item) => {
      if (item.id === nodeId) {
        return updater(item);
      }
      if (isBuilderGroup(item)) {
        return updateNode(item, nodeId, updater);
      }
      return item;
    }),
  };
}

function removeNode(group: BuilderGroup, nodeId: string): BuilderGroup {
  const remainingItems = group.items
    .filter((item) => item.id !== nodeId)
    .map((item) => (isBuilderGroup(item) ? removeNode(item, nodeId) : item));

  return {
    ...group,
    items: remainingItems,
  };
}

// ---------------------------------------------------------------------------
// Data helpers
// ---------------------------------------------------------------------------

function toCatalogFieldOptions(schema: FeatureExpressionSchema): CatalogFieldOption[] {
  return [...schema.headers, ...schema.requestBody, ...schema.derived].map((field) => ({
    category: field.group as BuilderFieldCategory,
    description: field.description,
    detail: field.path,
    example: field.example,
    label: field.label,
    path: field.path,
    type: inferBuilderFieldType(field.type),
  }));
}

function buildFallbackFeatureSchema(inputContract: InputContract): FeatureExpressionSchema {
  return {
    headers: (inputContract.headers ?? []).map((header) => ({
      path: `headers.${header.expressionKey}`,
      label: header.label || header.headerName,
      description: header.description,
      type: header.type,
      example: header.headerName,
      group: 'headers',
    })),
    requestBody: flattenInputContractSchema(inputContract.requestBodySchema, ''),
    derived: [
      { path: 'derived.authenticated', label: 'Autenticado', type: 'boolean', group: 'derived' },
      {
        path: 'derived.bearerTokenPresent',
        label: 'Bearer presente',
        type: 'boolean',
        group: 'derived',
      },
      {
        path: 'derived.apiKeyPresent',
        label: 'API key presente',
        type: 'boolean',
        group: 'derived',
      },
      { path: 'derived.userId', label: 'User ID', type: 'string', group: 'derived' },
      { path: 'derived.email', label: 'Email', type: 'string', group: 'derived' },
    ],
    advancedMode: true,
  };
}

function flattenInputContractSchema(
  schema: Record<string, unknown> | undefined,
  prefix: string,
): FeatureExpressionField[] {
  if (!schema || !isObjectRecord(schema)) {
    return [];
  }

  const properties = isObjectRecord(schema.properties) ? schema.properties : null;
  if (!properties) {
    return [];
  }

  const fields: FeatureExpressionField[] = [];
  for (const [key, rawChild] of Object.entries(properties)) {
    if (!isObjectRecord(rawChild)) {
      continue;
    }
    const childPath = prefix ? `${prefix}.${key}` : key;
    const childType = Array.isArray(rawChild.type)
      ? (rawChild.type.find(
          (entry): entry is string => typeof entry === 'string' && entry !== 'null',
        ) ?? 'string')
      : typeof rawChild.type === 'string'
        ? rawChild.type
        : 'string';

    if (childType === 'object') {
      fields.push(...flattenInputContractSchema(rawChild, childPath));
      continue;
    }

    fields.push({
      path: `requestBody.${childPath}`,
      label: childPath,
      type: childType,
      group: 'requestBody',
    });
  }
  return fields;
}

function buildInputFieldSearchOptions(inputFields: CatalogFieldOption[]): SearchOption[] {
  return inputFields.map((field) => ({
    value: `input:${field.path}`,
    label: field.label,
    group: categoryLabel(field.category),
    detail: field.path,
    keywords: `${field.path} ${field.label} ${field.detail}`,
    inputRef: {
      refKind: 'input' as const,
      category: field.category,
      path: field.path,
      label: field.label,
      type: field.type,
    },
  }));
}

function buildSegmentSearchOptions(segments: SegmentOption[]): SearchOption[] {
  return segments.map((segment) => ({
    value: segment.key,
    label: segment.label,
    detail: segment.key,
    description: segment.recordKeyPath
      ? `Clave del segmento: ${segment.recordKeyPath}`
      : 'Sin record key configurado',
    keywords: `${segment.label} ${segment.key}`,
  }));
}

function buildSegmentFieldSearchOptions(
  schema: Record<string, unknown> | undefined,
  segmentKey: string,
): SearchOption[] {
  if (!schema || !segmentKey) {
    return [];
  }

  return flattenSchemaFields(schema).map((field) => ({
    value: `field:${field.path}`,
    label: field.path,
    group: 'Campos del segmento',
    detail: field.path,
    description: field.required ? 'Requerido en el schema del segmento' : 'Opcional',
    keywords: `${field.path} ${field.type} ${segmentKey}`,
    fieldType: inferBuilderFieldType(field.type),
  }));
}

function groupSearchOptions(options: SearchOption[]): [string, SearchOption[]][] {
  const groups = new Map<string, SearchOption[]>();
  for (const option of options) {
    const key = option.group ?? 'default';
    const groupOptions = groups.get(key) ?? [];
    groupOptions.push(option);
    groups.set(key, groupOptions);
  }
  return [...groups.entries()];
}

function categoryLabel(category: BuilderFieldCategory): string {
  switch (category) {
    case 'headers':
      return 'Headers';
    case 'requestBody':
      return 'Request body';
    case 'derived':
      return 'Campos procesados';
  }
}

function cleanScenarioHeaders(headers: Record<string, string>): Record<string, string> {
  return Object.fromEntries(Object.entries(headers).filter(([, value]) => value.trim().length > 0));
}

function humanizeSchemaType(type: string): string {
  const normalized = type.toLowerCase();
  if (normalized.includes('bool')) {
    return 'booleano';
  }
  if (normalized.includes('int') || normalized.includes('number')) {
    return 'numero';
  }
  return 'texto';
}

function inferBuilderFieldType(type: string | undefined): BuilderFieldType {
  if (!type) {
    return 'unknown';
  }
  const normalized = type.toLowerCase();
  if (normalized.includes('bool')) {
    return 'boolean';
  }
  if (normalized.includes('int') || normalized.includes('number')) {
    return 'number';
  }
  if (normalized.includes('string') || normalized.includes('text')) {
    return 'string';
  }
  return 'unknown';
}

function isObjectRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

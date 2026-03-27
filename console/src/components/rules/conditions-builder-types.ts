import type { ExternalApiBinding, SourceBindings } from '@/api/types';

export const RULE_CONDITIONS_BUILDER_METADATA_KEY = 'conditionsBuilder';
const BUILDER_VERSION = 2;

// ---------------------------------------------------------------------------
// Shared primitives
// ---------------------------------------------------------------------------

export type BuilderConnector = 'and' | 'or';
export type BuilderOperator =
  | '=='
  | '!='
  | '>'
  | '>='
  | '<'
  | '<='
  | 'contains'
  | 'startsWith'
  | 'endsWith'
  | 'matches'
  | 'in'
  | 'not in';

export type BuilderFieldCategory = 'headers' | 'requestBody' | 'derived';
export type BuilderFieldType = 'string' | 'number' | 'boolean' | 'unknown';

export type BuilderConditionKind = 'static' | 'externalApi' | 'segment';

export interface BuilderInputRef {
  refKind: 'input';
  category: BuilderFieldCategory;
  path: string;
  label: string;
  type: BuilderFieldType;
}

// ---------------------------------------------------------------------------
// 1. Static condition — input_param operator literal
// ---------------------------------------------------------------------------

export interface BuilderStaticCondition {
  id: string;
  kind: 'condition';
  conditionKind: 'static';
  left: BuilderInputRef | null;
  operator: BuilderOperator;
  rightLiteral: string;
}

// ---------------------------------------------------------------------------
// 2. External API condition — externalApi("key") → boolean
// ---------------------------------------------------------------------------

export interface BuilderExternalApiParamMapping {
  paramName: string;
  paramType: string;
  required: boolean;
  mode: 'input' | 'literal';
  inputRef: BuilderInputRef | null;
  literalValue: string;
}

export interface BuilderExternalApiCondition {
  id: string;
  kind: 'condition';
  conditionKind: 'externalApi';
  externalApiKey: string;
  externalApiName: string;
  cacheEnabled: boolean;
  cacheTTL: number;
  paramMappings: BuilderExternalApiParamMapping[];
  negate: boolean;
}

// ---------------------------------------------------------------------------
// 3. Segment condition — segmentKey.field operator value
// ---------------------------------------------------------------------------

export interface BuilderSegmentFieldOp {
  id: string;
  fieldPath: string;
  fieldLabel: string;
  fieldType: BuilderFieldType;
  operator: BuilderOperator;
  rightMode: 'literal' | 'input';
  rightLiteral: string;
  rightInputRef: BuilderInputRef | null;
}

export interface BuilderSegmentCondition {
  id: string;
  kind: 'condition';
  conditionKind: 'segment';
  segmentKey: string;
  segmentName: string;
  lookupInputRef: BuilderInputRef | null;
  fieldOps: BuilderSegmentFieldOp[];
  fieldOpsConnector: BuilderConnector;
}

// ---------------------------------------------------------------------------
// Union types
// ---------------------------------------------------------------------------

export type BuilderCondition =
  | BuilderStaticCondition
  | BuilderExternalApiCondition
  | BuilderSegmentCondition;

export interface BuilderGroup {
  id: string;
  kind: 'group';
  connector: BuilderConnector;
  items: BuilderNode[];
}

export type BuilderNode = BuilderCondition | BuilderGroup;

export interface BuilderMetadata {
  version: number;
  root: BuilderGroup;
}

// ---------------------------------------------------------------------------
// Serialization result
// ---------------------------------------------------------------------------

export interface SerializedConditionTree {
  expression: string;
  sourceBindings: SourceBindings;
  externalApiBindings: ExternalApiBinding[];
}

// ---------------------------------------------------------------------------
// Factory helpers
// ---------------------------------------------------------------------------

export function createBuilderId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return Math.random().toString(36).slice(2, 10);
}

export function emptyStaticCondition(): BuilderStaticCondition {
  return {
    id: createBuilderId(),
    kind: 'condition',
    conditionKind: 'static',
    left: null,
    operator: '==',
    rightLiteral: '',
  };
}

export function emptyExternalApiCondition(): BuilderExternalApiCondition {
  return {
    id: createBuilderId(),
    kind: 'condition',
    conditionKind: 'externalApi',
    externalApiKey: '',
    externalApiName: '',
    cacheEnabled: false,
    cacheTTL: 300,
    paramMappings: [],
    negate: false,
  };
}

export function emptySegmentCondition(): BuilderSegmentCondition {
  return {
    id: createBuilderId(),
    kind: 'condition',
    conditionKind: 'segment',
    segmentKey: '',
    segmentName: '',
    lookupInputRef: null,
    fieldOps: [],
    fieldOpsConnector: 'and',
  };
}

export function emptySegmentFieldOp(): BuilderSegmentFieldOp {
  return {
    id: createBuilderId(),
    fieldPath: '',
    fieldLabel: '',
    fieldType: 'string',
    operator: '==',
    rightMode: 'literal',
    rightLiteral: '',
    rightInputRef: null,
  };
}

export function emptyGroup(connector: BuilderConnector = 'and'): BuilderGroup {
  return {
    id: createBuilderId(),
    kind: 'group',
    connector,
    items: [],
  };
}

export function isBuilderGroup(node: BuilderNode): node is BuilderGroup {
  return node.kind === 'group';
}

// ---------------------------------------------------------------------------
// Operators
// ---------------------------------------------------------------------------

export function getOperatorOptions(type: BuilderFieldType): BuilderOperator[] {
  if (type === 'boolean') {
    return ['==', '!='];
  }
  if (type === 'number') {
    return ['==', '!=', '>', '>=', '<', '<=', 'in', 'not in'];
  }
  return ['==', '!=', 'contains', 'startsWith', 'endsWith', 'matches', 'in', 'not in'];
}

export function normalizeStaticCondition(c: BuilderStaticCondition): BuilderStaticCondition {
  const allowed = getOperatorOptions(c.left?.type ?? 'string');
  const operator = allowed.includes(c.operator) ? c.operator : allowed[0];
  if (
    c.left?.type === 'boolean' &&
    c.rightLiteral !== 'true' &&
    c.rightLiteral !== 'false'
  ) {
    return { ...c, operator, rightLiteral: 'true' };
  }
  return { ...c, operator };
}

// ---------------------------------------------------------------------------
// Serialization — conditions → expression + bindings
// ---------------------------------------------------------------------------

export function serializeConditionTree(root: BuilderGroup): SerializedConditionTree {
  const segmentBindings = new Map<string, { segmentKey: string; lookupPath: string }>();
  const externalApiBindings: ExternalApiBinding[] = [];
  const seenApiKeys = new Set<string>();

  const expression = serializeGroup(root, segmentBindings, externalApiBindings, seenApiKeys);

  return {
    expression,
    sourceBindings: { segments: [...segmentBindings.values()] },
    externalApiBindings,
  };
}

function serializeGroup(
  group: BuilderGroup,
  segmentBindings: Map<string, { segmentKey: string; lookupPath: string }>,
  externalApiBindings: ExternalApiBinding[],
  seenApiKeys: Set<string>,
): string {
  const parts = group.items
    .map((item) => serializeNode(item, segmentBindings, externalApiBindings, seenApiKeys))
    .filter((v) => v.trim().length > 0);

  if (parts.length === 0) return '';
  const connector = group.connector === 'and' ? ' && ' : ' || ';
  return parts.join(connector);
}

function serializeNode(
  node: BuilderNode,
  segmentBindings: Map<string, { segmentKey: string; lookupPath: string }>,
  externalApiBindings: ExternalApiBinding[],
  seenApiKeys: Set<string>,
): string {
  if (isBuilderGroup(node)) {
    const inner = serializeGroup(node, segmentBindings, externalApiBindings, seenApiKeys);
    return inner ? `(${inner})` : '';
  }

  switch (node.conditionKind) {
    case 'static':
      return serializeStaticCondition(node);
    case 'externalApi':
      return serializeExternalApiCondition(node, externalApiBindings, seenApiKeys);
    case 'segment':
      return serializeSegmentCondition(node, segmentBindings);
    default:
      return '';
  }
}

function serializeStaticCondition(c: BuilderStaticCondition): string {
  if (!c.left || !c.left.path.trim()) return '';
  const left = c.left.path;
  const right = formatLiteral(c.rightLiteral, c.left.type, c.operator);
  if (!right) return '';
  return `${left} ${c.operator} ${right}`;
}

function serializeExternalApiCondition(
  c: BuilderExternalApiCondition,
  bindings: ExternalApiBinding[],
  seenApiKeys: Set<string>,
): string {
  if (!c.externalApiKey) return '';

  if (!seenApiKeys.has(c.externalApiKey)) {
    seenApiKeys.add(c.externalApiKey);
    bindings.push({
      externalApiKey: c.externalApiKey,
      paramMappings: c.paramMappings
        .filter((m) => m.mode === 'literal' ? m.literalValue.trim().length > 0 : m.inputRef != null)
        .map((m) => ({
          paramName: m.paramName,
          mode: m.mode,
          inputPath: m.mode === 'input' ? (m.inputRef?.path ?? '') : '',
          literalValue: m.mode === 'literal' ? m.literalValue : '',
        })),
      failMode: 'open',
      cacheEnabled: c.cacheEnabled,
      cacheTTL: c.cacheEnabled ? c.cacheTTL : 0,
    });
  }

  const call = `externalApi("${c.externalApiKey}")`;
  return c.negate ? `!${call}` : call;
}

function serializeSegmentCondition(
  c: BuilderSegmentCondition,
  segmentBindings: Map<string, { segmentKey: string; lookupPath: string }>,
): string {
  if (!c.segmentKey || !c.lookupInputRef) return '';

  const bindingKey = `${c.segmentKey}:${c.lookupInputRef.path}`;
  if (!segmentBindings.has(bindingKey)) {
    segmentBindings.set(bindingKey, {
      segmentKey: c.segmentKey,
      lookupPath: c.lookupInputRef.path,
    });
  }

  if (c.fieldOps.length === 0) return '';

  const parts = c.fieldOps
    .filter((op) => isFieldOpComplete(op))
    .map((op) => {
      const left = `${c.segmentKey}.${op.fieldPath}`;
      const right =
        op.rightMode === 'input' && op.rightInputRef
          ? op.rightInputRef.path
          : formatLiteral(op.rightLiteral, op.fieldType, op.operator);
      return `${left} ${op.operator} ${right}`;
    })
    .filter((v) => v.length > 0);

  if (parts.length === 0) return '';
  if (parts.length === 1) return parts[0];
  const connector = c.fieldOpsConnector === 'and' ? ' && ' : ' || ';
  return `(${parts.join(connector)})`;
}

// ---------------------------------------------------------------------------
// Completeness checks
// ---------------------------------------------------------------------------

export function isGroupComplete(group: BuilderGroup): boolean {
  if (group.items.length === 0) return false;
  return group.items.every((item) => {
    if (isBuilderGroup(item)) return isGroupComplete(item);
    return isConditionComplete(item);
  });
}

export function isConditionComplete(condition: BuilderCondition): boolean {
  switch (condition.conditionKind) {
    case 'static':
      return isStaticComplete(condition);
    case 'externalApi':
      return isExternalApiComplete(condition);
    case 'segment':
      return isSegmentComplete(condition);
    default:
      return false;
  }
}

function isStaticComplete(c: BuilderStaticCondition): boolean {
  if (!c.left || !c.left.path.trim()) return false;
  if (c.left.type === 'boolean') {
    return c.rightLiteral === 'true' || c.rightLiteral === 'false';
  }
  return c.rightLiteral.trim().length > 0;
}

function isExternalApiComplete(c: BuilderExternalApiCondition): boolean {
  if (!c.externalApiKey) return false;
  return c.paramMappings
    .filter((m) => m.required)
    .every((m) => (m.mode === 'input' ? m.inputRef != null : m.literalValue.trim().length > 0));
}

function isSegmentComplete(c: BuilderSegmentCondition): boolean {
  if (!c.segmentKey || !c.lookupInputRef) return false;
  if (c.fieldOps.length === 0) return false;
  return c.fieldOps.every(isFieldOpComplete);
}

function isFieldOpComplete(op: BuilderSegmentFieldOp): boolean {
  if (!op.fieldPath.trim()) return false;
  if (op.rightMode === 'input') return op.rightInputRef != null;
  if (op.fieldType === 'boolean') {
    return op.rightLiteral === 'true' || op.rightLiteral === 'false';
  }
  return op.rightLiteral.trim().length > 0;
}

// ---------------------------------------------------------------------------
// Metadata persistence
// ---------------------------------------------------------------------------

export function buildBuilderMetadata(root: BuilderGroup): BuilderMetadata {
  return { version: BUILDER_VERSION, root };
}

export function extractBuilderRoot(
  metadata: Record<string, unknown> | undefined,
): BuilderGroup | null {
  if (!metadata) return null;
  const raw = metadata[RULE_CONDITIONS_BUILDER_METADATA_KEY];
  if (!isRecord(raw)) return null;
  if (raw.version !== BUILDER_VERSION) return null;
  return parseGroup(raw.root);
}

export function withoutBuilderMetadata(
  metadata: Record<string, unknown> | undefined,
): Record<string, unknown> {
  if (!metadata) return {};
  const { [RULE_CONDITIONS_BUILDER_METADATA_KEY]: _, ...rest } = metadata;
  return rest;
}

export function withBuilderMetadata(
  metadata: Record<string, unknown> | undefined,
  root: BuilderGroup,
): Record<string, unknown> {
  return {
    ...(metadata ?? {}),
    [RULE_CONDITIONS_BUILDER_METADATA_KEY]: buildBuilderMetadata(root),
  };
}

// ---------------------------------------------------------------------------
// Literal formatting
// ---------------------------------------------------------------------------

function formatLiteral(value: string, type: BuilderFieldType, operator: BuilderOperator): string {
  if (operator === 'in' || operator === 'not in') {
    const entries = value
      .split(',')
      .map((e) => e.trim())
      .filter(Boolean)
      .map((e) => formatScalar(e, type));
    return entries.length > 0 ? `[${entries.join(', ')}]` : '';
  }
  return formatScalar(value, type);
}

function formatScalar(value: string, type: BuilderFieldType): string {
  if (type === 'boolean') return value === 'false' ? 'false' : 'true';
  if (type === 'number') {
    const n = Number(value);
    return Number.isFinite(n) ? String(n) : '0';
  }
  return JSON.stringify(value);
}

// ---------------------------------------------------------------------------
// Parsing from metadata (restore from saved state)
// ---------------------------------------------------------------------------

function parseGroup(value: unknown): BuilderGroup | null {
  if (!isRecord(value) || value.kind !== 'group' || typeof value.id !== 'string') return null;
  const v = value as Record<string, unknown>;
  const connector: BuilderConnector = v.connector === 'or' ? 'or' : 'and';
  const rawItems = Array.isArray(v.items) ? v.items : [];
  const items = rawItems
    .map((item) => parseNode(item))
    .filter((item): item is BuilderNode => item != null);
  return { id: v.id as string, kind: 'group', connector, items };
}

function parseNode(value: unknown): BuilderNode | null {
  if (!isRecord(value)) return null;
  if (value.kind === 'group') return parseGroup(value);
  if (value.kind !== 'condition') return null;

  const v = value as Record<string, unknown>;
  if (typeof v.id !== 'string') return null;

  const conditionKind = v.conditionKind as string;

  switch (conditionKind) {
    case 'static':
      return parseStaticCondition(v);
    case 'externalApi':
      return parseExternalApiCondition(v);
    case 'segment':
      return parseSegmentCondition(v);
    default:
      return null;
  }
}

function parseStaticCondition(v: Record<string, unknown>): BuilderStaticCondition {
  return {
    id: v.id as string,
    kind: 'condition',
    conditionKind: 'static',
    left: parseInputRef(v.left),
    operator: parseOperator(v.operator),
    rightLiteral: typeof v.rightLiteral === 'string' ? v.rightLiteral : '',
  };
}

function parseExternalApiCondition(v: Record<string, unknown>): BuilderExternalApiCondition {
  const rawMappings = Array.isArray(v.paramMappings) ? v.paramMappings : [];
  return {
    id: v.id as string,
    kind: 'condition',
    conditionKind: 'externalApi',
    externalApiKey: typeof v.externalApiKey === 'string' ? v.externalApiKey : '',
    externalApiName: typeof v.externalApiName === 'string' ? v.externalApiName : '',
    cacheEnabled: v.cacheEnabled === true,
    cacheTTL: typeof v.cacheTTL === 'number' && Number.isFinite(v.cacheTTL) ? v.cacheTTL : 300,
    paramMappings: rawMappings.map(parseParamMapping).filter(Boolean) as BuilderExternalApiParamMapping[],
    negate: v.negate === true,
  };
}

function parseSegmentCondition(v: Record<string, unknown>): BuilderSegmentCondition {
  const rawOps = Array.isArray(v.fieldOps) ? v.fieldOps : [];
  return {
    id: v.id as string,
    kind: 'condition',
    conditionKind: 'segment',
    segmentKey: typeof v.segmentKey === 'string' ? v.segmentKey : '',
    segmentName: typeof v.segmentName === 'string' ? v.segmentName : '',
    lookupInputRef: parseInputRef(v.lookupInputRef),
    fieldOps: rawOps.map(parseFieldOp).filter(Boolean) as BuilderSegmentFieldOp[],
    fieldOpsConnector: v.fieldOpsConnector === 'or' ? 'or' : 'and',
  };
}

function parseParamMapping(value: unknown): BuilderExternalApiParamMapping | null {
  if (!isRecord(value) || typeof value.paramName !== 'string') return null;
  return {
    paramName: value.paramName,
    paramType: typeof value.paramType === 'string' ? value.paramType : 'any',
    required: value.required === true,
    mode: value.mode === 'literal' ? 'literal' : 'input',
    inputRef: parseInputRef(value.inputRef),
    literalValue: typeof value.literalValue === 'string' ? value.literalValue : '',
  };
}

function parseFieldOp(value: unknown): BuilderSegmentFieldOp | null {
  if (!isRecord(value) || typeof value.id !== 'string') return null;
  return {
    id: value.id as string,
    fieldPath: typeof value.fieldPath === 'string' ? value.fieldPath : '',
    fieldLabel: typeof value.fieldLabel === 'string' ? value.fieldLabel : '',
    fieldType: parseFieldType(value.fieldType),
    operator: parseOperator(value.operator),
    rightMode: value.rightMode === 'input' ? 'input' : 'literal',
    rightLiteral: typeof value.rightLiteral === 'string' ? value.rightLiteral : '',
    rightInputRef: parseInputRef(value.rightInputRef),
  };
}

function parseInputRef(value: unknown): BuilderInputRef | null {
  if (!isRecord(value) || typeof value.path !== 'string' || typeof value.label !== 'string') {
    return null;
  }
  return {
    refKind: 'input',
    category: parseFieldCategory(value.category),
    path: value.path,
    label: value.label,
    type: parseFieldType(value.type),
  };
}

function parseFieldCategory(value: unknown): BuilderFieldCategory {
  if (value === 'headers' || value === 'requestBody' || value === 'derived') return value;
  return 'headers';
}

function parseFieldType(value: unknown): BuilderFieldType {
  if (value === 'string' || value === 'number' || value === 'boolean') return value;
  return 'unknown';
}

function parseOperator(value: unknown): BuilderOperator {
  const allowed: BuilderOperator[] = [
    '==', '!=', '>', '>=', '<', '<=',
    'contains', 'startsWith', 'endsWith', 'matches', 'in', 'not in',
  ];
  return allowed.includes(value as BuilderOperator) ? (value as BuilderOperator) : '==';
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

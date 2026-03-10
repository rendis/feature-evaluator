import { inferSchema } from '@jsonhero/schema-infer';

import type {
  ExternalApi,
  ExternalApiExpressionVariable,
  ExternalApiParam,
  ExternalApiParamLocation,
  ExternalApiParamType,
  ExternalApiRequestConfig,
  ExternalApiResponseValidation,
  ExternalApiTestResponse,
  ExternalApiURLParamKind,
  ExternalApiValidationMode,
} from '@/api/types';

const TEMPLATE_PARAM_REGEX = /\{\{\s*([^}]+?)\s*\}\}/g;
const URL_VARIABLE_LOCATIONS = new Set<DraftVariableLocation>(['url_domain', 'url_path', 'url_query']);

let visualNodeCounter = 0;
let draftHeaderCounter = 0;

export type VisualNodeType = 'string' | 'number' | 'boolean' | 'object' | 'array';
export type BodyRootType = 'object' | 'array';
export type DraftBodyMode = 'json' | 'visual';
export type DraftHTTPCodeMode = '2xx' | 'custom';
export type DraftVariableType = 'string' | 'number' | 'boolean' | 'any';
export type DraftVariableOrigin = 'detected' | 'manual';
export type DraftVariableLocation =
  | 'url_domain'
  | 'url_path'
  | 'url_query'
  | 'header_key'
  | 'header_value'
  | 'body';

export interface VisualNode {
  id: string;
  key: string;
  type: VisualNodeType;
  value: string;
  children: VisualNode[];
}

export interface DraftHeader {
  id: string;
  key: string;
  value: string;
}

export interface DraftVariableConfig {
  origin: DraftVariableOrigin;
  type: DraftVariableType;
  required: boolean;
  locations: Set<DraftVariableLocation>;
}

export interface ExternalApiInputDescriptor {
  name: string;
  type: ExternalApiParamType;
  required: boolean;
  origin: DraftVariableOrigin;
}

export interface DraftTestState {
  inputs: Record<string, string>;
  result: ExternalApiTestResponse | null;
}

export interface ExternalApiDraft {
  name: string;
  active: boolean;
  method: ExternalApiRequestConfig['method'];
  url: string;
  headers: DraftHeader[];
  bodyMode: DraftBodyMode;
  bodyRootType: BodyRootType;
  bodyRaw: string;
  bodyVisual: VisualNode[];
  validationMode: ExternalApiValidationMode;
  httpCodeMode: DraftHTTPCodeMode;
  httpCodeCustom: string;
  responseSchema: string;
  responseSchemaVisual: VisualNode[];
  responseSchemaRootType: BodyRootType;
  expression: string;
  variables: Record<string, DraftVariableConfig>;
  secretPayload: Record<string, string>;
  replaceSecret: boolean;
  testInputs: Record<string, string>;
  testResult: ExternalApiTestResponse | null;
}

interface DetectedDraftVariableUsage {
  locations: Set<DraftVariableLocation>;
}

type DraftVariableSeed = Record<string, DraftVariableConfig>;

export interface RenderedDraftRequest {
  method: ExternalApiRequestConfig['method'];
  url: string;
  headers: Record<string, string>;
  body: unknown;
  bodyText?: string;
}

export function createEmptyExternalApi(): ExternalApi {
  return {
    id: '',
    key: '',
    name: '',
    active: true,
    request: {
      method: 'POST',
      urlTemplate: '',
      headers: [],
      bodyTemplate: {},
    },
    params: [],
    expressionVariables: [],
    responseValidation: {
      mode: 'both',
      http: { mode: 'any_2xx', codes: [] },
      body: { expression: 'response.body != null', schema: {}, sampleResponseText: '' },
    },
    hasSecrets: false,
    version: 0,
    createdAt: '',
    updatedAt: '',
    createdBy: '',
    updatedBy: '',
  };
}

export function createExternalApiDraft(source?: ExternalApi): ExternalApiDraft {
  const externalApi = source ?? createEmptyExternalApi();
  const bodyState = buildBodyEditorState(externalApi.request.bodyTemplate);
  const initialResponseSample = buildInitialResponseSample(
    externalApi.responseValidation.body.sampleResponseText,
    externalApi.responseValidation.body.schema,
  );
  const responseState = buildVisualState(initialResponseSample);

  const draft: ExternalApiDraft = {
    name: externalApi.name,
    active: externalApi.active,
    method: externalApi.request.method,
    url: externalApi.request.urlTemplate,
    headers: externalApi.request.headers.map((header) => createDraftHeader(header)),
    bodyMode: 'json',
    bodyRootType: bodyState.rootType,
    bodyRaw: bodyState.bodyRaw,
    bodyVisual: bodyState.bodyVisual,
    validationMode: externalApi.responseValidation.mode,
    httpCodeMode:
      externalApi.responseValidation.http.mode === 'status_codes' ? 'custom' : '2xx',
    httpCodeCustom: (externalApi.responseValidation.http.codes ?? []).join(', '),
    responseSchema: initialResponseSample,
    responseSchemaVisual: responseState.nodes,
    responseSchemaRootType: responseState.rootType,
    expression: externalApi.responseValidation.body.expression ?? '',
    variables: {},
    secretPayload: {},
    replaceSecret: externalApi.hasSecrets ? false : true,
    testInputs: {},
    testResult: null,
  };

  const seededVariables = seedDraftVariables(
    externalApi.params,
    externalApi.expressionVariables ?? [],
  );
  return {
    ...draft,
    variables: detectDraftVariables(draft, seededVariables),
  };
}

export function buildBodyEditorState(bodyTemplate: unknown): {
  bodyRaw: string;
  bodyVisual: VisualNode[];
  rootType: BodyRootType;
} {
  const normalized = normalizeBodyTemplate(bodyTemplate);
  const rootType: BodyRootType = Array.isArray(normalized) ? 'array' : 'object';

  return {
    bodyRaw: JSON.stringify(normalized, null, 2),
    bodyVisual: jsonToVisualNodes(normalized),
    rootType,
  };
}

export function normalizeBodyTemplate(bodyTemplate: unknown): Record<string, unknown> | unknown[] {
  if (Array.isArray(bodyTemplate)) {
    return bodyTemplate;
  }
  if (bodyTemplate && typeof bodyTemplate === 'object') {
    return bodyTemplate as Record<string, unknown>;
  }
  return {};
}

export function buildResponseSchemaState(validation: ExternalApiResponseValidation): string {
  return buildInitialResponseSample(
    validation.body.sampleResponseText,
    validation.body.schema ?? {},
  );
}

export function inferJsonSchema(value: unknown): Record<string, unknown> {
  return inferSchema(value).toJSONSchema() as Record<string, unknown>;
}

export function inferJsonSchemaFromText(text: string): {
  schema: Record<string, unknown> | null;
  parsed: unknown;
} {
  const parsed = JSON.parse(text) as unknown;
  return {
    schema: inferJsonSchema(parsed),
    parsed,
  };
}

export function jsonSchemaToExample(schema: unknown): unknown {
  if (!schema || typeof schema !== 'object') {
    return {};
  }

  const node = schema as Record<string, unknown>;
  const type = Array.isArray(node.type)
    ? node.type.find((value): value is string => typeof value === 'string')
    : node.type;

  if (type === 'object' || (!type && node.properties)) {
    const properties = (node.properties as Record<string, unknown> | undefined) ?? {};
    return Object.fromEntries(
      Object.entries(properties).map(([key, child]) => [key, jsonSchemaToExample(child)]),
    );
  }

  if (type === 'array' || (!type && node.items)) {
    if (!node.items) {
      return [];
    }
    return [jsonSchemaToExample(node.items)];
  }

  if (type === 'number' || type === 'integer') {
    return 0;
  }

  if (type === 'boolean') {
    return false;
  }

  if (type === 'null') {
    return null;
  }

  return '';
}

export function buildParamCatalog(
  request: ExternalApiRequestConfig,
  previous: ExternalApiParam[],
): ExternalApiParam[] {
  const usages = new Map<string, DetectedDraftVariableUsage>();
  const previousMap = new Map(
    previous.map((param) => [
      param.name,
      {
        type: normalizeDraftVariableType(param.type),
        required: param.required,
      },
    ]),
  );

  for (const match of extractURLTemplateMatches(request.urlTemplate)) {
    recordDraftUsage(usages, match.name, toDraftURLLocation(match.urlKind));
  }

  for (const header of request.headers) {
    for (const placeholder of extractPlaceholders(header.keyTemplate)) {
      recordDraftUsage(usages, placeholder, 'header_key');
    }
    for (const placeholder of extractPlaceholders(header.valueTemplate)) {
      recordDraftUsage(usages, placeholder, 'header_value');
    }
  }

  walkBodyTemplatePlaceholders(request.bodyTemplate, (placeholder) =>
    recordDraftUsage(usages, placeholder, 'body'),
  );

  return [...usages.entries()]
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([name, usage]) => {
      const existing = previousMap.get(name);
      const urlKind = getDraftVariableURLKind(usage.locations);
      const required =
        urlKind === 'domain' || urlKind === 'path' ? true : (existing?.required ?? false);

      return {
        name,
        type: toExternalApiParamType(existing?.type ?? 'any'),
        required,
        locations: collapseDraftLocations(usage.locations),
        urlKind,
      };
    });
}

export function collectSecretKeys(request: ExternalApiRequestConfig): string[] {
  const keys = new Set<string>();
  const collect = (placeholder: string) => {
    if (!placeholder.startsWith('secret.')) {
      return;
    }
    const key = placeholder.slice('secret.'.length).trim();
    if (key) {
      keys.add(key);
    }
  };

  for (const placeholder of extractPlaceholders(request.urlTemplate)) {
    collect(placeholder);
  }
  for (const header of request.headers) {
    for (const placeholder of extractPlaceholders(header.keyTemplate)) {
      collect(placeholder);
    }
    for (const placeholder of extractPlaceholders(header.valueTemplate)) {
      collect(placeholder);
    }
  }
  walkBodyTemplatePlaceholders(request.bodyTemplate, collect);

  return [...keys].sort();
}

export function collectSecretKeysFromDraft(draft: ExternalApiDraft): string[] {
  return collectSecretKeys(buildRequestConfigFromDraft(draft));
}

export function buildAvailableTokens(
  responseSchemaRaw: string,
  variables: Record<string, DraftVariableConfig>,
): string[] {
  const tokens = new Set<string>([
    'response',
    'response.status',
    'response.header',
    'response.body',
    'contains(',
    'startsWith(',
    'endsWith(',
    'now(',
    'dateBefore(',
    'dateAfter(',
    'true',
    'false',
    'and',
    'or',
    'not',
  ]);

  const parsed = parseJson(responseSchemaRaw);
  if (parsed !== undefined) {
    walkResponseExampleTokens(parsed, 'response.body', tokens);
  }

  for (const name of Object.keys(variables)) {
    tokens.add(`vars.${name}`);
  }

  return [...tokens];
}

export function schemaToExpressionTokens(schema: Record<string, unknown> | undefined): string[] {
  if (!schema || Object.keys(schema).length === 0) {
    return buildAvailableTokens('', {});
  }

  return buildAvailableTokens(JSON.stringify(jsonSchemaToExample(schema), null, 2), {});
}

export function parseParamInputValue(rawValue: string, type: ExternalApiParamType | DraftVariableType): unknown {
  const normalizedType = normalizeDraftVariableType(type);
  const trimmedValue = rawValue.trim();

  if (normalizedType === 'number') {
    if (trimmedValue === '') return undefined;
    const value = Number(rawValue);
    return Number.isNaN(value) ? undefined : value;
  }
  if (normalizedType === 'boolean') {
    if (rawValue === '') return undefined;
    return rawValue === 'true';
  }
  if (normalizedType === 'any') {
    if (trimmedValue === '') return undefined;

    const looksLikeStructuredJson =
      trimmedValue.startsWith('{') ||
      trimmedValue.startsWith('[') ||
      trimmedValue === 'true' ||
      trimmedValue === 'false' ||
      trimmedValue === 'null' ||
      (trimmedValue.startsWith('"') && trimmedValue.endsWith('"'));

    if (!looksLikeStructuredJson) {
      return rawValue;
    }

    try {
      return JSON.parse(rawValue) as unknown;
    } catch {
      return rawValue;
    }
  }
  if (rawValue === '') return undefined;
  return rawValue;
}

export function jsonToVisualNodes(data: unknown, prefix = 'root'): VisualNode[] {
  if (Array.isArray(data)) {
    return data.map((value, index) => createVisualNode(value, '', `${prefix}-${index}`));
  }
  if (data && typeof data === 'object') {
    return Object.entries(data as Record<string, unknown>).map(([key, value], index) =>
      createVisualNode(value, key, `${prefix}-${index}`),
    );
  }
  return [];
}

export function visualNodesToJson(
  nodes: VisualNode[],
  rootType: BodyRootType,
): Record<string, unknown> | unknown[] {
  if (rootType === 'array') {
    return nodes.map((node) => visualNodeToValue(node));
  }

  const result: Record<string, unknown> = {};
  for (const node of nodes) {
    if (!node.key.trim()) {
      continue;
    }
    result[node.key.trim()] = visualNodeToValue(node);
  }
  return result;
}

export function createVisualNodeDraft(): VisualNode {
  visualNodeCounter += 1;
  return {
    id: `node-${visualNodeCounter}`,
    key: '',
    type: 'string',
    value: '',
    children: [],
  };
}

export function createDraftHeader(
  header?: Pick<ExternalApiRequestConfig['headers'][number], 'keyTemplate' | 'valueTemplate'>,
): DraftHeader {
  draftHeaderCounter += 1;
  return {
    id: `header-${draftHeaderCounter}`,
    key: header?.keyTemplate ?? '',
    value: header?.valueTemplate ?? '',
  };
}

export function extractPlaceholders(value: string): string[] {
  const result = new Set<string>();
  for (const match of value.matchAll(TEMPLATE_PARAM_REGEX)) {
    const name = match[1]?.trim();
    if (name) {
      result.add(name);
    }
  }
  return [...result];
}

export function detectDraftVariables(
  draft: Pick<
    ExternalApiDraft,
    'method' | 'url' | 'headers' | 'bodyMode' | 'bodyRaw' | 'bodyVisual' | 'variables'
  >,
  seed: DraftVariableSeed = draft.variables,
): Record<string, DraftVariableConfig> {
  const usages = new Map<string, DetectedDraftVariableUsage>();
  const manualVariables = Object.fromEntries(
    Object.entries(seed).filter(([, config]) => config.origin === 'manual'),
  );

  for (const match of extractURLTemplateMatches(draft.url)) {
    recordDraftUsage(usages, match.name, toDraftURLLocation(match.urlKind));
  }

  for (const header of draft.headers) {
    for (const placeholder of extractPlaceholders(header.key)) {
      recordDraftUsage(usages, placeholder, 'header_key');
    }
    for (const placeholder of extractPlaceholders(header.value)) {
      recordDraftUsage(usages, placeholder, 'header_value');
    }
  }

  if (['POST', 'PUT', 'PATCH'].includes(draft.method)) {
    if (draft.bodyMode === 'json') {
      for (const placeholder of extractPlaceholders(draft.bodyRaw)) {
        recordDraftUsage(usages, placeholder, 'body');
      }
    } else {
      walkVisualNodesPlaceholders(draft.bodyVisual, (placeholder) =>
        recordDraftUsage(usages, placeholder, 'body'),
      );
    }
  }

  return [...usages.entries()]
    .sort(([left], [right]) => left.localeCompare(right))
    .reduce<Record<string, DraftVariableConfig>>((accumulator, [name, usage]) => {
      const existing = seed[name];
      const locations = new Set(usage.locations);
      const required =
        hasRequiredURLLocation(locations) ? true : (existing?.required ?? false);

      accumulator[name] = {
        origin: 'detected',
        type: existing?.type ?? 'any',
        required,
        locations,
      };
      return accumulator;
    }, manualVariables);
}

export function areDraftVariablesEqual(
  left: Record<string, DraftVariableConfig>,
  right: Record<string, DraftVariableConfig>,
): boolean {
  const leftKeys = Object.keys(left).sort();
  const rightKeys = Object.keys(right).sort();

  if (leftKeys.length !== rightKeys.length) {
    return false;
  }

  for (let index = 0; index < leftKeys.length; index += 1) {
    if (leftKeys[index] !== rightKeys[index]) {
      return false;
    }

    const leftValue = left[leftKeys[index]];
    const rightValue = right[rightKeys[index]];
    if (!leftValue || !rightValue) {
      return false;
    }

    if (
      leftValue.origin !== rightValue.origin ||
      leftValue.type !== rightValue.type ||
      leftValue.required !== rightValue.required
    ) {
      return false;
    }

    const leftLocations = [...leftValue.locations].sort();
    const rightLocations = [...rightValue.locations].sort();
    if (leftLocations.length !== rightLocations.length) {
      return false;
    }

    for (let locationIndex = 0; locationIndex < leftLocations.length; locationIndex += 1) {
      if (leftLocations[locationIndex] !== rightLocations[locationIndex]) {
        return false;
      }
    }
  }

  return true;
}

export function cloneDraftVariables(
  variables: Record<string, DraftVariableConfig>,
): Record<string, DraftVariableConfig> {
  return Object.fromEntries(
    Object.entries(variables).map(([name, config]) => [
      name,
      {
        ...config,
        locations: new Set(config.locations),
      },
    ]),
  );
}

export function draftVariablesToParams(
  variables: Record<string, DraftVariableConfig>,
): ExternalApiParam[] {
  return Object.entries(variables)
    .filter(([, config]) => config.origin === 'detected')
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([name, config]) => {
      const urlKind = getDraftVariableURLKind(config.locations);
      return {
        name,
        type: toExternalApiParamType(config.type),
        required: hasRequiredURLLocation(config.locations) ? true : config.required,
        locations: collapseDraftLocations(config.locations),
        urlKind,
      };
    });
}

export function draftVariablesToExpressionVariables(
  variables: Record<string, DraftVariableConfig>,
): ExternalApiExpressionVariable[] {
  return Object.entries(variables)
    .filter(([, config]) => config.origin === 'manual')
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([name, config]) => ({
      name,
      type: toExternalApiParamType(config.type),
      required: config.required,
    }));
}

export function draftVariablesToInputDescriptors(
  variables: Record<string, DraftVariableConfig>,
): ExternalApiInputDescriptor[] {
  return Object.entries(variables)
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([name, config]) => ({
      name,
      type: toExternalApiParamType(config.type),
      required: hasRequiredURLLocation(config.locations) ? true : config.required,
      origin: config.origin,
    }));
}

export function buildRequestConfigFromDraft(draft: ExternalApiDraft): ExternalApiRequestConfig {
  return {
    method: draft.method,
    urlTemplate: draft.url,
    headers: draft.headers
      .filter((header) => header.key.trim().length > 0 || header.value.trim().length > 0)
      .map((header) => ({
        keyTemplate: header.key,
        valueTemplate: header.value,
      })),
    bodyTemplate:
      draft.method === 'GET' || draft.method === 'DELETE' ? null : buildDraftBodyTemplate(draft),
  };
}

export function buildResponseValidationFromDraft(
  draft: ExternalApiDraft,
): ExternalApiResponseValidation {
  const parsedResponse = parseJson(draft.responseSchema);
  const normalizedSample =
    parsedResponse === undefined
      ? draft.responseSchema.trim()
      : JSON.stringify(parsedResponse, null, 2);

  return {
    mode: draft.validationMode,
    http: {
      mode: draft.httpCodeMode === 'custom' ? 'status_codes' : 'any_2xx',
      codes:
        draft.httpCodeMode === 'custom'
          ? draft.httpCodeCustom
              .split(',')
              .map((value) => Number(value.trim()))
              .filter((value) => !Number.isNaN(value))
          : [],
    },
    body: {
      expression: draft.expression,
      schema:
        parsedResponse === undefined ? {} : inferJsonSchema(parsedResponse),
      sampleResponseText: normalizedSample,
    },
  };
}

export function buildExternalApiPartsFromDraft(draft: ExternalApiDraft): {
  request: ExternalApiRequestConfig;
  params: ExternalApiParam[];
  expressionVariables: ExternalApiExpressionVariable[];
  responseValidation: ExternalApiResponseValidation;
} {
  return {
    request: buildRequestConfigFromDraft(draft),
    params: draftVariablesToParams(draft.variables),
    expressionVariables: draftVariablesToExpressionVariables(draft.variables),
    responseValidation: buildResponseValidationFromDraft(draft),
  };
}

export function renderDraftRequest(
  draft: ExternalApiDraft,
  testInputs: Record<string, string>,
): RenderedDraftRequest {
  const replaceValue = (source: string) =>
    source.replace(TEMPLATE_PARAM_REGEX, (_match, rawName: string) => {
      const name = rawName.trim();
      if (name.startsWith('secret.')) {
        const key = name.slice('secret.'.length);
        return draft.secretPayload[key]?.trim() ? '••••••' : `{{${name}}}`;
      }
      return testInputs[name] ?? '';
    });

  const bodySource =
    draft.method === 'GET' || draft.method === 'DELETE'
      ? undefined
      : draft.bodyMode === 'visual'
        ? JSON.stringify(visualNodesToJson(draft.bodyVisual, draft.bodyRootType), null, 2)
        : draft.bodyRaw;

  const bodyText = bodySource == null ? undefined : replaceValue(bodySource);
  const parsedBody = bodyText == null ? undefined : parseJson(bodyText);

  return {
    method: draft.method,
    url: replaceValue(draft.url),
    headers: Object.fromEntries(
      draft.headers
        .filter((header) => header.key.trim().length > 0)
        .map((header) => [replaceValue(header.key), replaceValue(header.value)]),
    ),
    body: parsedBody === undefined ? bodyText : parsedBody,
    bodyText,
  };
}

function buildDraftBodyTemplate(draft: ExternalApiDraft): Record<string, unknown> | unknown[] {
  if (draft.bodyMode === 'visual') {
    return visualNodesToJson(draft.bodyVisual, draft.bodyRootType);
  }

  const parsed = parseJson(draft.bodyRaw);
  if (Array.isArray(parsed)) {
    return parsed;
  }
  if (parsed && typeof parsed === 'object') {
    return parsed as Record<string, unknown>;
  }
  return {};
}

function buildInitialResponseSample(
  sampleResponseText: string | undefined,
  schema: Record<string, unknown> | undefined,
): string {
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

function buildVisualState(rawValue: string): {
  nodes: VisualNode[];
  rootType: BodyRootType;
} {
  const parsed = parseJson(rawValue);
  if (parsed == null || typeof parsed !== 'object') {
    return {
      nodes: [],
      rootType: 'object',
    };
  }

  const nextState = buildBodyEditorState(parsed);
  return {
    nodes: nextState.bodyVisual,
    rootType: nextState.rootType,
  };
}

function seedDraftVariables(
  params: ExternalApiParam[],
  expressionVariables: ExternalApiExpressionVariable[],
): DraftVariableSeed {
  const seeded: DraftVariableSeed = {};

  for (const variable of expressionVariables) {
    seeded[variable.name] = {
      origin: 'manual',
      type: normalizeDraftVariableType(variable.type),
      required: variable.required,
      locations: new Set<DraftVariableLocation>(),
    };
  }

  for (const param of params) {
    const locations = new Set<DraftVariableLocation>();
    for (const location of param.locations) {
      if (location === 'url') {
        locations.add(toDraftURLLocation(param.urlKind ?? 'path'));
        continue;
      }
      if (location === 'header') {
        locations.add('header_value');
        continue;
      }
      if (location === 'body') {
        locations.add('body');
      }
    }

    seeded[param.name] = {
      origin: 'detected',
      type: normalizeDraftVariableType(param.type),
      required: param.required,
      locations,
    };
  }

  return seeded;
}

function walkResponseExampleTokens(value: unknown, prefix: string, tokens: Set<string>) {
  tokens.add(prefix);

  if (Array.isArray(value)) {
    if (value.length > 0) {
      walkResponseExampleTokens(value[0], prefix, tokens);
    }
    return;
  }

  if (!value || typeof value !== 'object') {
    return;
  }

  for (const [key, child] of Object.entries(value as Record<string, unknown>)) {
    const next = `${prefix}.${key}`;
    tokens.add(next);
    walkResponseExampleTokens(child, next, tokens);
  }
}

function walkVisualNodesPlaceholders(
  nodes: VisualNode[],
  visit: (placeholder: string) => void,
) {
  for (const node of nodes) {
    if (node.key.trim()) {
      for (const placeholder of extractPlaceholders(node.key)) {
        visit(placeholder);
      }
    }
    for (const placeholder of extractPlaceholders(String(node.value ?? ''))) {
      visit(placeholder);
    }
    if (node.children.length > 0) {
      walkVisualNodesPlaceholders(node.children, visit);
    }
  }
}

function recordDraftUsage(
  usages: Map<string, DetectedDraftVariableUsage>,
  name: string,
  location: DraftVariableLocation,
) {
  if (!name || name.startsWith('secret.')) {
    return;
  }

  const usage = usages.get(name) ?? { locations: new Set<DraftVariableLocation>() };
  usage.locations.add(location);
  usages.set(name, usage);
}

function hasRequiredURLLocation(locations: Set<DraftVariableLocation>): boolean {
  return locations.has('url_domain') || locations.has('url_path');
}

function collapseDraftLocations(
  locations: Set<DraftVariableLocation>,
): ExternalApiParamLocation[] {
  const collapsed = new Set<ExternalApiParamLocation>();

  for (const location of locations) {
    if (URL_VARIABLE_LOCATIONS.has(location)) {
      collapsed.add('url');
      continue;
    }
    if (location === 'header_key' || location === 'header_value') {
      collapsed.add('header');
      continue;
    }
    if (location === 'body') {
      collapsed.add('body');
    }
  }

  return [...collapsed].sort();
}

function getDraftVariableURLKind(
  locations: Set<DraftVariableLocation>,
): ExternalApiURLParamKind | undefined {
  if (locations.has('url_domain')) {
    return 'domain';
  }
  if (locations.has('url_path')) {
    return 'path';
  }
  if (locations.has('url_query')) {
    return 'query';
  }
  return undefined;
}

function toDraftURLLocation(kind: ExternalApiURLParamKind): DraftVariableLocation {
  if (kind === 'domain') {
    return 'url_domain';
  }
  if (kind === 'query') {
    return 'url_query';
  }
  return 'url_path';
}

function normalizeDraftVariableType(
  type: ExternalApiParamType | DraftVariableType,
): DraftVariableType {
  if (type === 'bool') {
    return 'boolean';
  }
  return type;
}

function toExternalApiParamType(type: DraftVariableType): ExternalApiParamType {
  return type === 'boolean' ? 'bool' : type;
}

function createVisualNode(value: unknown, key: string, id: string): VisualNode {
  if (Array.isArray(value)) {
    return {
      id,
      key,
      type: 'array',
      value: '',
      children: jsonToVisualNodes(value, id),
    };
  }
  if (value && typeof value === 'object') {
    return {
      id,
      key,
      type: 'object',
      value: '',
      children: jsonToVisualNodes(value, id),
    };
  }
  if (typeof value === 'number') {
    return { id, key, type: 'number', value: String(value), children: [] };
  }
  if (typeof value === 'boolean') {
    return { id, key, type: 'boolean', value: value ? 'true' : 'false', children: [] };
  }
  return { id, key, type: 'string', value: String(value ?? ''), children: [] };
}

function visualNodeToValue(node: VisualNode): unknown {
  if (node.type === 'object') {
    return visualNodesToJson(node.children, 'object');
  }
  if (node.type === 'array') {
    return visualNodesToJson(node.children, 'array');
  }
  if (node.type === 'number') {
    return node.value === '' ? 0 : Number(node.value);
  }
  if (node.type === 'boolean') {
    return node.value === 'true';
  }
  return node.value;
}

function walkBodyTemplatePlaceholders(value: unknown, visit: (placeholder: string) => void) {
  if (typeof value === 'string') {
    for (const placeholder of extractPlaceholders(value)) {
      visit(placeholder);
    }
    return;
  }
  if (Array.isArray(value)) {
    for (const item of value) {
      walkBodyTemplatePlaceholders(item, visit);
    }
    return;
  }
  if (value && typeof value === 'object') {
    for (const [key, child] of Object.entries(value as Record<string, unknown>)) {
      for (const placeholder of extractPlaceholders(key)) {
        visit(placeholder);
      }
      walkBodyTemplatePlaceholders(child, visit);
    }
  }
}

function extractURLTemplateMatches(
  urlTemplate: string,
): { name: string; urlKind: ExternalApiURLParamKind }[] {
  const matches: { name: string; urlKind: ExternalApiURLParamKind }[] = [];
  if (!urlTemplate) {
    return matches;
  }

  const queryIndex = urlTemplate.indexOf('?');
  const schemeIndex = urlTemplate.indexOf('://');
  const hostStart = schemeIndex >= 0 ? schemeIndex + 3 : 0;
  const pathIndexRelative = urlTemplate.slice(hostStart).indexOf('/');
  const pathIndex = pathIndexRelative >= 0 ? hostStart + pathIndexRelative : urlTemplate.length;

  for (const match of urlTemplate.matchAll(TEMPLATE_PARAM_REGEX)) {
    const name = match[1]?.trim();
    if (!name) {
      continue;
    }
    const index = match.index ?? 0;
    let urlKind: ExternalApiURLParamKind = 'path';
    if (queryIndex >= 0 && index > queryIndex) {
      urlKind = 'query';
    } else if (index < pathIndex) {
      urlKind = 'domain';
    }
    matches.push({ name, urlKind });
  }

  return matches;
}

function parseJson(value: string): unknown | undefined {
  try {
    return JSON.parse(value) as unknown;
  } catch {
    return undefined;
  }
}

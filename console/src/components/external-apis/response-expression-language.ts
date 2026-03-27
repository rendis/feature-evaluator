
import { jsonSchemaToExample } from './external-api-builder-utils';

import type { DraftVariableConfig } from './external-api-builder-utils';
import type {
  ExternalApiExpressionAction,
  ExternalApiExpressionProfile,
} from '@/api/types';

export type ResponseExpressionValueType =
  | 'object'
  | 'array'
  | 'string'
  | 'number'
  | 'boolean'
  | 'null'
  | 'unknown';

type CompletionKind = 'start' | 'member' | 'predicate' | 'rhs' | 'logic';
type StartBoundary = 'empty' | 'group' | 'logical' | 'after-not';

interface SymbolMember {
  label: string;
  insertText: string;
  path: string;
  type: ResponseExpressionValueType;
  boost?: number;
}

interface SymbolNode {
  path: string;
  type: ResponseExpressionValueType;
  description?: string;
  members: SymbolMember[];
}

interface ParsedCompletionContext {
  kind: CompletionKind;
  node?: SymbolNode;
  path?: string;
  operator?: string;
  allowUnaryNot?: boolean;
  filterFrom: number;
  filterText: string;
  applyFrom: number;
  applyTo: number;
}

export interface ResponseExpressionCatalog {
  profile: ExternalApiExpressionProfile;
  symbols: Map<string, SymbolNode>;
}

export interface ResponseExpressionCompletion {
  label: string;
  insertText: string;
  detail?: string;
  category: string;
  type: 'keyword' | 'variable' | 'property' | 'function' | 'text';
  boost?: number;
  selectionOffset?: number;
  applyFrom: number;
  applyTo: number;
}

const IDENTIFIER_PATTERN = /^[A-Za-z_][A-Za-z0-9_]*$/;
const PATH_PATTERN = String.raw`[A-Za-z_][A-Za-z0-9_]*(?:\[[0-9]+\]|\[(?:"[^"]*"|'[^']*')\]|\.[A-Za-z_][A-Za-z0-9_]*)*`;
const COMPLETE_CLAUSE_REGEX = new RegExp(
  `(?:len\\(${PATH_PATTERN}\\)|${PATH_PATTERN})\\s*(==|!=|>=|<=|>|<|contains|startsWith|endsWith|matches)\\s*(?:true|false|null|nil|-?\\d+(?:\\.\\d+)?|"(?:[^"\\\\]|\\\\.)*"|'(?:[^'\\\\]|\\\\.)*')\\s*$`,
);

export const DEFAULT_EXTERNAL_API_EXPRESSION_PROFILE: ExternalApiExpressionProfile = {
  keywords: ['and', 'or', 'not', 'true', 'false', 'null', 'nil'],
  symbols: [
    {
      path: 'response',
      type: 'object',
      description: 'Response envelope. Use response.status, response.header and response.body.',
    },
    {
      path: 'response.status',
      type: 'number',
      description: 'HTTP status code returned by the upstream service.',
    },
    {
      path: 'response.header',
      type: 'object',
      description: 'Normalized response headers. Access with response.header["x-header"].',
    },
    {
      path: 'response.body',
      type: 'unknown',
      description: 'Parsed response body. Members depend on the configured response schema/sample.',
    },
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
      id: 'string-eq-empty',
      label: '== ""',
      category: 'comparison',
      appliesTo: ['string'],
      template: '{{path}} == ""',
      priority: 100,
    },
    {
      id: 'string-ne-empty',
      label: '!= ""',
      category: 'comparison',
      appliesTo: ['string'],
      template: '{{path}} != ""',
      priority: 99,
    },
    {
      id: 'string-contains',
      label: 'contains ""',
      detail: 'Expr infix string containment operator',
      category: 'string-op',
      appliesTo: ['string'],
      template: '{{path}} contains ""',
      priority: 96,
    },
    {
      id: 'string-starts-with',
      label: 'startsWith ""',
      detail: 'Expr infix prefix operator',
      category: 'string-op',
      appliesTo: ['string'],
      template: '{{path}} startsWith ""',
      priority: 95,
    },
    {
      id: 'string-ends-with',
      label: 'endsWith ""',
      detail: 'Expr infix suffix operator',
      category: 'string-op',
      appliesTo: ['string'],
      template: '{{path}} endsWith ""',
      priority: 94,
    },
    {
      id: 'string-matches',
      label: 'matches ""',
      detail: 'Expr infix regex operator',
      category: 'string-op',
      appliesTo: ['string'],
      template: '{{path}} matches ""',
      priority: 93,
    },
    {
      id: 'number-eq',
      label: '== 0',
      category: 'comparison',
      appliesTo: ['number'],
      template: '{{path}} == 0',
      priority: 100,
    },
    {
      id: 'number-ne',
      label: '!= 0',
      category: 'comparison',
      appliesTo: ['number'],
      template: '{{path}} != 0',
      priority: 99,
    },
    {
      id: 'number-gt',
      label: '> 0',
      category: 'comparison',
      appliesTo: ['number'],
      template: '{{path}} > 0',
      priority: 98,
    },
    {
      id: 'number-gte',
      label: '>= 0',
      category: 'comparison',
      appliesTo: ['number'],
      template: '{{path}} >= 0',
      priority: 97,
    },
    {
      id: 'number-lt',
      label: '< 0',
      category: 'comparison',
      appliesTo: ['number'],
      template: '{{path}} < 0',
      priority: 96,
    },
    {
      id: 'number-lte',
      label: '<= 0',
      category: 'comparison',
      appliesTo: ['number'],
      template: '{{path}} <= 0',
      priority: 95,
    },
    {
      id: 'array-len',
      label: 'len(...) > 0',
      detail: 'Preferred Expr length check for arrays',
      category: 'array-op',
      appliesTo: ['array'],
      template: 'len({{path}}) > 0',
      priority: 100,
    },
    {
      id: 'array-index',
      label: '[0]',
      category: 'navigation',
      appliesTo: ['array'],
      template: '{{path}}[0]',
      priority: 95,
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
};

export function buildResponseExpressionCatalog({
  profile = DEFAULT_EXTERNAL_API_EXPRESSION_PROFILE,
  sampleResponseText,
  schema,
  variables = {},
  responseHeaderKeys = [],
}: {
  profile?: ExternalApiExpressionProfile;
  sampleResponseText?: string;
  schema?: Record<string, unknown>;
  variables?: Record<string, DraftVariableConfig>;
  responseHeaderKeys?: string[];
}): ResponseExpressionCatalog {
  const symbols = new Map<string, SymbolNode>();

  registerNode(
    symbols,
    'response',
    'object',
    'Response envelope. Use response.status, response.header and response.body.',
  );
  registerNode(symbols, 'response.status', 'number', 'HTTP status code returned by the upstream service.');
  registerNode(
    symbols,
    'response.header',
    'object',
    'Normalized response headers. Access with response.header["x-header"].',
  );
  registerNode(
    symbols,
    'response.body',
    'unknown',
    'Parsed response body. Members depend on the configured response schema/sample.',
  );
  addMember(symbols, 'response', {
    label: 'status',
    insertText: 'status',
    path: 'response.status',
    type: 'number',
    boost: 260,
  });
  addMember(symbols, 'response', {
    label: 'header',
    insertText: 'header',
    path: 'response.header',
    type: 'object',
    boost: 255,
  });
  addMember(symbols, 'response', {
    label: 'body',
    insertText: 'body',
    path: 'response.body',
    type: 'unknown',
    boost: 250,
  });

  for (const symbol of profile.symbols) {
    registerNode(symbols, symbol.path, toValueType(symbol.type), symbol.description);
  }

  if (Object.keys(variables).length > 0) {
    registerNode(symbols, 'vars', 'object', 'Request variables available to this external API.');
    for (const variableName of Object.keys(variables).sort((left, right) => left.localeCompare(right))) {
      const variablePath = `vars.${variableName}`;
      addMember(symbols, 'vars', {
        label: variableName,
        insertText: variableName,
        path: variablePath,
        type: toExpressionVariableType(variables[variableName]?.type),
        boost: 260,
      });
      registerNode(symbols, variablePath, toExpressionVariableType(variables[variableName]?.type));
    }
  }

  const responseValue = resolveResponseSample(sampleResponseText, schema);
  if (responseValue !== undefined) {
    const responseBodyType = inferValueType(responseValue);
    registerNode(symbols, 'response.body', responseBodyType);
    updateMemberType(symbols, 'response', 'body', responseBodyType);
    populateResponseValue(symbols, 'response.body', responseValue);
  }

  addMember(symbols, 'response.header', {
    label: '["header-name"]',
    insertText: '["header-name"]',
    path: 'response.header["header-name"]',
    type: 'string',
    boost: 200,
  });
  registerNode(symbols, 'response.header["header-name"]', 'string');

  for (const headerKey of [...new Set(responseHeaderKeys)].sort((left, right) =>
    left.localeCompare(right),
  )) {
    const label = `[${JSON.stringify(headerKey)}]`;
    const path = `response.header${label}`;
    addMember(symbols, 'response.header', {
      label,
      insertText: label,
      path,
      type: 'string',
      boost: 300,
    });
    registerNode(symbols, path, 'string');
  }

  return { profile, symbols };
}

export function getResponseExpressionCompletions({
  doc,
  cursor,
  catalog,
  explicit,
}: {
  doc: string;
  cursor: number;
  catalog: ResponseExpressionCatalog;
  explicit: boolean;
}): { from: number; options: ResponseExpressionCompletion[] } | null {
  const parsedContext = parseCompletionContext(doc, cursor, catalog, explicit);
  if (!parsedContext) {
    return null;
  }

  const options = buildOptionsForContext(parsedContext, catalog);
  if (options.length === 0) {
    return null;
  }

  return {
    from: parsedContext.filterFrom,
    options,
  };
}

export function shouldAutoTriggerCompletion(_previousText: string, nextText: string, cursor: number): boolean {
  if (cursor < 0 || cursor > nextText.length) {
    return false;
  }

  if (nextText.charAt(cursor - 1) === '.') {
    return true;
  }

  const left = nextText.slice(0, cursor);
  return (
    /(==|!=|>=|<=|>|<|contains|startsWith|endsWith|matches)\s*$/.test(left) ||
    /\(\s*$/.test(left) ||
    /(?:^|[\s(])(?:and|or|not)\s*$/.test(left)
  );
}

function buildOptionsForContext(
  parsedContext: ParsedCompletionContext,
  catalog: ResponseExpressionCatalog,
): ResponseExpressionCompletion[] {
  switch (parsedContext.kind) {
    case 'member':
      return buildMemberOptions(parsedContext, catalog);
    case 'predicate':
      return buildPredicateOptions(parsedContext, catalog);
    case 'rhs':
      return buildRHSOptions(parsedContext);
    case 'logic':
      return buildLogicOptions(parsedContext, catalog);
    case 'start':
      return buildStartOptions(parsedContext, catalog);
    default:
      return [];
  }
}

function buildStartOptions(
  parsedContext: ParsedCompletionContext,
  catalog: ResponseExpressionCatalog,
): ResponseExpressionCompletion[] {
  const symbolOptions = buildStartSymbolOptions(parsedContext, catalog);
  if (!parsedContext.allowUnaryNot) {
    return symbolOptions;
  }

  const notKeyword = catalog.profile.keywords.find((keyword) => keyword === 'not');
  if (!notKeyword) {
    return symbolOptions;
  }

  return [
    ...symbolOptions,
    {
      label: notKeyword,
      insertText: `${notKeyword} `,
      category: 'keyword',
      type: 'keyword' as const,
      boost: 180,
      applyFrom: parsedContext.applyFrom,
      applyTo: parsedContext.applyTo,
    },
  ];
}

function buildLogicOptions(
  parsedContext: ParsedCompletionContext,
  catalog: ResponseExpressionCatalog,
): ResponseExpressionCompletion[] {
  return catalog.profile.keywords
    .filter((keyword) => keyword === 'and' || keyword === 'or')
    .map((keyword, index) => ({
      label: keyword,
      insertText: ` ${keyword} `,
      category: 'keyword',
      type: 'keyword' as const,
      boost: 180 - index,
      applyFrom: parsedContext.applyFrom,
      applyTo: parsedContext.applyTo,
    }));
}

function buildMemberOptions(
  parsedContext: ParsedCompletionContext,
  catalog: ResponseExpressionCatalog,
): ResponseExpressionCompletion[] {
  if (!parsedContext.node || !parsedContext.path) {
    return [];
  }

  const { node, path } = parsedContext;

  if (node.type === 'object') {
    return node.members.map((member) => ({
      label: member.label,
      insertText: member.insertText,
      detail: member.type,
      category: member.path.startsWith('response.header[') ? 'header' : 'member',
      type: member.path.includes('[') ? 'text' : 'property',
      boost: member.boost ?? 100,
      selectionOffset: member.insertText === '["header-name"]' ? 2 : undefined,
      applyFrom: parsedContext.applyFrom,
      applyTo: parsedContext.applyTo,
    }));
  }

  return catalog.profile.actions
    .filter((action) => action.appliesTo.includes(node.type))
    .sort((left, right) => right.priority - left.priority)
    .map((action) => {
      const applied = applyActionTemplate(action, path);
      return {
        label: action.label,
        insertText: applied.text,
        detail: action.detail,
        category: action.category,
        type: action.category === 'navigation' ? 'property' : 'function',
        boost: action.priority,
        selectionOffset: applied.selectionOffset,
        applyFrom: parsedContext.applyFrom,
        applyTo: parsedContext.applyTo,
      };
    });
}

function buildPredicateOptions(
  parsedContext: ParsedCompletionContext,
  catalog: ResponseExpressionCatalog,
): ResponseExpressionCompletion[] {
  if (!parsedContext.node || !parsedContext.path) {
    return [];
  }

  if (parsedContext.node.type === 'array') {
    return catalog.profile.actions
      .filter((action) => action.appliesTo.includes('array'))
      .sort((left, right) => right.priority - left.priority)
      .map((action) => {
        const applied = applyActionTemplate(action, parsedContext.path ?? '');
        return {
          label: action.label,
          insertText: applied.text,
          detail: action.detail,
          category: action.category,
          type: action.category === 'navigation' ? 'property' : 'function',
          boost: action.priority,
          selectionOffset: applied.selectionOffset,
          applyFrom: parsedContext.path
            ? parsedContext.applyFrom - parsedContext.path.length
            : parsedContext.applyFrom,
          applyTo: parsedContext.applyTo,
        };
      });
  }

  const operators = getPredicateOperators(parsedContext.node.type);
  return operators.map((operator, index) => ({
    label: operator.label,
    insertText: operator.insertText,
    category: operator.category,
    type: operator.category === 'navigation' ? 'property' : 'keyword',
    boost: 200 - index,
    applyFrom: parsedContext.applyFrom,
    applyTo: parsedContext.applyTo,
  }));
}

function buildRHSOptions(parsedContext: ParsedCompletionContext): ResponseExpressionCompletion[] {
  if (!parsedContext.node || !parsedContext.operator) {
    return [];
  }

  return getRHSValues(parsedContext.node.type, parsedContext.operator).map((value, index) => ({
    label: value.label,
    insertText: value.insertText,
    category: 'literal',
    type: 'keyword',
    boost: 180 - index,
    selectionOffset: value.selectionOffset,
    applyFrom: parsedContext.applyFrom,
    applyTo: parsedContext.applyTo,
  }));
}

function parseCompletionContext(
  doc: string,
  cursor: number,
  catalog: ResponseExpressionCatalog,
  explicit: boolean,
): ParsedCompletionContext | null {
  const left = doc.slice(0, cursor);
  const rhsContext = parseRHSContext(left, cursor, catalog);
  if (rhsContext) {
    return rhsContext;
  }

  const memberContext = parseMemberContext(left, cursor, catalog);
  if (memberContext) {
    return memberContext;
  }

  const predicateContext = parsePredicateContext(left, cursor, catalog, explicit);
  if (predicateContext) {
    return predicateContext;
  }

  const logicContext = parseLogicContext(left, cursor, explicit);
  if (logicContext) {
    return logicContext;
  }

  return parseStartContext(left, cursor);
}

function parseMemberContext(
  left: string,
  cursor: number,
  catalog: ResponseExpressionCatalog,
): ParsedCompletionContext | null {
  const regex = new RegExp(`(${PATH_PATTERN})\\.([A-Za-z_][A-Za-z0-9_]*)?$`);
  const match = left.match(regex);
  if (!match) {
    return null;
  }

  const rawPath = normalizeExprPath(match[1] ?? '');
  const node = catalog.symbols.get(rawPath);
  if (!node) {
    return null;
  }

  const filterText = match[2] ?? '';
  const pathWithDot = match[0] ?? '';
  const pathFrom = cursor - pathWithDot.length;

  return {
    kind: 'member',
    node,
    path: rawPath,
    filterFrom: cursor - filterText.length,
    filterText,
    applyFrom: node.type === 'object' ? cursor - filterText.length : pathFrom,
    applyTo: cursor,
  };
}

function parsePredicateContext(
  left: string,
  cursor: number,
  catalog: ResponseExpressionCatalog,
  explicit: boolean,
): ParsedCompletionContext | null {
  const regex = new RegExp(`(${PATH_PATTERN})$`);
  const match = left.match(regex);
  if (!match) {
    return null;
  }

  const path = normalizeExprPath(match[1] ?? '');
  const node = catalog.symbols.get(path);
  if (!node || !explicit) {
    return null;
  }

  return {
    kind: 'predicate',
    node,
    path,
    filterFrom: cursor,
    filterText: '',
    applyFrom: cursor,
    applyTo: cursor,
  };
}

function parseRHSContext(
  left: string,
  cursor: number,
  catalog: ResponseExpressionCatalog,
): ParsedCompletionContext | null {
  const regex = new RegExp(
    `(${PATH_PATTERN})\\s*(==|!=|>=|<=|>|<|contains|startsWith|endsWith|matches)\\s*([A-Za-z0-9_"'-]*)$`,
  );
  const match = left.match(regex);
  if (!match) {
    return null;
  }

  const path = normalizeExprPath(match[1] ?? '');
  const node = catalog.symbols.get(path);
  if (!node) {
    return null;
  }

  const filterText = match[3] ?? '';
  return {
    kind: 'rhs',
    node,
    path,
    operator: match[2],
    filterFrom: cursor - filterText.length,
    filterText,
    applyFrom: cursor - filterText.length,
    applyTo: cursor,
  };
}

function parseLogicContext(left: string, cursor: number, explicit: boolean): ParsedCompletionContext | null {
  if (!explicit) {
    return null;
  }

  const rawPrefix = left.match(/([A-Za-z_]*)$/)?.[1] ?? '';
  const prefix = isLogicKeywordPrefix(rawPrefix) ? rawPrefix : '';
  const clauseText = prefix ? left.slice(0, -prefix.length) : left;
  if (!isCompleteClause(clauseText)) {
    return null;
  }

  return {
    kind: 'logic',
    filterFrom: cursor - prefix.length,
    filterText: prefix,
    applyFrom: cursor - prefix.length,
    applyTo: cursor,
  };
}

function parseStartContext(left: string, cursor: number): ParsedCompletionContext | null {
  const prefixMatch = left.match(/([A-Za-z_][A-Za-z0-9_]*)?$/);
  const prefix = prefixMatch?.[1] ?? '';
  const textBeforePrefix = prefix ? left.slice(0, -prefix.length) : left;
  const boundary = getStartBoundary(textBeforePrefix);
  if (!boundary) {
    return null;
  }

  return {
    kind: 'start',
    allowUnaryNot: prefix.length > 0 || boundary === 'group' || boundary === 'logical',
    filterFrom: cursor - prefix.length,
    filterText: prefix,
    applyFrom: cursor - prefix.length,
    applyTo: cursor,
  };
}

function getPredicateOperators(type: ResponseExpressionValueType): {
  label: string;
  insertText: string;
  category: string;
}[] {
  switch (type) {
    case 'boolean':
      return [
        { label: '==', insertText: ' == ', category: 'comparison' },
        { label: '!=', insertText: ' != ', category: 'comparison' },
      ];
    case 'string':
      return [
        { label: '==', insertText: ' == ', category: 'comparison' },
        { label: '!=', insertText: ' != ', category: 'comparison' },
        { label: 'contains', insertText: ' contains ', category: 'string-op' },
        { label: 'startsWith', insertText: ' startsWith ', category: 'string-op' },
        { label: 'endsWith', insertText: ' endsWith ', category: 'string-op' },
        { label: 'matches', insertText: ' matches ', category: 'string-op' },
      ];
    case 'number':
      return [
        { label: '==', insertText: ' == ', category: 'comparison' },
        { label: '!=', insertText: ' != ', category: 'comparison' },
        { label: '>', insertText: ' > ', category: 'comparison' },
        { label: '>=', insertText: ' >= ', category: 'comparison' },
        { label: '<', insertText: ' < ', category: 'comparison' },
        { label: '<=', insertText: ' <= ', category: 'comparison' },
      ];
    case 'array':
      return [
        { label: 'len(...) > 0', insertText: '', category: 'array-op' },
        { label: '[0]', insertText: '[0]', category: 'navigation' },
        { label: '==', insertText: ' == ', category: 'comparison' },
        { label: '!=', insertText: ' != ', category: 'comparison' },
      ];
    default:
      return [
        { label: '==', insertText: ' == ', category: 'comparison' },
        { label: '!=', insertText: ' != ', category: 'comparison' },
      ];
  }
}

function getRHSValues(
  type: ResponseExpressionValueType,
  operator: string,
): { label: string; insertText: string; selectionOffset?: number }[] {
  if (operator === 'contains' || operator === 'startsWith' || operator === 'endsWith' || operator === 'matches') {
    return [{ label: '""', insertText: '""', selectionOffset: 1 }];
  }

  if (operator === '>' || operator === '>=' || operator === '<' || operator === '<=') {
    return [{ label: '0', insertText: '0' }];
  }

  switch (type) {
    case 'boolean':
      return [
        { label: 'true', insertText: 'true' },
        { label: 'false', insertText: 'false' },
        { label: 'null', insertText: 'null' },
        { label: 'nil', insertText: 'nil' },
      ];
    case 'string':
      return [
        { label: '""', insertText: '""', selectionOffset: 1 },
        { label: 'null', insertText: 'null' },
        { label: 'nil', insertText: 'nil' },
      ];
    case 'number':
      return [
        { label: '0', insertText: '0' },
        { label: 'null', insertText: 'null' },
        { label: 'nil', insertText: 'nil' },
      ];
    default:
      return [
        { label: 'null', insertText: 'null' },
        { label: 'nil', insertText: 'nil' },
      ];
  }
}

function resolveResponseSample(
  sampleResponseText: string | undefined,
  schema: Record<string, unknown> | undefined,
): unknown | undefined {
  const trimmedSample = sampleResponseText?.trim();
  if (trimmedSample) {
    try {
      return JSON.parse(trimmedSample) as unknown;
    } catch {
      return undefined;
    }
  }

  if (!schema || Object.keys(schema).length === 0) {
    return undefined;
  }

  return jsonSchemaToExample(schema);
}

function populateResponseValue(symbols: Map<string, SymbolNode>, path: string, value: unknown) {
  const type = inferValueType(value);
  registerNode(symbols, path, type);

  if (type === 'object' && value && typeof value === 'object' && !Array.isArray(value)) {
    for (const [key, childValue] of Object.entries(value as Record<string, unknown>)) {
      const access = IDENTIFIER_PATTERN.test(key) ? key : `[${JSON.stringify(key)}]`;
      const childPath = IDENTIFIER_PATTERN.test(key) ? `${path}.${key}` : `${path}${access}`;
      const childType = inferValueType(childValue);
      addMember(symbols, path, {
        label: access,
        insertText: access,
        path: childPath,
        type: childType,
        boost: 250,
      });
      populateResponseValue(symbols, childPath, childValue);
    }
    return;
  }

  if (type === 'array') {
    const arrayValue = Array.isArray(value) ? value : [];
    const childValue = arrayValue[0];
    const childType = inferValueType(childValue);
    const childPath = `${path}[0]`;
    addMember(symbols, path, {
      label: '[0]',
      insertText: '[0]',
      path: childPath,
      type: childType,
      boost: 240,
    });
    registerNode(symbols, childPath, childType);
    if (childValue !== undefined) {
      populateResponseValue(symbols, childPath, childValue);
    }
  }
}

function applyActionTemplate(action: ExternalApiExpressionAction, path: string): {
  text: string;
  selectionOffset?: number;
} {
  const text = action.template.replaceAll('{{path}}', path);
  return {
    text,
    selectionOffset: text.includes('""') ? text.indexOf('""') + 1 : undefined,
  };
}

function registerNode(
  symbols: Map<string, SymbolNode>,
  path: string,
  type: ResponseExpressionValueType,
  description?: string,
) {
  const existing = symbols.get(path);
  if (existing) {
    existing.type = type === 'unknown' ? existing.type : type;
    if (description) {
      existing.description = description;
    }
    return existing;
  }

  const node: SymbolNode = {
    path,
    type,
    description,
    members: [],
  };
  symbols.set(path, node);
  return node;
}

function addMember(symbols: Map<string, SymbolNode>, parentPath: string, member: SymbolMember) {
  const parent = registerNode(symbols, parentPath, 'unknown');
  if (parent.members.some((existing) => existing.label === member.label)) {
    return;
  }
  parent.members.push(member);
}

function inferValueType(value: unknown): ResponseExpressionValueType {
  if (value === null) {
    return 'null';
  }
  if (Array.isArray(value)) {
    return 'array';
  }
  switch (typeof value) {
    case 'boolean':
      return 'boolean';
    case 'number':
      return 'number';
    case 'string':
      return 'string';
    case 'object':
      return 'object';
    default:
      return 'unknown';
  }
}

function toValueType(type: string): ResponseExpressionValueType {
  switch (type) {
    case 'object':
    case 'array':
    case 'string':
    case 'number':
    case 'boolean':
    case 'null':
      return type;
    default:
      return 'unknown';
  }
}

function normalizeExprPath(path: string): string {
  return path.replace(/\['([^']+)'\]/g, (_, key: string) => `[${JSON.stringify(key)}]`);
}

function buildStartSymbolOptions(
  parsedContext: ParsedCompletionContext,
  catalog: ResponseExpressionCatalog,
): ResponseExpressionCompletion[] {
  return getStartSymbols(catalog).map((symbol) => ({
    label: symbol.path,
    insertText: symbol.path,
    detail: symbol.description,
    category: 'symbol',
    type: 'variable' as const,
    boost: symbol.path === 'response' ? 260 : 240,
    applyFrom: parsedContext.applyFrom,
    applyTo: parsedContext.applyTo,
  }));
}

function getStartBoundary(text: string): StartBoundary | null {
  const trimmed = text.trimEnd();
  if (trimmed === '') {
    return 'empty';
  }

  if (trimmed.endsWith('(')) {
    return 'group';
  }

  const boundaryMatch = trimmed.match(/(?:^|[\s(])(and|or|not)$/);
  if (!boundaryMatch) {
    return null;
  }

  return boundaryMatch[1] === 'not' ? 'after-not' : 'logical';
}

function isLogicKeywordPrefix(value: string): boolean {
  return value === '' || 'and'.startsWith(value) || 'or'.startsWith(value);
}

function isCompleteClause(text: string): boolean {
  return COMPLETE_CLAUSE_REGEX.test(text.trimEnd());
}

function toExpressionVariableType(type: DraftVariableConfig['type'] | undefined): ResponseExpressionValueType {
  switch (type) {
    case 'string':
      return 'string';
    case 'number':
      return 'number';
    case 'boolean':
      return 'boolean';
    default:
      return 'unknown';
  }
}

function getStartSymbols(catalog: ResponseExpressionCatalog): SymbolNode[] {
  return ['response', 'vars']
    .map((path) => catalog.symbols.get(path))
    .filter((symbol): symbol is SymbolNode => Boolean(symbol));
}

function updateMemberType(
  symbols: Map<string, SymbolNode>,
  parentPath: string,
  memberLabel: string,
  type: ResponseExpressionValueType,
) {
  const parent = symbols.get(parentPath);
  if (!parent) {
    return;
  }

  const member = parent.members.find((candidate) => candidate.label === memberLabel);
  if (member) {
    member.type = type;
  }
}

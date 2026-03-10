import { inferSchema } from '@jsonhero/schema-infer';
import { csv2json, json2csv } from 'csv42';

export type JsonRecord = Record<string, unknown>;
export type CsvDelimiter = 'auto' | ',' | ';' | '\t' | '|';

export interface SchemaFieldDescriptor {
  path: string;
  type: string;
  required: boolean;
  nullable: boolean;
}

export interface KeyCandidateAnalysis {
  path: string;
  score: number;
  unique: boolean;
  duplicateCount: number;
  distinctCount: number;
  totalCount: number;
  identifierLike: boolean;
  sampleValue: string;
}

export interface ScalarPathValue {
  rowNumber: number;
  value: string;
}

export interface DuplicateGroup {
  value: string;
  count: number;
  rowNumbers: number[];
}

export interface DuplicateReportRow {
  duplicateField: string;
  duplicateValue: string;
  occurrenceCount: number;
  rowNumber: number;
  recordJson: string;
}

export function isJsonRecord(value: unknown): value is JsonRecord {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export function detectCsvDelimiter(text: string): Exclude<CsvDelimiter, 'auto'> {
  const headerLine = text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .find((line) => line.length > 0);

  if (!headerLine) {
    return ',';
  }

  const candidates: Exclude<CsvDelimiter, 'auto'>[] = [',', ';', '\t', '|'];
  let best: Exclude<CsvDelimiter, 'auto'> = ',';
  let bestScore = -1;

  for (const candidate of candidates) {
    const score = headerLine.split(candidate).length - 1;
    if (score > bestScore) {
      best = candidate;
      bestScore = score;
    }
  }

  return bestScore > 0 ? best : ',';
}

export function resolveCsvDelimiter(
  text: string,
  delimiter: CsvDelimiter,
): Exclude<CsvDelimiter, 'auto'> {
  return delimiter === 'auto' ? detectCsvDelimiter(text) : delimiter;
}

export function parseCsvRecords(
  text: string,
  delimiter: Exclude<CsvDelimiter, 'auto'> = ',',
): JsonRecord[] {
  return csv2json<JsonRecord>(text, { delimiter, nested: true }) as JsonRecord[];
}

export function parseJsonRecords(text: string): JsonRecord[] {
  const parsed = JSON.parse(text) as unknown;
  if (!Array.isArray(parsed)) {
    throw new Error('JSON input must be an array of objects');
  }
  if (!parsed.every(isJsonRecord)) {
    throw new Error('JSON input must contain only objects');
  }
  return parsed;
}

export function inferSegmentSchema(records: JsonRecord[]): Record<string, unknown> {
  const schema = inferSchema(records).toJSONSchema() as Record<string, unknown>;
  return normalizeSchemaNode(schema, [records]) as Record<string, unknown>;
}

export function getValueAtPath(record: JsonRecord, path: string): unknown {
  if (!path) return undefined;
  return path.split('.').reduce<unknown>((current, part) => {
    if (!isJsonRecord(current)) return undefined;
    return current[part];
  }, record);
}

export function collectFieldPaths(records: JsonRecord[]): string[] {
  const paths = new Set<string>();

  const walk = (value: unknown, prefix: string) => {
    if (isJsonRecord(value)) {
      for (const [key, nested] of Object.entries(value)) {
        const next = prefix ? `${prefix}.${key}` : key;
        walk(nested, next);
      }
      return;
    }

    if (Array.isArray(value)) {
      if (prefix) paths.add(prefix);
      return;
    }

    if (prefix) paths.add(prefix);
  };

  for (const record of records) {
    walk(record, '');
  }

  return [...paths].sort();
}

export function analyzeKeyCandidates(records: JsonRecord[]): KeyCandidateAnalysis[] {
  const fields = collectFieldPaths(records);
  const analyses: KeyCandidateAnalysis[] = [];

  for (const path of fields) {
    const values = collectScalarPathValues(records, path);
    if (!values) {
      continue;
    }

    const serializedValues = values.map((entry) => entry.value);
    const distinctCount = new Set(serializedValues).size;
    const duplicateCount = serializedValues.length - distinctCount;
    const identifierWeight = getIdentifierHintWeight(path);
    const identifierLike = identifierWeight > 0;
    analyses.push({
      path,
      score: scoreKeyCandidate(path, serializedValues, duplicateCount, identifierWeight),
      unique: duplicateCount === 0,
      duplicateCount,
      distinctCount,
      totalCount: serializedValues.length,
      identifierLike,
      sampleValue: serializedValues[0] ?? '',
    });
  }

  return analyses.sort((left, right) => {
    if (left.unique !== right.unique) {
      return left.unique ? -1 : 1;
    }
    if (left.score !== right.score) {
      return right.score - left.score;
    }
    return left.path.localeCompare(right.path);
  });
}

export function collectKeyCandidates(records: JsonRecord[]): string[] {
  return analyzeKeyCandidates(records)
    .filter((candidate) => candidate.unique)
    .map((candidate) => candidate.path);
}

export function getPreferredKeyCandidate(
  candidates: KeyCandidateAnalysis[],
): KeyCandidateAnalysis | null {
  return candidates.find((candidate) => candidate.unique) ?? null;
}

export function collectScalarPathValues(
  records: JsonRecord[],
  path: string,
): ScalarPathValue[] | null {
  const values: ScalarPathValue[] = [];

  for (const [index, record] of records.entries()) {
    const candidate = serializeKeyCandidateValue(getValueAtPath(record, path));
    if (candidate == null) {
      return null;
    }
    values.push({ rowNumber: index + 1, value: candidate });
  }

  return values;
}

export function getDuplicateGroups(records: JsonRecord[], path: string): DuplicateGroup[] {
  return analyzeDuplicateField(records, path)?.groups ?? [];
}

export function buildDuplicateReportRows(
  records: JsonRecord[],
  path: string,
): DuplicateReportRow[] {
  const analysis = analyzeDuplicateField(records, path);
  if (!analysis) {
    return [];
  }

  return analysis.values
    .filter((entry) => analysis.groupMap.has(entry.value))
    .map((entry) => ({
      duplicateField: path,
      duplicateValue: entry.value,
      occurrenceCount: analysis.groupMap.get(entry.value)?.count ?? 0,
      rowNumber: entry.rowNumber,
      recordJson: JSON.stringify(records[entry.rowNumber - 1] ?? {}),
    }));
}

export function createDuplicateReportCsv(records: JsonRecord[], path: string): string {
  const rows = buildDuplicateReportRows(records, path);
  if (rows.length === 0) {
    return '';
  }

  return json2csv(rows);
}

export function flattenSchemaFields(schema: Record<string, unknown>): SchemaFieldDescriptor[] {
  const fields: SchemaFieldDescriptor[] = [];
  const root = getSchemaRoot(schema);

  const walk = (node: Record<string, unknown>, path: string, required: boolean) => {
    const typeInfo = getSchemaType(node.type);
    const properties = isJsonRecord(node.properties) ? node.properties : null;

    if (properties) {
      const requiredFields = Array.isArray(node.required)
        ? new Set(node.required.filter((v): v is string => typeof v === 'string'))
        : new Set<string>();
      for (const [key, child] of Object.entries(properties)) {
        if (!isJsonRecord(child)) continue;
        const nextPath = path ? `${path}.${key}` : key;
        walk(child, nextPath, requiredFields.has(key));
      }
      return;
    }

    if (path) {
      fields.push({
        path,
        type: typeInfo.type,
        required,
        nullable: typeInfo.nullable,
      });
    }
  };

  walk(root, '', true);
  return fields;
}

function getSchemaRoot(schema: Record<string, unknown>): Record<string, unknown> {
  if (schema.type === 'array' && isJsonRecord(schema.items)) {
    return schema.items;
  }
  return schema;
}

function getSchemaType(typeValue: unknown): { type: string; nullable: boolean } {
  if (Array.isArray(typeValue)) {
    const types = typeValue.filter((value): value is string => typeof value === 'string');
    return {
      type: types.filter((value) => value !== 'null').join(' | ') || 'unknown',
      nullable: types.includes('null'),
    };
  }
  if (typeof typeValue === 'string') {
    return { type: typeValue, nullable: false };
  }
  return { type: 'unknown', nullable: false };
}

export function formatPreviewValue(value: unknown): string {
  if (value == null) return '-';
  if (typeof value === 'string' || typeof value === 'number') return String(value);
  if (typeof value === 'boolean') return value ? 'true' : 'false';
  return JSON.stringify(value);
}

function serializeKeyCandidateValue(value: unknown): string | null {
  if (typeof value === 'string') {
    const trimmed = value.trim();
    return trimmed.length > 0 ? trimmed : null;
  }

  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(value);
  }

  return null;
}

function analyzeDuplicateField(
  records: JsonRecord[],
  path: string,
): {
  values: ScalarPathValue[];
  groups: DuplicateGroup[];
  groupMap: Map<string, DuplicateGroup>;
} | null {
  const values = collectScalarPathValues(records, path);
  if (!values) {
    return null;
  }

  const rowMap = new Map<string, number[]>();
  for (const entry of values) {
    const rows = rowMap.get(entry.value) ?? [];
    rows.push(entry.rowNumber);
    rowMap.set(entry.value, rows);
  }

  const groups = [...rowMap.entries()]
    .filter(([, rowNumbers]) => rowNumbers.length > 1)
    .map(([value, rowNumbers]) => ({
      value,
      count: rowNumbers.length,
      rowNumbers,
    }))
    .sort((left, right) => {
      if (left.count !== right.count) {
        return right.count - left.count;
      }
      return left.value.localeCompare(right.value);
    });

  return {
    values,
    groups,
    groupMap: new Map(groups.map((group) => [group.value, group])),
  };
}

function normalizeSchemaNode(
  node: Record<string, unknown>,
  samples: unknown[],
): Record<string, unknown> {
  const normalized: Record<string, unknown> = { ...node };
  const observedTypes = collectObservedTypes(samples);

  if (isJsonRecord(node.properties)) {
    normalized.properties = Object.fromEntries(
      Object.entries(node.properties).map(([key, child]) => [
        key,
        isJsonRecord(child)
          ? normalizeSchemaNode(child, collectPropertySamples(samples, key))
          : child,
      ]),
    );
  }

  if (isJsonRecord(node.items)) {
    normalized.items = normalizeSchemaNode(node.items, collectArraySamples(samples));
  }

  if (Array.isArray(node.anyOf)) {
    const normalizedAlternatives = dedupeSchemaAlternatives(
      node.anyOf.map((alternative) =>
        isJsonRecord(alternative) ? normalizeSchemaNode(alternative, samples) : alternative,
      ),
    );

    if (observedTypes.has('null') && !alternativesAllowType(normalizedAlternatives, 'null')) {
      normalizedAlternatives.push({ type: 'null' });
    }

    const collapsedTypes = collapseSimpleSchemaAlternatives(normalizedAlternatives);
    if (collapsedTypes) {
      delete normalized.anyOf;
      normalized.type = mergeSchemaTypeDeclaration(normalized.type, collapsedTypes);
    } else {
      normalized.anyOf = normalizedAlternatives;
    }
  }

  if ('type' in normalized) {
    normalized.type = mergeSchemaTypeDeclaration(normalized.type, observedTypes);
  }

  return normalized;
}

function collectPropertySamples(samples: unknown[], key: string): unknown[] {
  const propertySamples: unknown[] = [];

  for (const sample of samples) {
    if (!isJsonRecord(sample) || !(key in sample)) {
      continue;
    }

    propertySamples.push(sample[key]);
  }

  return propertySamples;
}

function collectArraySamples(samples: unknown[]): unknown[] {
  return samples.flatMap((sample) => (Array.isArray(sample) ? sample : []));
}

function collectObservedTypes(samples: unknown[]): Set<string> {
  const types = new Set<string>();

  for (const sample of samples) {
    types.add(getJsonValueType(sample));
  }

  return types;
}

function getJsonValueType(value: unknown): string {
  if (value === null) {
    return 'null';
  }

  if (typeof value === 'string') {
    return 'string';
  }

  if (typeof value === 'boolean') {
    return 'boolean';
  }

  if (typeof value === 'number') {
    return Number.isInteger(value) ? 'integer' : 'number';
  }

  if (Array.isArray(value)) {
    return 'array';
  }

  if (isJsonRecord(value)) {
    return 'object';
  }

  return 'unknown';
}

function dedupeSchemaAlternatives(alternatives: unknown[]): Record<string, unknown>[] {
  const seen = new Set<string>();
  const unique: Record<string, unknown>[] = [];

  for (const alternative of alternatives) {
    if (!isJsonRecord(alternative)) {
      continue;
    }

    const serialized = stableStringify(alternative);
    if (seen.has(serialized)) {
      continue;
    }

    seen.add(serialized);
    unique.push(alternative);
  }

  return unique;
}

function stableStringify(value: unknown): string {
  if (Array.isArray(value)) {
    return `[${value.map((entry) => stableStringify(entry)).join(',')}]`;
  }

  if (isJsonRecord(value)) {
    const keys = Object.keys(value).sort();
    return `{${keys.map((key) => `${JSON.stringify(key)}:${stableStringify(value[key])}`).join(',')}}`;
  }

  return JSON.stringify(value);
}

function alternativesAllowType(
  alternatives: Record<string, unknown>[],
  expectedType: string,
): boolean {
  return alternatives.some((alternative) => extractSchemaTypes(alternative.type).has(expectedType));
}

function collapseSimpleSchemaAlternatives(
  alternatives: Record<string, unknown>[],
): Set<string> | null {
  const types = new Set<string>();

  for (const alternative of alternatives) {
    if (Object.keys(alternative).length !== 1 || !('type' in alternative)) {
      return null;
    }

    const alternativeTypes = extractSchemaTypes(alternative.type);
    if (alternativeTypes.size === 0) {
      return null;
    }

    for (const type of alternativeTypes) {
      types.add(type);
    }
  }

  return types.size > 0 ? types : null;
}

function mergeSchemaTypeDeclaration(
  currentTypeValue: unknown,
  observedTypes: Set<string>,
): unknown {
  const mergedTypes = extractSchemaTypes(currentTypeValue);
  for (const observedType of observedTypes) {
    if (observedType !== 'unknown') {
      mergedTypes.add(observedType);
    }
  }

  if (mergedTypes.size === 0) {
    return currentTypeValue;
  }

  const orderedTypes = [...mergedTypes].sort(compareSchemaTypeNames);
  return orderedTypes.length === 1 ? orderedTypes[0] : orderedTypes;
}

function extractSchemaTypes(typeValue: unknown): Set<string> {
  const types = new Set<string>();

  if (typeof typeValue === 'string') {
    types.add(typeValue);
    return types;
  }

  if (Array.isArray(typeValue)) {
    for (const candidate of typeValue) {
      if (typeof candidate === 'string') {
        types.add(candidate);
      }
    }
  }

  return types;
}

function compareSchemaTypeNames(left: string, right: string): number {
  const order = ['integer', 'number', 'string', 'boolean', 'object', 'array', 'null'];
  const leftIndex = order.indexOf(left);
  const rightIndex = order.indexOf(right);

  if (leftIndex !== -1 || rightIndex !== -1) {
    if (leftIndex === -1) return 1;
    if (rightIndex === -1) return -1;
    return leftIndex - rightIndex;
  }

  return left.localeCompare(right);
}

function getIdentifierHintWeight(path: string): number {
  const parts = path.toLowerCase().split('.');
  const last = parts.at(-1) ?? '';

  if (
    ['id', 'key', 'uuid', 'uid', 'rbd', 'rut'].includes(last) ||
    last.endsWith('_id') ||
    last.endsWith('_key')
  ) {
    return 180;
  }

  if (
    ['code', 'codigo', 'cod', 'slug', 'folio', 'identifier', 'identificador'].includes(last) ||
    last.endsWith('_code') ||
    last.endsWith('_codigo')
  ) {
    return 110;
  }

  return 0;
}

function scoreKeyCandidate(
  path: string,
  values: string[],
  duplicateCount: number,
  identifierWeight: number,
): number {
  const depth = path.split('.').length - 1;
  const averageLength = values.reduce((sum, value) => sum + value.length, 0) / values.length;

  let score = duplicateCount === 0 ? 1000 : Math.max(0, 400 - duplicateCount * 25);
  score += identifierWeight;
  score += Math.max(0, 30 - depth * 8);

  if (averageLength <= 24) {
    score += 20;
  } else if (averageLength <= 48) {
    score += 10;
  } else if (averageLength > 96) {
    score -= 20;
  }

  return score;
}

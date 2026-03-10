import type { CsvDelimiter, JsonRecord, KeyCandidateAnalysis } from '../segment-data-utils';

export type ImportStep = 'upload' | 'key' | 'schema' | 'confirm' | 'results';

export interface ImportResultState {
  inserted: number;
  datasetVersion?: string;
  previewFields?: string[];
}

export interface ImportState {
  step: ImportStep;
  sourceType: 'csv' | 'json';
  csvDelimiter: CsvDelimiter;
  fileName: string;
  rawData: string;
  records: JsonRecord[];
  schema: Record<string, unknown> | null;
  keyCandidates: KeyCandidateAnalysis[];
  recordKeyPath: string;
  duplicateReportPath: string;
  parseError: string | null;
  result: ImportResultState | null;
}

export function initialImportState(): ImportState {
  return {
    step: 'upload',
    sourceType: 'csv',
    csvDelimiter: 'auto',
    fileName: '',
    rawData: '',
    records: [],
    schema: null,
    keyCandidates: [],
    recordKeyPath: '',
    duplicateReportPath: '',
    parseError: null,
    result: null,
  };
}

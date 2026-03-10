import { Download, Upload } from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { initialImportState } from './import-types';

import {
  analyzeKeyCandidates,
  createDuplicateReportCsv,
  flattenSchemaFields,
  getDuplicateGroups,
  getPreferredKeyCandidate,
  getValueAtPath,
  inferSegmentSchema,
  parseCsvRecords,
  parseJsonRecords,
  resolveCsvDelimiter,
  type CsvDelimiter,
  type KeyCandidateAnalysis,
} from '../segment-data-utils';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { useGlobalLoadingModal } from '@/hooks/use-global-loading';
import { getVisibleErrorMessage } from '@/lib/display-error';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useImportSegmentData } from '@/mutations/segment-mutations';

interface ImportWizardProps {
  segmentKey: string;
  onClose: () => void;
}

const CSV_DELIMITER_OPTIONS = [
  { labelKey: 'import.delimiters.auto', value: 'auto' },
  { labelKey: 'import.delimiters.comma', value: ',' },
  { labelKey: 'import.delimiters.semicolon', value: ';' },
  { labelKey: 'import.delimiters.tab', value: '\t' },
  { labelKey: 'import.delimiters.pipe', value: '|' },
] as const;

export function ImportWizard({ segmentKey, onClose }: ImportWizardProps) {
  const { t } = useTranslation('segments');
  const [state, setState] = useState(initialImportState());
  const [isParsing, setIsParsing] = useState(false);
  const parseTimeoutRef = useRef<number | null>(null);
  const importData = useImportSegmentData(segmentKey);

  const schemaFields = useMemo(
    () => (state.schema ? flattenSchemaFields(state.schema) : []),
    [state.schema],
  );
  const uniqueKeyCandidates = useMemo(
    () => state.keyCandidates.filter((candidate) => candidate.unique),
    [state.keyCandidates],
  );
  const duplicateKeyCandidates = useMemo(
    () => state.keyCandidates.filter((candidate) => !candidate.unique),
    [state.keyCandidates],
  );
  const selectedCandidate = useMemo(
    () => state.keyCandidates.find((candidate) => candidate.path === state.recordKeyPath) ?? null,
    [state.keyCandidates, state.recordKeyPath],
  );
  const selectedDuplicateCandidate = useMemo(
    () => duplicateKeyCandidates.find((candidate) => candidate.path === state.duplicateReportPath) ?? null,
    [duplicateKeyCandidates, state.duplicateReportPath],
  );
  const selectedDuplicateGroups = useMemo(
    () => (
      state.duplicateReportPath
        ? getDuplicateGroups(state.records, state.duplicateReportPath)
        : []
    ),
    [state.records, state.duplicateReportPath],
  );
  const selectedDuplicateValues = useMemo(
    () => new Set(selectedDuplicateGroups.map((group) => group.value)),
    [selectedDuplicateGroups],
  );

  const update = (patch: Partial<typeof state>) => setState((current) => ({ ...current, ...patch }));

  useGlobalLoadingModal(isParsing || importData.isPending, {
    title: isParsing
      ? t('import.processingTitle', { defaultValue: 'Procesando archivo' })
      : t('import.importingTitle', { defaultValue: 'Importando segmento' }),
    description: isParsing
      ? t('import.processingDescription', {
          defaultValue: 'Revisando registros y generando el schema.',
        })
      : t('import.importingDescription', {
          defaultValue: 'Espera un momento mientras reemplazamos los registros del segmento.',
        }),
  });

  useEffect(() => () => {
    if (parseTimeoutRef.current !== null) {
      window.clearTimeout(parseTimeoutRef.current);
    }
  }, []);

  const parseSource = () => {
    if (isParsing) return;

    const sourceType = state.sourceType;
    const rawData = state.rawData;
    const csvDelimiter = state.csvDelimiter;

    setIsParsing(true);
    parseTimeoutRef.current = window.setTimeout(() => {
      try {
        const records = sourceType === 'csv'
          ? parseCsvRecords(rawData, resolveCsvDelimiter(rawData, csvDelimiter))
          : parseJsonRecords(rawData);
        if (records.length === 0) {
          update({
            parseError: t('import.errors.noRecords'),
            records: [],
            schema: null,
            keyCandidates: [],
            recordKeyPath: '',
            duplicateReportPath: '',
          });
          return;
        }

        const schema = inferSegmentSchema(records);
        const keyCandidates = analyzeKeyCandidates(records);
        const preferredKey = getPreferredKeyCandidate(keyCandidates)?.path ?? '';
        const preferredDuplicatePath = keyCandidates.find((candidate) => !candidate.unique)?.path ?? '';

        update({
          step: 'key',
          records,
          schema,
          keyCandidates,
          recordKeyPath: preferredKey,
          duplicateReportPath: preferredDuplicatePath,
          parseError: null,
        });
      } catch (error) {
        const message = sourceType === 'csv'
          ? formatCsvParseError(error, csvDelimiter, rawData, t)
          : getVisibleErrorMessage(error, t('import.error'));
        update({
          parseError: message,
          records: [],
          schema: null,
          keyCandidates: [],
          recordKeyPath: '',
          duplicateReportPath: '',
        });
      } finally {
        setIsParsing(false);
        parseTimeoutRef.current = null;
      }
    }, 0);
  };

  const handleSourceTypeChange = (value: 'csv' | 'json') => {
    setState({
      ...initialImportState(),
      sourceType: value,
      csvDelimiter: value === 'csv' ? state.csvDelimiter : 'auto',
    });
  };

  const handleImport = () => {
    if (!state.schema || !state.recordKeyPath) return;

    importData.mutate(
      {
        mode: 'replace',
        sourceType: state.sourceType,
        recordKeyPath: state.recordKeyPath,
        schema: state.schema,
        records: state.records,
      },
      {
        onSuccess: (result) => update({ step: 'results', result }),
        onError: () => toast.error(t('import.error')),
      },
    );
  };

  const handleDownloadDuplicateReport = () => {
    if (!state.duplicateReportPath) return;

    const csv = createDuplicateReportCsv(state.records, state.duplicateReportPath);
    if (!csv) {
      toast.error(
        t('import.errors.noDuplicateValuesForReport', {
          defaultValue: 'No hay duplicados para exportar en el campo seleccionado.',
        }),
      );
      return;
    }

    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' });
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = buildDuplicateReportFileName(
      segmentKey,
      state.fileName,
      state.duplicateReportPath,
    );
    document.body.append(link);
    link.click();
    link.remove();
    window.URL.revokeObjectURL(url);
  };

  if (state.step === 'results' && state.result) {
    return (
      <div className="space-y-4">
        <h2 className="text-lg font-semibold">{t('import.resultsTitle')}</h2>
        <div className="rounded-md border bg-muted/20 p-4">
          <p>{t('import.inserted', { count: state.result.inserted })}</p>
          {state.result.datasetVersion ? <p className="text-muted-foreground text-sm">{state.result.datasetVersion}</p> : null}
        </div>
        <div className="flex justify-end">
          <Button onClick={onClose}>{t('actions.close', { ns: 'common', defaultValue: 'Close' })}</Button>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {state.step === 'upload' ? (
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>{t('import.sourceType')}</Label>
            <Select value={state.sourceType} onValueChange={handleSourceTypeChange}>
              <SelectTrigger className="w-48">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="csv">CSV</SelectItem>
                <SelectItem value="json">JSON</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {state.sourceType === 'csv' ? (
            <div className="space-y-2">
              <Label>{t('import.delimiter')}</Label>
              <Select value={state.csvDelimiter} onValueChange={(value: CsvDelimiter) => update({ csvDelimiter: value })}>
                <SelectTrigger className="w-56">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {CSV_DELIMITER_OPTIONS.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {t(option.labelKey)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          ) : null}

          <FilePicker
            sourceType={state.sourceType}
            fileName={state.fileName}
            onLoaded={(rawData, fileName) => update({ rawData, fileName, parseError: null })}
          />

          <div className="space-y-2">
            <Label>{t('import.paste')}</Label>
            <textarea
              value={state.rawData}
              onChange={(event) => update({ rawData: event.target.value, parseError: null })}
              rows={10}
              className="flex w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-xs"
              placeholder={getSourcePlaceholder(state.sourceType, state.csvDelimiter)}
            />
          </div>

          {state.parseError ? <p className="text-destructive text-sm">{state.parseError}</p> : null}

          <div className="flex justify-end">
            <Button onClick={parseSource} disabled={!state.rawData.trim() || isParsing}>
              {t('import.next')}
            </Button>
          </div>
        </div>
      ) : null}

      {state.step === 'key' ? (
        <div className="space-y-4">
          <h2 className="text-lg font-semibold">{t('import.keyTitle')}</h2>
          <div className="space-y-2">
            <Label>{t('import.keyField')}</Label>
            <Select value={state.recordKeyPath} onValueChange={(value) => update({ recordKeyPath: value })}>
              <SelectTrigger>
                <SelectValue placeholder={t('import.selectKey')} />
              </SelectTrigger>
              <SelectContent>
                {uniqueKeyCandidates.map((candidate) => (
                  <SelectItem key={candidate.path} value={candidate.path}>
                    <div className="flex min-w-0 items-center gap-2">
                      <span className="truncate">{candidate.path}</span>
                      {candidate.identifierLike ? (
                        <Badge variant="success" className="shrink-0">
                          {t('import.recommended')}
                        </Badge>
                      ) : null}
                    </div>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {uniqueKeyCandidates.length === 0 ? (
              <p className="text-destructive text-sm">{t('import.errors.noUniqueKeyCandidates')}</p>
            ) : null}
            {state.keyCandidates.length === 0 ? (
              <p className="text-destructive text-sm">{t('import.errors.noKeyCandidates')}</p>
            ) : null}
            {uniqueKeyCandidates.length > 0 ? (
              <p className="text-muted-foreground text-sm">
                {t('import.keySuggestions', {
                  count: uniqueKeyCandidates.length,
                  analyzed: state.keyCandidates.length,
                })}
              </p>
            ) : null}
          </div>

          {selectedCandidate ? <KeyCandidateCard candidate={selectedCandidate} /> : null}

          {duplicateKeyCandidates.length > 0 ? (
            <div className="space-y-4 rounded-md border bg-muted/20 p-4">
              <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
                <div className="min-w-0 flex-1 space-y-2">
                  <p className="text-sm font-medium">{t('import.excludedTitle')}</p>
                  <Label>
                    {t('import.duplicateField', {
                      defaultValue: 'Campo duplicado',
                    })}
                  </Label>
                  <Select
                    value={state.duplicateReportPath}
                    onValueChange={(value) => update({ duplicateReportPath: value })}
                  >
                    <SelectTrigger>
                      <SelectValue
                        placeholder={t('import.selectDuplicateField', {
                          defaultValue: 'Seleccione un campo duplicado',
                        })}
                      />
                    </SelectTrigger>
                    <SelectContent>
                      {duplicateKeyCandidates.map((candidate) => (
                        <SelectItem key={candidate.path} value={candidate.path}>
                          <div className="flex min-w-0 items-center gap-2">
                            <span className="truncate">{candidate.path}</span>
                            <span className="text-muted-foreground text-xs">
                              {t('import.duplicateSummary', { count: candidate.duplicateCount })}
                            </span>
                          </div>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  className="shrink-0"
                  onClick={handleDownloadDuplicateReport}
                  disabled={!selectedDuplicateCandidate || selectedDuplicateGroups.length === 0}
                >
                  <Download className="mr-2 h-4 w-4" />
                  {t('import.downloadDuplicateReport', {
                    defaultValue: 'Descargar reporte de duplicados',
                  })}
                </Button>
              </div>

              {selectedDuplicateCandidate ? <KeyCandidateCard candidate={selectedDuplicateCandidate} /> : null}

              {selectedDuplicateGroups.length > 0 ? (
                <div className="space-y-2">
                  <p className="text-sm font-medium">
                    {t('import.duplicateValuesTitle', {
                      defaultValue: 'Valores duplicados detectados',
                    })}
                  </p>
                  <div className="flex flex-wrap gap-2">
                    {selectedDuplicateGroups.slice(0, 8).map((group) => (
                      <div
                        key={group.value}
                        className="rounded-md border bg-background px-3 py-2 text-xs"
                      >
                        <p className="max-w-[260px] break-all font-mono">{group.value}</p>
                        <p className="text-muted-foreground">
                          {t('import.rowsSummary', {
                            count: group.count,
                            rows: formatRowNumbers(group.rowNumbers),
                            defaultValue: '{{count}} filas: {{rows}}',
                          })}
                        </p>
                      </div>
                    ))}
                  </div>
                  {selectedDuplicateGroups.length > 8 ? (
                    <p className="text-muted-foreground text-xs">
                      {t('import.moreDuplicateValues', {
                        count: selectedDuplicateGroups.length - 8,
                        defaultValue: '+{{count}} valores duplicados adicionales en el reporte',
                      })}
                    </p>
                  ) : null}
                  <p className="text-muted-foreground text-xs">
                    {t('import.duplicatePreviewHint', {
                      defaultValue: 'El preview resalta las filas duplicadas para el campo seleccionado.',
                    })}
                  </p>
                </div>
              ) : null}
            </div>
          ) : null}

          <PreviewTable
            records={state.records}
            duplicateHighlightPath={state.duplicateReportPath}
            duplicateHighlightValues={selectedDuplicateValues}
          />

          <div className="flex justify-between">
            <Button variant="outline" onClick={() => update({ step: 'upload' })}>{t('import.back')}</Button>
            <Button onClick={() => update({ step: 'schema' })} disabled={!state.recordKeyPath || !selectedCandidate?.unique}>
              {t('import.next')}
            </Button>
          </div>
        </div>
      ) : null}

      {state.step === 'schema' && state.schema ? (
        <div className="space-y-4">
          <h2 className="text-lg font-semibold">{t('import.schemaTitle')}</h2>
          <div className="max-w-full overflow-x-auto rounded-md border">
            <table className="min-w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/50">
                  <th className="px-4 py-3 text-left font-medium">{t('schema.path')}</th>
                  <th className="px-4 py-3 text-left font-medium">{t('schema.type')}</th>
                </tr>
              </thead>
              <tbody>
                {schemaFields.map((field) => (
                  <tr key={field.path} className="border-b last:border-0">
                    <td className="px-4 py-3 font-mono text-xs break-all">{field.path}</td>
                    <td className="px-4 py-3">{field.type}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <pre className="max-h-72 overflow-auto rounded-md border bg-muted/20 p-4 text-xs">{JSON.stringify(state.schema, null, 2)}</pre>

          <div className="flex justify-between">
            <Button variant="outline" onClick={() => update({ step: 'key' })}>{t('import.back')}</Button>
            <Button onClick={() => update({ step: 'confirm' })}>{t('import.next')}</Button>
          </div>
        </div>
      ) : null}

      {state.step === 'confirm' ? (
        <div className="space-y-4">
          <h2 className="text-lg font-semibold">{t('import.confirmTitle')}</h2>
          <div className="rounded-md border bg-muted/20 p-4 text-sm">
            <p>{t('import.summaryRecords', { count: state.records.length })}</p>
            <p>{t('import.summaryKey', { key: state.recordKeyPath })}</p>
            <p>{t('import.summarySource', { source: state.sourceType.toUpperCase() })}</p>
          </div>

          <div className="flex justify-between">
            <Button variant="outline" onClick={() => update({ step: 'schema' })}>{t('import.back')}</Button>
            <Button onClick={handleImport} disabled={isParsing || importData.isPending}>
              {importData.isPending ? t('import.importing') : t('import.startImport')}
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}

function FilePicker({
  fileName,
  sourceType,
  onLoaded,
}: {
  fileName: string;
  sourceType: 'csv' | 'json';
  onLoaded: (value: string, fileName: string) => void;
}) {
  const { t } = useTranslation('segments');
  const inputRef = useRef<HTMLInputElement | null>(null);

  return (
    <div className="space-y-3 rounded-lg border border-dashed bg-muted/20 p-4">
      <input
        ref={inputRef}
        className="sr-only"
        type="file"
        accept={sourceType === 'csv' ? '.csv,.tsv,.txt' : '.json'}
        onChange={(event) => {
          const file = event.target.files?.[0];
          if (!file) return;
          const reader = new FileReader();
          reader.onload = (loadEvent) => {
            const text = loadEvent.target?.result;
            if (typeof text === 'string') onLoaded(text, file.name);
          };
          reader.readAsText(file);
        }}
      />
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="space-y-1">
          <p className="text-sm font-medium">{t('import.fileTitle')}</p>
          <p className="text-muted-foreground text-xs">
            {sourceType === 'csv' ? t('import.fileHelpCsv') : t('import.fileHelpJson')}
          </p>
          <p className="text-xs font-mono text-muted-foreground">
            {fileName ? t('import.selectedFile', { name: fileName }) : t('import.noFileSelected')}
          </p>
        </div>
        <Button type="button" variant="outline" onClick={() => inputRef.current?.click()}>
          <Upload className="mr-2 h-4 w-4" />
          {fileName ? t('import.replaceFile') : t('import.chooseFile')}
        </Button>
      </div>
    </div>
  );
}

function getSourcePlaceholder(sourceType: 'csv' | 'json', delimiter: CsvDelimiter): string {
  if (sourceType === 'json') {
    return '[{"key":"u_1","name":"Ada","score":10}]';
  }

  const resolvedDelimiter = delimiter === 'auto' ? ',' : delimiter;
  const visibleDelimiter = resolvedDelimiter === '\t' ? '\t' : resolvedDelimiter;
  return `key${visibleDelimiter}name${visibleDelimiter}score\nu_1${visibleDelimiter}Ada${visibleDelimiter}10`;
}

function formatCsvParseError(
  error: unknown,
  selectedDelimiter: CsvDelimiter,
  rawData: string,
  t: ReturnType<typeof useTranslation<'segments'>>['t'],
): string {
  const fallback = getVisibleErrorMessage(error, t('import.error'));
  const detectedDelimiter = resolveCsvDelimiter(rawData, 'auto');

  if (selectedDelimiter === 'auto') {
    return fallback;
  }

  if (detectedDelimiter !== selectedDelimiter) {
    return t('import.errors.invalidDelimiter', {
      current: formatDelimiterLabel(selectedDelimiter),
      suggested: formatDelimiterLabel(detectedDelimiter),
      defaultValue: fallback,
    });
  }

  return fallback;
}

function formatDelimiterLabel(delimiter: Exclude<CsvDelimiter, 'auto'>): string {
  if (delimiter === '\t') {
    return 'tab';
  }

  return delimiter;
}

function PreviewTable({
  records,
  duplicateHighlightPath,
  duplicateHighlightValues,
}: {
  records: Record<string, unknown>[];
  duplicateHighlightPath?: string;
  duplicateHighlightValues?: ReadonlySet<string>;
}) {
  const preview = records.slice(0, 5);
  const columns = Object.keys(preview[0] ?? {});

  return (
    <div className="max-w-full overflow-x-auto rounded-md border">
      <table className="w-max min-w-full text-xs">
        <thead>
          <tr className="border-b bg-muted/50">
            {columns.map((column) => (
              <th key={column} className="px-3 py-2 text-left font-medium whitespace-nowrap">{column}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {preview.map((record, index) => (
            <tr
              key={index}
              className={isHighlightedDuplicate(
                record,
                duplicateHighlightPath,
                duplicateHighlightValues,
              )
                ? 'border-b bg-warning/10 last:border-0'
                : 'border-b last:border-0'}
            >
              {columns.map((column) => (
                <td key={column} className="max-w-[180px] px-3 py-2 align-top">
                  {typeof record[column] === 'object' ? JSON.stringify(record[column]) : String(record[column] ?? '')}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function KeyCandidateCard({ candidate }: { candidate: KeyCandidateAnalysis }) {
  const { t } = useTranslation('segments');

  return (
    <div className="space-y-3 rounded-md border bg-muted/20 p-4">
      <div className="flex flex-wrap items-center gap-2">
        <p className="font-mono text-sm">{candidate.path}</p>
        {candidate.unique ? (
          <Badge variant="success">{t('import.unique')}</Badge>
        ) : (
          <Badge variant="warning">{t('import.notUnique')}</Badge>
        )}
        {candidate.identifierLike ? <Badge variant="secondary">{t('import.recommended')}</Badge> : null}
      </div>
      <div className="grid gap-3 text-xs text-muted-foreground sm:grid-cols-3">
        <p>{t('import.distinctSummary', { count: candidate.distinctCount, total: candidate.totalCount })}</p>
        <p>{t('import.duplicateSummary', { count: candidate.duplicateCount })}</p>
        <p>{t('import.sampleValue', { value: candidate.sampleValue || '-' })}</p>
      </div>
    </div>
  );
}

function isHighlightedDuplicate(
  record: Record<string, unknown>,
  path?: string,
  duplicateValues?: ReadonlySet<string>,
): boolean {
  if (!path || !duplicateValues || duplicateValues.size === 0) {
    return false;
  }

  const value = resolveDuplicateHighlightValue(record, path);
  return value != null && duplicateValues.has(value);
}

function resolveDuplicateHighlightValue(
  record: Record<string, unknown>,
  path: string,
): string | null {
  const value = getValueAtPath(record, path);

  if (typeof value === 'string') {
    const trimmed = value.trim();
    return trimmed.length > 0 ? trimmed : null;
  }

  if (typeof value === 'number' && Number.isFinite(value)) {
    return String(value);
  }

  return null;
}

function formatRowNumbers(rowNumbers: number[]): string {
  const preview = rowNumbers.slice(0, 8).join(', ');
  return rowNumbers.length > 8 ? `${preview}, ...` : preview;
}

function buildDuplicateReportFileName(
  segmentKey: string,
  fileName: string,
  path: string,
): string {
  const baseName = sanitizeFilePart(
    fileName.replace(/\.[^.]+$/, '') || segmentKey || 'segment',
  );
  const fieldName = sanitizeFilePart(path || 'field');

  return `${baseName}_${fieldName}_duplicates.csv`;
}

function sanitizeFilePart(value: string): string {
  return value
    .normalize('NFKD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/[^a-zA-Z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
    .toLowerCase() || 'segment';
}

import {
  analyzeKeyCandidates,
  buildDuplicateReportRows,
  createDuplicateReportCsv,
  detectCsvDelimiter,
  getPreferredKeyCandidate,
  getDuplicateGroups,
  inferSegmentSchema,
  parseCsvRecords,
  resolveCsvDelimiter,
} from './segment-data-utils';

describe('segment data utils', () => {
  it('parses CSV with a custom delimiter', () => {
    expect(
      parseCsvRecords('key;name;score\nu_1;Ada;10', ';'),
    ).toEqual([
      { key: 'u_1', name: 'Ada', score: 10 },
    ]);
  });

  it('detects semicolon-delimited CSV files', () => {
    expect(detectCsvDelimiter('key;name;score\nu_1;Ada;10')).toBe(';');
  });

  it('resolves auto delimiter to the detected separator', () => {
    expect(resolveCsvDelimiter('key|name|score\nu_1|Ada|10', 'auto')).toBe('|');
  });

  it('prefers unique identifier-like fields as key suggestions', () => {
    const records = [
      { id: 1, code: 'A-1', nombre: 'Ada', region_id: 16 },
      { id: 2, code: 'A-2', nombre: 'Beto', region_id: 16 },
      { id: 3, code: 'A-3', nombre: 'Carla', region_id: 16 },
    ];

    const analyses = analyzeKeyCandidates(records);

    expect(analyses[0]?.path).toBe('id');
    expect(analyses.find((candidate) => candidate.path === 'region_id')?.unique).toBe(false);
    expect(getPreferredKeyCandidate(analyses)?.path).toBe('id');
  });

  it('groups duplicate values for the selected field only', () => {
    const records = [
      { rbd: 1001, nombre: 'A' },
      { rbd: 1001, nombre: 'B' },
      { rbd: 2002, nombre: 'C' },
      { rbd: 3003, nombre: 'D' },
      { rbd: 3003, nombre: 'E' },
    ];

    expect(getDuplicateGroups(records, 'rbd')).toEqual([
      { value: '1001', count: 2, rowNumbers: [1, 2] },
      { value: '3003', count: 2, rowNumbers: [4, 5] },
    ]);
  });

  it('builds a duplicate report with the affected rows', () => {
    const records = [
      { rbd: 1001, nombre: 'A' },
      { rbd: 1001, nombre: 'B' },
      { rbd: 2002, nombre: 'C' },
    ];

    expect(buildDuplicateReportRows(records, 'rbd')).toEqual([
      {
        duplicateField: 'rbd',
        duplicateValue: '1001',
        occurrenceCount: 2,
        rowNumber: 1,
        recordJson: '{"rbd":1001,"nombre":"A"}',
      },
      {
        duplicateField: 'rbd',
        duplicateValue: '1001',
        occurrenceCount: 2,
        rowNumber: 2,
        recordJson: '{"rbd":1001,"nombre":"B"}',
      },
    ]);
  });

  it('exports a CSV report for duplicated values', () => {
    const records = [
      { rbd: 1001, nombre: 'A' },
      { rbd: 1001, nombre: 'B' },
      { rbd: 2002, nombre: 'C' },
    ];

    const csv = createDuplicateReportCsv(records, 'rbd');

    expect(csv).toContain('duplicateField');
    expect(csv).toContain('duplicateValue');
    expect(csv).toContain('1001');
    expect(csv).toContain('rowNumber');
  });

  it('normalizes inferred schema unions and keeps null values', () => {
    const schema = inferSegmentSchema([
      { telefono: 123, nombre: 'Ada' },
      { telefono: null, nombre: 'Beto' },
      { telefono: 'abc', nombre: 'Carla' },
    ]);

    expect(schema).toEqual({
      type: 'array',
      items: {
        type: 'object',
        properties: {
          telefono: {
            type: ['integer', 'string', 'null'],
          },
          nombre: {
            type: 'string',
          },
        },
        required: ['telefono', 'nombre'],
      },
    });
  });
});

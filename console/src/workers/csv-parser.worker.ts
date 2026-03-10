import Papa from 'papaparse';

export interface CsvParseRequest {
  type: 'parse';
  data: string;
}

export interface CsvParseResult {
  type: 'result';
  rows: string[][];
  headers: string[];
  errors: { row: number; message: string }[];
}

self.onmessage = (e: MessageEvent<CsvParseRequest>) => {
  if (e.data.type !== 'parse') return;

  const result = Papa.parse<string[]>(e.data.data, {
    skipEmptyLines: true,
  });

  const rows = result.data;
  const headers = rows.length > 0 ? rows[0] : [];
  const dataRows = rows.slice(1);

  const errors = result.errors.map((err) => ({
    row: err.row ?? 0,
    message: err.message,
  }));

  const response: CsvParseResult = {
    type: 'result',
    rows: dataRows,
    headers,
    errors,
  };

  self.postMessage(response);
};

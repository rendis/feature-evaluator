import type { ExternalApiTestResponse } from '@/api/types';

export function getExternalApiTestResponseBody(
  testResult: ExternalApiTestResponse | null | undefined,
): unknown | undefined {
  const details = testResult?.details;
  if (!details || !Object.hasOwn(details, 'responseBody')) {
    return undefined;
  }
  return details.responseBody;
}

export function isExternalApiSchemaUsableBody(
  value: unknown,
): value is Record<string, unknown> | unknown[] {
  return value != null && typeof value === 'object';
}

export function formatExternalApiPreview(value: unknown): string {
  if (value == null) {
    return '';
  }

  if (typeof value === 'string') {
    const trimmed = value.trim();
    if (trimmed === '') {
      return value;
    }

    try {
      const parsed = JSON.parse(value) as unknown;
      if (parsed != null && typeof parsed === 'object') {
        return JSON.stringify(parsed, null, 2);
      }
    } catch {
      return value;
    }

    return value;
  }

  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

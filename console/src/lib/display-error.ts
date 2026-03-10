import { ApiClientError } from '@/api/client';

export function getVisibleErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof ApiClientError) {
    return error.message;
  }

  return fallback;
}

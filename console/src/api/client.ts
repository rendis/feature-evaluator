import type { ApiError } from './types';

import { env } from '@/config/env';
import { getWorkspaceKey } from '@/stores/workspace-store';

export class ApiClientError extends Error {
  code: string;
  messageKey: string;
  details?: Record<string, unknown>;
  requestId: string;
  status: number;

  constructor(status: number, apiError: ApiError['error']) {
    super(apiError.message);
    this.name = 'ApiClientError';
    this.code = apiError.code;
    this.messageKey = apiError.messageKey;
    this.details = apiError.details;
    this.requestId = apiError.requestId;
    this.status = status;
  }
}

let getAccessToken: (() => string | null) | null = null;

/** Register a function that returns the current access token. */
export function setTokenProvider(provider: () => string | null) {
  getAccessToken = provider;
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const url = `${env.apiUrl}${path}`;
  const headers = new Headers(options.headers);

  if (!headers.has('Content-Type') && options.body) {
    headers.set('Content-Type', 'application/json');
  }

  const token = getAccessToken?.();
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  headers.set('X-Workspace', getWorkspaceKey());

  const response = await fetch(url, { ...options, headers });

  if (!response.ok) {
    const body = (await response.json().catch(() => ({
      error: {
        code: 'UNKNOWN',
        message: response.statusText,
        messageKey: 'error.unknown',
        requestId: '',
      },
    }))) as ApiError;
    throw new ApiClientError(response.status, body.error);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined }),
  put: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'PUT', body: body ? JSON.stringify(body) : undefined }),
  patch: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'PATCH', body: body ? JSON.stringify(body) : undefined }),
  delete: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'DELETE', body: body ? JSON.stringify(body) : undefined }),
};

import type { Page } from '@playwright/test';

export interface RecordedRequest {
  url: string;
  method: string;
  headers: Record<string, string>;
  body?: unknown;
  rawBody?: string | null;
}

export interface JsonRouteResponse {
  status?: number;
  body?: unknown;
  headers?: Record<string, string>;
}

export async function recordJsonRequests(
  page: Page,
  matcher: string | RegExp,
  respondWith: JsonRouteResponse | ((request: RecordedRequest) => JsonRouteResponse) = {},
) {
  const requests: RecordedRequest[] = [];

  await page.route(matcher, async (route) => {
    const request = route.request();
    const rawBody = request.postData();
    let body: unknown = undefined;
    if (rawBody) {
      try {
        body = JSON.parse(rawBody);
      } catch {
        body = rawBody;
      }
    }

    const recorded: RecordedRequest = {
      url: request.url(),
      method: request.method(),
      headers: request.headers(),
      body,
      rawBody,
    };
    requests.push(recorded);

    if (request.method() === 'OPTIONS') {
      await route.fulfill({
        status: 204,
        headers: corsHeaders(),
        body: '',
      });
      return;
    }

    const response = typeof respondWith === 'function' ? respondWith(recorded) : respondWith;
    await route.fulfill({
      status: response.status ?? 200,
      headers: {
        ...corsHeaders(),
        ...response.headers,
      },
      body: JSON.stringify(response.body ?? {}),
    });
  });

  return requests;
}

function corsHeaders() {
  return {
    'access-control-allow-origin': '*',
    'access-control-allow-methods': 'GET,POST,PUT,PATCH,DELETE,OPTIONS',
    'access-control-allow-headers': 'content-type,authorization,x-workspace',
  };
}

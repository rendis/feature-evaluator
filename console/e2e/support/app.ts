import { recordJsonRequests } from './network';

import type { Page } from '@playwright/test';

export interface WorkspaceFixture {
  id?: string;
  key: string;
  name?: string;
  description?: string;
  archivedAt?: string | null;
  createdAt?: string;
  updatedAt?: string;
  createdBy?: string;
}

export async function primeAuthDisabledWorkspace(page: Page, workspaceKey = 'e2e-workspace') {
  await page.addInitScript((key) => {
    window.localStorage.setItem('fe-workspace', key);
  }, workspaceKey);

  const workspaces = [
    {
      id: 'workspace-e2e',
      key: workspaceKey,
      name: 'E2E Workspace',
      description: 'Workspace used by browser tests',
      createdAt: new Date('2026-03-26T00:00:00.000Z').toISOString(),
      updatedAt: new Date('2026-03-26T00:00:00.000Z').toISOString(),
      createdBy: 'e2e',
      archivedAt: null,
    },
  ];

  await recordJsonRequests(page, '**/admin/workspaces*', (request) => {
    if (request.method === 'GET') {
      return {
        body: { data: workspaces },
      };
    }

    return {
      status: 204,
      body: {},
    };
  });

  return { workspaces };
}

export async function openAppPath(page: Page, path: string) {
  await page.goto(path, { waitUntil: 'domcontentloaded' });
}

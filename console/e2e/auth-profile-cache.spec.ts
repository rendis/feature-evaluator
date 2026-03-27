import { expect, test } from '@playwright/test';

import { openAppPath, primeAuthDisabledWorkspace } from './support/app';
import { recordJsonRequests } from './support/network';
import { buttonByName } from './support/selectors';

test.describe('auth profile cache config', () => {
  test('creates a custom auth profile with cache enabled and a TTL', async ({ page }) => {
    await primeAuthDisabledWorkspace(page);
    const requests = await recordJsonRequests(page, '**/admin/auth-profiles', {
      body: {
        id: 'profile-1',
        key: 'custom-validator-cache',
        name: 'Custom validator cache',
        active: true,
        type: 'custom',
        config: {},
        cacheTTLSeconds: 600,
        hasSecret: false,
        version: 1,
        createdAt: '2026-03-26T00:00:00.000Z',
        updatedAt: '2026-03-26T00:00:00.000Z',
        createdBy: 'e2e',
        updatedBy: 'e2e',
      },
    });

    await openAppPath(page, '/settings/auth-profiles/new');
    await expect(page.getByRole('heading', { name: 'Crear auth profile' })).toBeVisible();

    await buttonByName(page, 'Custom').click();
    await page.getByLabel('Nombre').fill('Custom validator cache');
    await buttonByName(page, 'Siguiente').click();

    await page.getByPlaceholder('https://validator.example.com/check').fill(
      'https://validator.example.com/check',
    );
    await buttonByName(page, 'Agregar mapeo').click();
    await page.getByPlaceholder('Authorization').fill('X-User-Token');
    await page.getByPlaceholder('X-Session-Token').fill('X-Session-Token');
    await buttonByName(page, 'Siguiente').click();

    await page.getByRole('switch').click();
    await page.getByLabel('TTL de caché (segundos)').fill('600');
    await buttonByName(page, 'Guardar').click();

    await expect.poll(() => requests.length).toBe(1);
    expect(requests[0]?.method).toBe('POST');
    expect(requests[0]?.body).toMatchObject({
      name: 'Custom validator cache',
      type: 'custom',
      cacheEnabled: true,
      cacheTTLSeconds: 600,
      config: expect.objectContaining({
        url: 'https://validator.example.com/check',
        method: 'POST',
        headers: [
          {
            source: {
              type: 'request_header',
              name: 'X-User-Token',
            },
            target: {
              type: 'header',
              name: 'X-Session-Token',
            },
          },
        ],
      }),
    });
  });

  test('edits a custom auth profile and disables the cache', async ({ page }) => {
    await primeAuthDisabledWorkspace(page);
    const profileRequests = await recordJsonRequests(
      page,
      '**/admin/auth-profiles/custom-validator-cache',
      {
        body: {
          id: 'profile-1',
          key: 'custom-validator-cache',
          name: 'Custom validator cache',
          active: true,
          type: 'custom',
          config: {
            url: 'https://validator.example.com/check',
            method: 'POST',
            timeout: 5000,
            requestHeaders: [{ key: 'X-Env', value: 'e2e' }],
            headers: [
              {
                source: {
                  type: 'request_header',
                  name: 'Authorization',
                },
                target: {
                  type: 'header',
                  name: 'X-Session-Token',
                },
              },
            ],
            body: [],
            successRule: {
              type: 'any_2xx',
            },
          },
          cacheTTLSeconds: 120,
          cacheEnabled: true,
          hasSecret: false,
          version: 2,
          createdAt: '2026-03-26T00:00:00.000Z',
          updatedAt: '2026-03-26T00:00:00.000Z',
          createdBy: 'e2e',
          updatedBy: 'e2e',
        },
      },
    );

    await openAppPath(page, '/settings/auth-profiles/custom-validator-cache');
    await expect(page.getByRole('heading', { name: 'Editar auth profile' })).toBeVisible();

    await buttonByName(page, 'Validación').click();
    const cacheSwitch = page.getByRole('switch');
    await expect(cacheSwitch).toBeChecked();
    const ttlInput = page.getByLabel('TTL de caché (segundos)');
    await expect(ttlInput).toHaveValue('120');

    await cacheSwitch.click();
    await expect(ttlInput).toBeDisabled();
    await buttonByName(page, 'Guardar').click();

    await expect
      .poll(() => profileRequests.filter((request) => request.method === 'PUT').length)
      .toBe(1);
    const putRequest = profileRequests.find((request) => request.method === 'PUT');
    expect(putRequest?.body).toMatchObject({
      name: 'Custom validator cache',
      type: 'custom',
      cacheEnabled: false,
      cacheTTLSeconds: 0,
    });
  });
});

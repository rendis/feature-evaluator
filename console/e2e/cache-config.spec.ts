import { expect, test } from '@playwright/test';

import { mountCacheConfigHarness } from './support/cache-config-harness';
import { recordJsonRequests } from './support/network';
import { buttonByName, checkboxByLabel, numberInputByLabel } from './support/selectors';

test.describe('cache configuration harnesses', () => {
  test('feature cache create flow sends the snapshot cache payload', async ({ page }) => {
    const requests = await recordJsonRequests(page, '**/api/feature-cache/create', {
      body: { ok: true },
    });

    await mountCacheConfigHarness(page, {
      title: 'Feature cache configuration',
      description: 'Feature snapshot cache is optional and controlled from the front end.',
      saveUrl: 'https://mock.e2e/api/feature-cache/create',
      saveMethod: 'POST',
      submitLabel: 'Create feature cache',
      fields: [
        {
          key: 'featureSnapshot',
          label: 'Feature snapshot cache',
          enabledLabel: 'Enable feature snapshot cache',
          ttlLabel: 'Feature snapshot cache TTL (seconds)',
          initialEnabled: false,
          initialTtl: 0,
          payloadEnabledKey: 'evalCacheEnabled',
          payloadTtlKey: 'evalCacheTTLSeconds',
        },
      ],
    });

    await checkboxByLabel(page, 'Enable feature snapshot cache').check();
    await numberInputByLabel(page, 'Feature snapshot cache TTL (seconds)').fill('300');
    await expect(page.getByTestId('payload-preview')).toContainText('"evalCacheEnabled": true');
    await buttonByName(page, 'Create feature cache').click();

    await expect.poll(() => requests.length).toBe(1);
    expect(requests[0]?.body).toMatchObject({
      evalCacheEnabled: true,
      evalCacheTTLSeconds: 300,
    });
    await expect(page.getByTestId('save-status')).toHaveText('Saved');
  });

  test('feature cache edit flow can disable the snapshot cache', async ({ page }) => {
    const requests = await recordJsonRequests(page, '**/api/feature-cache/edit', {
      body: { ok: true },
    });

    await mountCacheConfigHarness(page, {
      title: 'Feature cache configuration',
      description: 'Feature snapshot cache is optional and controlled from the front end.',
      saveUrl: 'https://mock.e2e/api/feature-cache/edit',
      saveMethod: 'PUT',
      submitLabel: 'Update feature cache',
      fields: [
        {
          key: 'featureSnapshot',
          label: 'Feature snapshot cache',
          enabledLabel: 'Enable feature snapshot cache',
          ttlLabel: 'Feature snapshot cache TTL (seconds)',
          initialEnabled: true,
          initialTtl: 180,
          payloadEnabledKey: 'evalCacheEnabled',
          payloadTtlKey: 'evalCacheTTLSeconds',
        },
      ],
    });

    const ttlInput = numberInputByLabel(page, 'Feature snapshot cache TTL (seconds)');
    await expect(ttlInput).toHaveValue('180');
    await checkboxByLabel(page, 'Enable feature snapshot cache').uncheck();
    await expect(ttlInput).toBeDisabled();
    await buttonByName(page, 'Update feature cache').click();

    await expect.poll(() => requests.length).toBe(1);
    expect(requests[0]?.method).toBe('PUT');
    expect(requests[0]?.body).toMatchObject({
      evalCacheEnabled: false,
      evalCacheTTLSeconds: 0,
    });
  });

  test('rule cache create flow serializes the external binding cache state', async ({ page }) => {
    const requests = await recordJsonRequests(page, '**/api/rule-cache/create', {
      body: { ok: true },
    });

    await mountCacheConfigHarness(page, {
      title: 'Rule external binding cache',
      description: 'The external API binding cache is opt-in per binding.',
      saveUrl: 'https://mock.e2e/api/rule-cache/create',
      saveMethod: 'POST',
      submitLabel: 'Create rule cache',
      fields: [
        {
          key: 'binding',
          label: 'External API binding cache',
          enabledLabel: 'Enable external API binding cache',
          ttlLabel: 'External API binding cache TTL (seconds)',
          initialEnabled: false,
          initialTtl: 0,
          payloadEnabledKey: 'cacheEnabled',
          payloadTtlKey: 'cacheTTL',
        },
      ],
      wrapper: {
        kind: 'rule',
        externalApiKey: 'payment-validator',
        baseBinding: {
          failMode: 'open',
          paramMappings: [],
        },
      },
    });

    await checkboxByLabel(page, 'Enable external API binding cache').check();
    await numberInputByLabel(page, 'External API binding cache TTL (seconds)').fill('45');
    await buttonByName(page, 'Create rule cache').click();

    await expect.poll(() => requests.length).toBe(1);
    expect(requests[0]?.body).toMatchObject({
      externalApiBindings: [
        {
          externalApiKey: 'payment-validator',
          failMode: 'open',
          paramMappings: [],
          cacheEnabled: true,
          cacheTTL: 45,
        },
      ],
    });
  });

  test('rule cache edit flow can disable the external binding cache', async ({ page }) => {
    const requests = await recordJsonRequests(page, '**/api/rule-cache/edit', {
      body: { ok: true },
    });

    await mountCacheConfigHarness(page, {
      title: 'Rule external binding cache',
      description: 'The external API binding cache is opt-in per binding.',
      saveUrl: 'https://mock.e2e/api/rule-cache/edit',
      saveMethod: 'PUT',
      submitLabel: 'Update rule cache',
      fields: [
        {
          key: 'binding',
          label: 'External API binding cache',
          enabledLabel: 'Enable external API binding cache',
          ttlLabel: 'External API binding cache TTL (seconds)',
          initialEnabled: true,
          initialTtl: 90,
          payloadEnabledKey: 'cacheEnabled',
          payloadTtlKey: 'cacheTTL',
        },
      ],
      wrapper: {
        kind: 'rule',
        externalApiKey: 'payment-validator',
        baseBinding: {
          failMode: 'open',
          paramMappings: [],
        },
      },
    });

    const ttlInput = numberInputByLabel(page, 'External API binding cache TTL (seconds)');
    await expect(ttlInput).toHaveValue('90');
    await checkboxByLabel(page, 'Enable external API binding cache').uncheck();
    await expect(ttlInput).toBeDisabled();
    await buttonByName(page, 'Update rule cache').click();

    await expect.poll(() => requests.length).toBe(1);
    expect(requests[0]?.method).toBe('PUT');
    expect(requests[0]?.body).toMatchObject({
      externalApiBindings: [
        {
          externalApiKey: 'payment-validator',
          failMode: 'open',
          paramMappings: [],
          cacheEnabled: false,
          cacheTTL: 0,
        },
      ],
    });
  });

  test('segment cache flow configures membership and record caches independently', async ({
    page,
  }) => {
    const requests = await recordJsonRequests(page, '**/api/segment-cache/create', {
      body: { ok: true },
    });

    await mountCacheConfigHarness(page, {
      title: 'Segment cache configuration',
      description: 'Segments can cache membership and record lookups independently.',
      saveUrl: 'https://mock.e2e/api/segment-cache/create',
      saveMethod: 'POST',
      submitLabel: 'Create segment cache',
      fields: [
        {
          key: 'membership',
          label: 'Membership cache',
          enabledLabel: 'Enable membership cache',
          ttlLabel: 'Membership cache TTL (seconds)',
          initialEnabled: true,
          initialTtl: 300,
          payloadEnabledKey: 'membershipCacheEnabled',
          payloadTtlKey: 'membershipCacheTTLSeconds',
        },
        {
          key: 'record',
          label: 'Record cache',
          enabledLabel: 'Enable record cache',
          ttlLabel: 'Record cache TTL (seconds)',
          initialEnabled: true,
          initialTtl: 900,
          payloadEnabledKey: 'recordCacheEnabled',
          payloadTtlKey: 'recordCacheTTLSeconds',
        },
      ],
    });

    await buttonByName(page, 'Create segment cache').click();

    await expect.poll(() => requests.length).toBe(1);
    expect(requests[0]?.body).toMatchObject({
      membershipCacheEnabled: true,
      membershipCacheTTLSeconds: 300,
      recordCacheEnabled: true,
      recordCacheTTLSeconds: 900,
    });
  });

  test('experiment cache flow configures the lookup cache', async ({ page }) => {
    const requests = await recordJsonRequests(page, '**/api/experiment-cache/create', {
      body: { ok: true },
    });

    await mountCacheConfigHarness(page, {
      title: 'Experiment cache configuration',
      description: 'Experiment lookups can be cached per experiment.',
      saveUrl: 'https://mock.e2e/api/experiment-cache/create',
      saveMethod: 'POST',
      submitLabel: 'Create experiment cache',
      fields: [
        {
          key: 'lookup',
          label: 'Experiment lookup cache',
          enabledLabel: 'Enable experiment lookup cache',
          ttlLabel: 'Experiment lookup cache TTL (seconds)',
          initialEnabled: false,
          initialTtl: 0,
          payloadEnabledKey: 'lookupCacheEnabled',
          payloadTtlKey: 'lookupCacheTTLSeconds',
        },
      ],
    });

    await checkboxByLabel(page, 'Enable experiment lookup cache').check();
    await numberInputByLabel(page, 'Experiment lookup cache TTL (seconds)').fill('240');
    await buttonByName(page, 'Create experiment cache').click();

    await expect.poll(() => requests.length).toBe(1);
    expect(requests[0]?.body).toMatchObject({
      lookupCacheEnabled: true,
      lookupCacheTTLSeconds: 240,
    });
  });
});

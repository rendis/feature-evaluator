import { expect, test } from '@playwright/test';

import { recordJsonRequests } from './support/network';
import { mountObservabilityHarness } from './support/observability-harness';
import { buttonByName, tabByName } from './support/selectors';

test.describe('feature observability harness', () => {
  test('smoke tests summary, rules and traces interactions', async ({ page }) => {
    const summaryRequests = await recordJsonRequests(page, '**/api/observability/summary', {
      body: {
        totalEvaluations: 128,
        usedRedisRate: 0.75,
        averageDurationMs: 14.2,
        p95DurationMs: 31.8,
      },
    });
    const rulesRequests = await recordJsonRequests(page, '**/api/observability/rules', {
      body: [
        {
          ruleId: 'rule-redis-hit',
          name: 'Redis hit rule',
          cacheStatus: 'hit',
          durationMs: 3.1,
          matched: true,
          expressionCompileCacheHit: true,
          externalCalls: [
            {
              apiKey: 'payment-validator',
              cacheStatus: 'hit',
              durationMs: 12.4,
              httpStatus: 200,
            },
          ],
        },
        {
          ruleId: 'rule-computed',
          name: 'Computed rule',
          cacheStatus: 'computed',
          durationMs: 9.7,
          matched: false,
          expressionCompileCacheHit: false,
          externalCalls: [
            {
              apiKey: 'fraud-validator',
              cacheStatus: 'miss',
              durationMs: 18.9,
              httpStatus: 204,
            },
          ],
        },
      ],
    });
    const tracesRequests = await recordJsonRequests(page, '**/api/observability/traces', {
      body: [
        {
          traceId: 'trace-hit',
          requestId: 'req-redis-hit',
          ruleId: 'rule-redis-hit',
          cacheStatus: 'hit',
          usedRedis: true,
          outcome: 'matched',
          totalDurationMs: 18.5,
          steps: [
            { component: 'feature', cacheStatus: 'hit', durationMs: 1.1 },
            { component: 'rule', cacheStatus: 'hit', durationMs: 3.1 },
          ],
        },
        {
          traceId: 'trace-computed',
          requestId: 'req-computed',
          ruleId: 'rule-computed',
          cacheStatus: 'computed',
          usedRedis: false,
          outcome: 'missed',
          totalDurationMs: 42.9,
          steps: [
            { component: 'feature', cacheStatus: 'computed', durationMs: 2.3 },
            { component: 'rule', cacheStatus: 'computed', durationMs: 9.7 },
          ],
        },
      ],
    });

    await mountObservabilityHarness(page, {
      title: 'Feature observability',
      featureKey: 'checkout-flow',
      summaryUrl: 'https://mock.e2e/api/observability/summary',
      rulesUrl: 'https://mock.e2e/api/observability/rules',
      tracesUrl: 'https://mock.e2e/api/observability/traces',
    });

    await expect.poll(() => summaryRequests.length).toBe(1);
    await expect(page.getByTestId('summary-cards')).toContainText('128');
    await expect(page.getByTestId('summary-cards')).toContainText('75%');
    await expect(page.getByTestId('summary-cards')).toContainText('14.2ms');
    await expect(page.getByTestId('summary-cards')).toContainText('31.8ms');

    await tabByName(page, 'Rules').click();
    await expect.poll(() => rulesRequests.length).toBe(1);
    await expect(page.getByTestId('rules-list')).toContainText('Redis hit rule');
    await expect(page.getByTestId('rules-list')).toContainText('hit');
    await expect(page.getByTestId('rules-list')).toContainText('computed');
    await expect(page.getByTestId('rules-list')).toContainText('compile cache hit');

    await tabByName(page, 'Traces').click();
    await expect.poll(() => tracesRequests.length).toBe(1);
    await expect(page.getByTestId('traces-list')).toContainText('req-redis-hit');
    await expect(page.getByTestId('traces-list')).toContainText('req-computed');

    await buttonByName(page, 'Redis hit').click();
    await expect(page.getByTestId('traces-list').locator('article')).toHaveCount(1);

    await page
      .getByRole('searchbox', { name: 'Search request, rule or trace' })
      .fill('req-redis-hit');
    await expect(page.getByTestId('traces-list').locator('article')).toHaveCount(1);

    await page.getByRole('button', { name: 'Details' }).click();
    await expect(page.getByTestId('trace-details')).toContainText('req-redis-hit');
    await expect(page.getByTestId('trace-details')).toContainText('usedRedis');
  });
});

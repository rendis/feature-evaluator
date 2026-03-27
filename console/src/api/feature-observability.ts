import type {
  FeatureObservabilityCacheBackend,
  FeatureObservabilityCacheStatus,
  FeatureObservabilityOverview,
  FeatureObservabilityRuleDetail,
  FeatureObservabilityRuleExternalCallMetric,
  FeatureObservabilityRuleMetric,
  FeatureObservabilityTrace,
  FeatureObservabilityTraceStep,
  PaginatedResponse,
} from './types';

import { api } from './client';

const BASE = '/admin/features';

export interface FeatureObservabilityTraceParams {
  page?: number;
  pageSize?: number;
  search?: string;
  ruleId?: string;
  usedRedis?: boolean;
  cacheStatus?: FeatureObservabilityCacheStatus | 'all';
}

interface RawObservabilityComponentResponse {
  name: string;
  cacheBackend: FeatureObservabilityCacheBackend;
  cacheEnabled: boolean;
  cacheStatus?: FeatureObservabilityCacheStatus;
  ttlSeconds?: number | null;
  count: number;
  totalDurationMs: number;
  hitCount: number;
  missCount: number;
  disabledCount: number;
  computedCount: number;
  notApplicableCount?: number;
}

interface RawObservabilityOverviewResponse {
  featureKey: string;
  count: number;
  usedRedisCount: number;
  errorCount: number;
  totalDurationMs: number;
  components: RawObservabilityComponentResponse[];
}

interface RawObservabilityExternalCallResponse {
  apiKey: string;
  count: number;
  totalDurationMs: number;
  hitCount: number;
  missCount: number;
  disabledCount: number;
  computedCount: number;
  notApplicableCount?: number;
}

interface RawObservabilityRuleResponse {
  ruleId: string;
  priority: number;
  count: number;
  matchedCount: number;
  totalDurationMs: number;
  expressionDurationMs: number;
  compileCacheHitCount: number;
  externalCalls: RawObservabilityExternalCallResponse[];
}

interface RawObservabilityComponentTraceResponse {
  name: string;
  cacheBackend: FeatureObservabilityCacheBackend;
  cacheEnabled: boolean;
  cacheStatus: FeatureObservabilityCacheStatus;
  ttlSeconds?: number | null;
  durationMs: number;
  outcome?: string;
}

interface RawObservabilityExternalCallTraceResponse {
  apiKey: string;
  durationMs: number;
  cacheStatus: FeatureObservabilityCacheStatus;
  passed: boolean;
  httpStatus?: number | null;
}

interface RawObservabilityRuleTraceResponse {
  ruleId: string;
  priority: number;
  matched: boolean;
  durationMs: number;
  expressionDurationMs: number;
  compileCacheHit: boolean;
  externalCalls: RawObservabilityExternalCallTraceResponse[];
}

interface RawObservabilityTraceResponse {
  id?: string;
  featureKey: string;
  requestId: string;
  environment?: string;
  usedRedis: boolean;
  cacheStatus?: FeatureObservabilityCacheStatus;
  totalDurationMs: number;
  resultReason?: string;
  errorCode?: string;
  createdAt: string;
  components: RawObservabilityComponentTraceResponse[];
  rules: RawObservabilityRuleTraceResponse[];
}

function buildQuery(params: FeatureObservabilityTraceParams = {}) {
  const query = new URLSearchParams();

  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '' || value === 'all') {
      continue;
    }

    query.set(key, String(value));
  }

  const qs = query.toString();
  return qs ? `?${qs}` : '';
}

function averageDuration(totalDurationMs: number, count: number) {
  return count > 0 ? totalDurationMs / count : 0;
}

function deriveCacheStatusFromCounts({
  hitCount,
  missCount,
  disabledCount,
  computedCount,
  notApplicableCount = 0,
}: {
  hitCount: number;
  missCount: number;
  disabledCount: number;
  computedCount: number;
  notApplicableCount?: number;
}): FeatureObservabilityCacheStatus {
  if (hitCount > 0) return 'hit';
  if (missCount > 0) return 'miss';
  if (disabledCount > 0) return 'disabled';
  if (computedCount > 0) return 'computed';
  if (notApplicableCount > 0) return 'not_applicable';
  return 'computed';
}

function deriveRuleCacheStatus(rule: RawObservabilityRuleResponse): FeatureObservabilityCacheStatus {
  const external = aggregateExternalCounts(rule.externalCalls);
  if (external.totalCalls > 0) {
    return deriveCacheStatusFromCounts(external);
  }
  if (rule.compileCacheHitCount > 0) {
    return 'hit';
  }
  return 'computed';
}

function deriveRuleCacheBackend(rule: RawObservabilityRuleResponse): FeatureObservabilityCacheBackend {
  if (rule.externalCalls.length > 0) {
    return 'redis';
  }
  if (rule.compileCacheHitCount > 0) {
    return 'memory';
  }
  return 'none';
}

function aggregateExternalCounts(calls: RawObservabilityExternalCallResponse[]) {
  return calls.reduce(
    (acc, call) => ({
      totalCalls: acc.totalCalls + call.count,
      hitCount: acc.hitCount + call.hitCount,
      missCount: acc.missCount + call.missCount,
      disabledCount: acc.disabledCount + call.disabledCount,
      computedCount: acc.computedCount + call.computedCount,
      notApplicableCount: acc.notApplicableCount + (call.notApplicableCount ?? 0),
    }),
    {
      totalCalls: 0,
      hitCount: 0,
      missCount: 0,
      disabledCount: 0,
      computedCount: 0,
      notApplicableCount: 0,
    },
  );
}

function mapExternalCall(call: RawObservabilityExternalCallResponse): FeatureObservabilityRuleExternalCallMetric {
  const cacheStatus = deriveCacheStatusFromCounts(call);
  const cacheEnabled = call.hitCount > 0 || call.missCount > 0 || (call.disabledCount === 0 && call.count > 0);

  return {
    apiKey: call.apiKey,
    cacheEnabled,
    cacheStatus,
    durationMs: averageDuration(call.totalDurationMs, call.count),
    totalCalls: call.count,
    hitCount: call.hitCount,
    missCount: call.missCount,
    computedCount: call.computedCount,
    ttlSeconds: null,
    passedCount: 0,
    failedCount: 0,
    httpStatus: null,
  };
}

function mapRule(rule: RawObservabilityRuleResponse): FeatureObservabilityRuleMetric {
  return {
    ruleId: rule.ruleId,
    ruleName: rule.ruleId,
    priority: rule.priority,
    matchedCount: rule.matchedCount,
    totalCount: rule.count,
    averageDurationMs: averageDuration(rule.totalDurationMs, rule.count),
    p95DurationMs: 0,
    cacheEnabled: rule.externalCalls.length > 0,
    cacheStatus: deriveRuleCacheStatus(rule),
    cacheBackend: deriveRuleCacheBackend(rule),
    expressionDurationMs: averageDuration(rule.expressionDurationMs, rule.count),
    expressionCompileCacheHitCount: rule.compileCacheHitCount,
    expressionCompileCacheMissCount: Math.max(rule.count - rule.compileCacheHitCount, 0),
    externalCalls: rule.externalCalls.map(mapExternalCall),
  };
}

function mapTraceStep(step: RawObservabilityComponentTraceResponse): FeatureObservabilityTraceStep {
  return {
    component: step.name,
    cacheEnabled: step.cacheEnabled,
    cacheBackend: step.cacheBackend,
    cacheStatus: step.cacheStatus,
    ttlSeconds: step.ttlSeconds ?? null,
    durationMs: step.durationMs,
    outcome: step.outcome,
  };
}

function deriveTraceRule(trace: RawObservabilityTraceResponse) {
  const matchedRule = trace.rules.find((rule) => rule.matched) ?? trace.rules[0];
  return matchedRule
    ? { ruleId: matchedRule.ruleId, ruleName: matchedRule.ruleId, matched: matchedRule.matched }
    : { ruleId: null, ruleName: null, matched: false };
}

function deriveTraceCacheStatus(trace: RawObservabilityTraceResponse): FeatureObservabilityCacheStatus {
  if (trace.cacheStatus) {
    return trace.cacheStatus;
  }
  const steps = trace.components;
  if (steps.some((step) => step.cacheStatus === 'hit')) return 'hit';
  if (steps.some((step) => step.cacheStatus === 'miss')) return 'miss';
  if (steps.some((step) => step.cacheStatus === 'disabled')) return 'disabled';
  if (steps.some((step) => step.cacheStatus === 'computed')) return 'computed';
  return 'not_applicable';
}

function mapTrace(trace: RawObservabilityTraceResponse): FeatureObservabilityTrace {
  const rule = deriveTraceRule(trace);

  return {
    id: trace.id ?? `${trace.requestId || 'trace'}-${trace.createdAt}`,
    requestId: trace.requestId,
    createdAt: trace.createdAt,
    totalDurationMs: trace.totalDurationMs,
    usedRedis: trace.usedRedis,
    cacheStatus: deriveTraceCacheStatus(trace),
    matched: rule.matched,
    ruleId: rule.ruleId,
    ruleName: rule.ruleName,
    reason: trace.resultReason || trace.errorCode || '',
    featureKey: trace.featureKey,
    steps: trace.components.map(mapTraceStep),
  };
}

function mapOverview(raw: RawObservabilityOverviewResponse, ruleCount = 0, matchedRuleCount = 0): FeatureObservabilityOverview {
  return {
    featureKey: raw.featureKey,
    generatedAt: null,
    summary: [],
    components: raw.components.map((component) => ({
      component: component.name,
      cacheEnabled: component.cacheEnabled,
      cacheBackend: component.cacheBackend,
      cacheStatus: component.cacheStatus ?? deriveCacheStatusFromCounts(component),
      ttlSeconds: component.ttlSeconds ?? null,
      totalDurationMs: component.totalDurationMs,
      averageDurationMs: averageDuration(component.totalDurationMs, component.count),
      p95DurationMs: 0,
      hitCount: component.hitCount,
      missCount: component.missCount,
      computedCount: component.computedCount,
      disabledCount: component.disabledCount,
      outcome: undefined,
    })),
    totalEvaluations: raw.count,
    usedRedisCount: raw.usedRedisCount,
    usedRedisRatio: raw.count > 0 ? raw.usedRedisCount / raw.count : 0,
    averageDurationMs: averageDuration(raw.totalDurationMs, raw.count),
    p95DurationMs: 0,
    ruleCount,
    matchedRuleCount,
    slowEvaluations: 0,
  };
}

export const featureObservabilityApi = {
  async overview(featureKey: string) {
    const [overview, rules] = await Promise.all([
      api.get<RawObservabilityOverviewResponse>(`${BASE}/${featureKey}/observability/overview`),
      api.get<{ data: RawObservabilityRuleResponse[] }>(`${BASE}/${featureKey}/observability/rules`),
    ]);

    return mapOverview(
      overview,
      rules.data.length,
      rules.data.reduce((sum, rule) => sum + rule.matchedCount, 0),
    );
  },
  async rules(featureKey: string) {
    const response = await api.get<{ data: RawObservabilityRuleResponse[] }>(
      `${BASE}/${featureKey}/observability/rules`,
    );
    return { data: response.data.map(mapRule) };
  },
  async rule(featureKey: string, ruleId: string) {
    const [ruleResponse, tracesResponse] = await Promise.all([
      api.get<RawObservabilityRuleResponse>(`${BASE}/${featureKey}/observability/rules/${ruleId}`),
      api.get<PaginatedResponse<RawObservabilityTraceResponse>>(
        `${BASE}/${featureKey}/observability/traces${buildQuery({ ruleId, page: 1, pageSize: 10 })}`,
      ),
    ]);

    return {
      ...mapRule(ruleResponse),
      generatedAt: null,
      traces: tracesResponse.data.map(mapTrace),
    } satisfies FeatureObservabilityRuleDetail;
  },
  async traces(featureKey: string, params: FeatureObservabilityTraceParams = {}) {
    const response = await api.get<PaginatedResponse<RawObservabilityTraceResponse>>(
      `${BASE}/${featureKey}/observability/traces${buildQuery(params)}`,
    );

    return {
      data: response.data.map(mapTrace),
      pagination: response.pagination,
    };
  },
};

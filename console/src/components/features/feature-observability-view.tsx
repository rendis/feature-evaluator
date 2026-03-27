import { useQuery } from '@tanstack/react-query';
import { AlertTriangle, BarChart3, Clock3, Database, ShieldCheck, ToggleLeft } from 'lucide-react';
import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';

import type {
  Feature,
  FeatureObservabilityCacheStatus,
  FeatureObservabilityOverview,
  FeatureObservabilityRuleDetail,
  FeatureObservabilityRuleMetric,
  FeatureObservabilityTrace,
} from '@/api/types';

import { featureObservabilityQueries } from '@/queries/feature-observability-queries';
import { useDebounce } from '@/hooks/use-debounce';
import { useLocaleFormatters } from '@/hooks/use-locale-formatters';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { EmptyState } from '@/components/shared/empty-state';
import { ApiErrorState } from '@/components/shared/error-state';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Skeleton } from '@/components/ui/skeleton';

interface FeatureObservabilityViewProps {
  feature: Feature;
  featureKey: string;
}

interface TraceFiltersState {
  search: string;
  ruleId: string;
  cacheStatus: FeatureObservabilityCacheStatus | 'all';
  usedRedis: 'all' | 'true' | 'false';
  page: number;
  pageSize: number;
}

const EMPTY_OVERVIEW: FeatureObservabilityOverview = {
  featureKey: '',
  summary: [],
  components: [],
  totalEvaluations: 0,
  usedRedisCount: 0,
  usedRedisRatio: 0,
  averageDurationMs: 0,
  p95DurationMs: 0,
  ruleCount: 0,
  matchedRuleCount: 0,
  slowEvaluations: 0,
};

export function FeatureObservabilityView({ feature, featureKey }: FeatureObservabilityViewProps) {
  const { t } = useTranslation('features');
  const overviewQuery = useQuery(featureObservabilityQueries.overview(featureKey));
  const rulesQuery = useQuery(featureObservabilityQueries.rules(featureKey));
  const [selectedRuleId, setSelectedRuleId] = useState<string | null>(null);
  const [filters, setFilters] = useState<TraceFiltersState>({
    search: '',
    ruleId: 'all',
    cacheStatus: 'all',
    usedRedis: 'all',
    page: 1,
    pageSize: 10,
  });

  const debouncedSearch = useDebounce(filters.search, 300);
  const tracesQuery = useQuery(
    featureObservabilityQueries.traces(featureKey, {
      page: filters.page,
      pageSize: filters.pageSize,
      search: debouncedSearch.trim() || undefined,
      ruleId: filters.ruleId === 'all' ? undefined : filters.ruleId,
      cacheStatus: filters.cacheStatus,
      usedRedis: filters.usedRedis === 'all' ? undefined : filters.usedRedis === 'true',
    }),
  );

  const selectedRuleQuery = useQuery({
    ...featureObservabilityQueries.rule(featureKey, selectedRuleId ?? ''),
    enabled: !!selectedRuleId,
  });

  const overview = overviewQuery.data ?? EMPTY_OVERVIEW;
  const rules = useMemo(() => {
    const ruleNameById = new Map((feature.rules ?? []).map((rule) => [rule.id, rule.name]));
    return (rulesQuery.data?.data ?? []).map((rule) => ({
      ...rule,
      ruleName: ruleNameById.get(rule.ruleId) ?? rule.ruleName,
    }));
  }, [feature.rules, rulesQuery.data?.data]);
  const traces = useMemo(() => {
    const ruleNameById = new Map((feature.rules ?? []).map((rule) => [rule.id, rule.name]));
    return (tracesQuery.data?.data ?? []).map((trace) => ({
      ...trace,
      ruleName: trace.ruleId ? ruleNameById.get(trace.ruleId) ?? trace.ruleName : trace.ruleName,
    }));
  }, [feature.rules, tracesQuery.data?.data]);
  const pagination = tracesQuery.data?.pagination;

  const traceRuleOptions = useMemo(
    () => rules.map((rule) => ({ value: rule.ruleId, label: `${rule.priority}. ${rule.ruleName}` })),
    [rules],
  );

  const selectedRule = useMemo(() => {
    const rule = selectedRuleQuery.data ?? null;
    if (!rule) {
      return null;
    }
    const ruleNameById = new Map((feature.rules ?? []).map((entry) => [entry.id, entry.name]));
    return {
      ...rule,
      ruleName: ruleNameById.get(rule.ruleId) ?? rule.ruleName,
      traces: rule.traces.map((trace) => ({
        ...trace,
        ruleName: trace.ruleId ? ruleNameById.get(trace.ruleId) ?? trace.ruleName : trace.ruleName,
      })),
    };
  }, [feature.rules, selectedRuleQuery.data]);
  const selectedRuleError = selectedRuleQuery.error;

  const onFilterChange = (patch: Partial<TraceFiltersState>) => {
    setFilters((current) => ({
      ...current,
      ...patch,
      page: patch.page ?? 1,
    }));
  };

  if (overviewQuery.isLoading || rulesQuery.isLoading || tracesQuery.isLoading) {
    return <FeatureObservabilitySkeleton />;
  }

  const queryError = overviewQuery.error ?? rulesQuery.error ?? tracesQuery.error;
  if (queryError) {
    return <ApiErrorState error={queryError} />;
  }

  return (
    <div className="space-y-8">
      <section className="space-y-4">
        <div className="flex items-center gap-2">
          <BarChart3 className="text-primary h-5 w-5" />
          <div>
            <h2 className="text-lg font-semibold">{t('observability.title', { defaultValue: 'Observability' })}</h2>
            <p className="text-muted-foreground text-sm">
              {feature.name} · {feature.key}
            </p>
          </div>
        </div>

        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          {overview.summary.length > 0
            ? overview.summary.map((card) => (
                <SummaryCard key={card.key} card={card} />
              ))
            : defaultSummaryCards(overview).map((card) => <SummaryCard key={card.key} card={card} />)}
        </div>
      </section>

      <section className="space-y-4">
        <div className="flex items-center gap-2">
          <ToggleLeft className="text-primary h-5 w-5" />
          <h3 className="text-lg font-semibold">
            {t('observability.rules.title', { defaultValue: 'Rules' })}
          </h3>
        </div>

        {rules.length === 0 ? (
          <EmptyState
            icon={<AlertTriangle className="h-10 w-10" />}
            title={t('observability.rules.empty.title', { defaultValue: 'No rules metrics yet' })}
            description={t('observability.rules.empty.description', {
              defaultValue: 'Run evaluations to see latency, hit ratio and external call metrics here.',
            })}
          />
        ) : (
          <RulesTable rules={rules} onSelectRule={setSelectedRuleId} />
        )}
      </section>

      <section className="space-y-4">
        <div className="flex items-center gap-2">
          <Clock3 className="text-primary h-5 w-5" />
          <h3 className="text-lg font-semibold">
            {t('observability.traces.title', { defaultValue: 'Traces' })}
          </h3>
        </div>

        <TraceFilters
          featureKey={featureKey}
          filters={filters}
          traceRuleOptions={traceRuleOptions}
          onChange={onFilterChange}
        />

        {traces.length === 0 ? (
          <EmptyState
            icon={<Database className="h-10 w-10" />}
            title={t('observability.traces.empty.title', { defaultValue: 'No traces yet' })}
            description={t('observability.traces.empty.description', {
              defaultValue: 'Filtered traces will appear here once evaluations are recorded.',
            })}
          />
        ) : (
          <>
            <TracesTable traces={traces} onSelectRule={setSelectedRuleId} />
            {pagination && pagination.totalPages > 1 ? (
              <TracePagination
                page={pagination.page}
                totalPages={pagination.totalPages}
                onPageChange={(page) => onFilterChange({ page })}
              />
            ) : null}
          </>
        )}
      </section>

      <RuleDetailDrawer
        rule={selectedRule}
        error={selectedRuleError}
        open={selectedRuleId != null}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedRuleId(null);
          }
        }}
      />
    </div>
  );
}

function SummaryCard({
  card,
}: {
  card: { key: string; label: string; value: number; unit?: 'count' | 'ms' | 'ratio'; accent?: string };
}) {
  const { formatNumber } = useLocaleFormatters();

  const value =
    card.unit === 'ratio'
      ? formatNumber(card.value / 100, { style: 'percent', maximumFractionDigits: 1 })
      : card.unit === 'ms'
        ? `${formatNumber(card.value)} ms`
        : formatNumber(card.value);

  const accentClass =
    card.accent === 'success'
      ? 'text-success'
      : card.accent === 'warning'
        ? 'text-amber-600'
        : card.accent === 'destructive'
          ? 'text-destructive'
          : 'text-foreground';

  return (
    <div className="rounded-2xl border bg-card p-5 shadow-sm">
      <p className="text-muted-foreground text-sm font-medium">{card.label}</p>
      <p className={`mt-1 text-2xl font-bold tracking-tight ${accentClass}`}>{value}</p>
    </div>
  );
}

function defaultSummaryCards(overview: FeatureObservabilityOverview) {
  return [
    {
      key: 'totalEvaluations',
      label: 'Total evaluations',
      value: overview.totalEvaluations,
      unit: 'count' as const,
    },
    {
      key: 'usedRedisRatio',
      label: 'Redis usage',
      value: overview.usedRedisRatio * 100,
      unit: 'ratio' as const,
      accent: overview.usedRedisRatio > 0.5 ? 'success' : 'warning',
    },
    {
      key: 'averageDurationMs',
      label: 'Average latency',
      value: overview.averageDurationMs,
      unit: 'ms' as const,
    },
    {
      key: 'p95DurationMs',
      label: 'p95 latency',
      value: overview.p95DurationMs,
      unit: 'ms' as const,
    },
  ];
}

function RulesTable({
  rules,
  onSelectRule,
}: {
  rules: FeatureObservabilityRuleMetric[];
  onSelectRule: (ruleId: string) => void;
}) {
  const { formatNumber } = useLocaleFormatters();

  return (
    <div className="overflow-x-auto rounded-2xl border bg-card shadow-sm">
      <table className="w-full text-left">
        <thead className="border-b bg-muted/50">
          <tr>
            <th className="px-4 py-3 text-xs font-medium">Rule</th>
            <th className="px-4 py-3 text-xs font-medium">Priority</th>
            <th className="px-4 py-3 text-xs font-medium">Matched</th>
            <th className="px-4 py-3 text-xs font-medium">Duration</th>
            <th className="px-4 py-3 text-xs font-medium">Cache</th>
            <th className="px-4 py-3 text-xs font-medium">Compile cache</th>
          </tr>
        </thead>
        <tbody>
          {rules.map((rule) => (
            <tr
              key={rule.ruleId}
              className="cursor-pointer border-b last:border-0 hover:bg-muted/30"
              onClick={() => onSelectRule(rule.ruleId)}
            >
              <td className="px-4 py-3">
                <div className="space-y-1">
                  <div className="font-medium">{rule.ruleName}</div>
                  <div className="text-muted-foreground font-mono text-xs">{rule.ruleId}</div>
                </div>
              </td>
              <td className="px-4 py-3 text-sm">{rule.priority}</td>
              <td className="px-4 py-3 text-sm">
                {formatNumber(rule.matchedCount)} / {formatNumber(rule.totalCount)}
              </td>
              <td className="px-4 py-3 text-sm">
                {formatNumber(rule.averageDurationMs)} ms
                <div className="text-muted-foreground text-xs">p95 {formatNumber(rule.p95DurationMs)} ms</div>
              </td>
              <td className="px-4 py-3 text-sm">
                <Badge variant={rule.cacheEnabled ? 'default' : 'outline'}>
                  {rule.cacheEnabled ? rule.cacheStatus : 'disabled'}
                </Badge>
              </td>
              <td className="px-4 py-3 text-sm">
                {formatNumber(rule.expressionCompileCacheHitCount)} hits · {formatNumber(rule.expressionCompileCacheMissCount)} misses
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function RuleDetailDrawer({
  rule,
  error,
  open,
  onOpenChange,
}: {
  rule: FeatureObservabilityRuleDetail | null;
  error: unknown;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { formatNumber } = useLocaleFormatters();
  const { t } = useTranslation('features');

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] max-w-4xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{rule?.ruleName ?? t('observability.ruleDrawer.title', { defaultValue: 'Rule details' })}</DialogTitle>
          <DialogDescription className="font-mono text-xs">{rule?.ruleId}</DialogDescription>
        </DialogHeader>

        {error ? (
          <ApiErrorState error={error} />
        ) : rule ? (
          <div className="space-y-6">
            <div className="grid gap-3 md:grid-cols-3">
              <InfoCard label="Priority" value={rule.priority} />
              <InfoCard label="Matched" value={`${formatNumber(rule.matchedCount)} / ${formatNumber(rule.totalCount)}`} />
              <InfoCard label="Expression" value={`${formatNumber(rule.expressionDurationMs)} ms`} />
            </div>

            <div className="rounded-2xl border bg-muted/10 p-4">
              <div className="mb-3 flex items-center gap-2">
                <ShieldCheck className="text-primary h-4 w-4" />
                <h4 className="font-semibold">
                  {t('observability.ruleDrawer.externalCalls', { defaultValue: 'External calls' })}
                </h4>
              </div>
              {rule.externalCalls.length === 0 ? (
                <p className="text-muted-foreground text-sm">
                  {t('observability.ruleDrawer.noExternalCalls', {
                    defaultValue: 'This rule does not call external validators.',
                  })}
                </p>
              ) : (
                <div className="space-y-3">
                  {rule.externalCalls.map((call) => (
                    <div key={call.apiKey} className="rounded-xl border bg-background p-3">
                      <div className="flex flex-wrap items-center gap-2">
                        <Badge variant={call.cacheEnabled ? 'default' : 'outline'}>{call.apiKey}</Badge>
                        <span className="text-muted-foreground text-sm">{call.cacheEnabled ? call.cacheStatus : 'disabled'}</span>
                        <span className="text-muted-foreground text-sm">{formatNumber(call.durationMs)} ms</span>
                      </div>
                      <div className="text-muted-foreground mt-2 text-xs">
                        {formatNumber(call.totalCalls)} calls · {formatNumber(call.hitCount)} hits · {formatNumber(call.missCount)} misses
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div className="rounded-2xl border bg-muted/10 p-4">
              <h4 className="mb-3 font-semibold">
                {t('observability.ruleDrawer.traces', { defaultValue: 'Recent traces' })}
              </h4>
              {rule.traces.length === 0 ? (
                <p className="text-muted-foreground text-sm">
                  {t('observability.ruleDrawer.noTraces', {
                    defaultValue: 'No traces recorded for this rule yet.',
                  })}
                </p>
              ) : (
                <div className="space-y-2">
                  {rule.traces.map((trace) => (
                    <TracePreview key={trace.id} trace={trace} />
                  ))}
                </div>
              )}
            </div>
          </div>
        ) : (
          <div className="py-10 text-center text-sm text-muted-foreground">
            {t('observability.ruleDrawer.loading', { defaultValue: 'Loading rule details…' })}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}

function TracePreview({ trace }: { trace: FeatureObservabilityTrace }) {
  const { formatNumber, formatDateTime } = useLocaleFormatters();

  return (
    <div className="rounded-xl border bg-background p-3">
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={trace.matched ? 'default' : 'outline'}>
          {trace.matched ? 'matched' : 'missed'}
        </Badge>
        <span className="text-muted-foreground text-sm font-mono">{trace.requestId}</span>
        <span className="text-muted-foreground text-sm">{formatDateTime(trace.createdAt)}</span>
        <span className="text-muted-foreground text-sm">{formatNumber(trace.totalDurationMs)} ms</span>
        <span className="text-muted-foreground text-sm">{trace.usedRedis ? 'redis' : 'no redis'}</span>
      </div>
      {trace.reason ? <p className="text-muted-foreground mt-2 text-sm">{trace.reason}</p> : null}
    </div>
  );
}

function TraceFilters({
  featureKey,
  filters,
  traceRuleOptions,
  onChange,
}: {
  featureKey: string;
  filters: TraceFiltersState;
  traceRuleOptions: { value: string; label: string }[];
  onChange: (patch: Partial<TraceFiltersState>) => void;
}) {
  const { t } = useTranslation('features');

  return (
    <div className="grid gap-3 rounded-2xl border bg-card p-4 shadow-sm lg:grid-cols-4">
      <div className="space-y-2">
        <Label htmlFor={`trace-search-${featureKey}`}>Search</Label>
        <Input
          id={`trace-search-${featureKey}`}
          value={filters.search}
          onChange={(event) => onChange({ search: event.target.value })}
          placeholder={t('observability.traces.filters.search', { defaultValue: 'Search request id…' })}
        />
      </div>
      <div className="space-y-2">
        <Label>{t('observability.traces.filters.rule', { defaultValue: 'Rule' })}</Label>
        <Select
          value={filters.ruleId}
          onValueChange={(value) => onChange({ ruleId: value })}
        >
          <SelectTrigger>
            <SelectValue placeholder={t('observability.traces.filters.all', { defaultValue: 'All' })} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t('observability.traces.filters.all', { defaultValue: 'All' })}</SelectItem>
            {traceRuleOptions.map((rule) => (
              <SelectItem key={rule.value} value={rule.value}>
                {rule.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-2">
        <Label>{t('observability.traces.filters.cacheStatus', { defaultValue: 'Cache status' })}</Label>
        <Select
          value={filters.cacheStatus}
          onValueChange={(value) => onChange({ cacheStatus: value as TraceFiltersState['cacheStatus'] })}
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t('observability.traces.filters.all', { defaultValue: 'All' })}</SelectItem>
            <SelectItem value="hit">hit</SelectItem>
            <SelectItem value="miss">miss</SelectItem>
            <SelectItem value="disabled">disabled</SelectItem>
            <SelectItem value="computed">computed</SelectItem>
            <SelectItem value="not_applicable">not_applicable</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-2">
        <Label>{t('observability.traces.filters.redis', { defaultValue: 'Used Redis' })}</Label>
        <Select
          value={filters.usedRedis}
          onValueChange={(value) => onChange({ usedRedis: value as TraceFiltersState['usedRedis'] })}
        >
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t('observability.traces.filters.all', { defaultValue: 'All' })}</SelectItem>
            <SelectItem value="true">redis</SelectItem>
            <SelectItem value="false">no redis</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}

function TracesTable({
  traces,
  onSelectRule,
}: {
  traces: FeatureObservabilityTrace[];
  onSelectRule: (ruleId: string) => void;
}) {
  const { formatNumber, formatDateTime } = useLocaleFormatters();

  return (
    <div className="overflow-x-auto rounded-2xl border bg-card shadow-sm">
      <table className="w-full text-left">
        <thead className="border-b bg-muted/50">
          <tr>
            <th className="px-4 py-3 text-xs font-medium">Request</th>
            <th className="px-4 py-3 text-xs font-medium">Rule</th>
            <th className="px-4 py-3 text-xs font-medium">Duration</th>
            <th className="px-4 py-3 text-xs font-medium">Cache</th>
            <th className="px-4 py-3 text-xs font-medium">Redis</th>
            <th className="px-4 py-3 text-xs font-medium">Date</th>
          </tr>
        </thead>
        <tbody>
          {traces.map((trace) => (
            <tr
              key={trace.id}
              className="border-b last:border-0 hover:bg-muted/30"
            >
              <td className="px-4 py-3 text-sm font-mono">{trace.requestId}</td>
              <td className="px-4 py-3 text-sm">
                <button type="button" className="text-primary hover:underline" onClick={() => trace.ruleId && onSelectRule(trace.ruleId)}>
                  {trace.ruleName ?? trace.ruleId ?? '—'}
                </button>
              </td>
              <td className="px-4 py-3 text-sm">{formatNumber(trace.totalDurationMs)} ms</td>
              <td className="px-4 py-3 text-sm">
                <Badge variant="outline">{trace.cacheStatus}</Badge>
              </td>
              <td className="px-4 py-3 text-sm">
                <Badge variant={trace.usedRedis ? 'default' : 'outline'}>
                  {trace.usedRedis ? 'yes' : 'no'}
                </Badge>
              </td>
              <td className="px-4 py-3 text-sm">{formatDateTime(trace.createdAt)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function TracePagination({
  page,
  totalPages,
  onPageChange,
}: {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center justify-center gap-4">
      <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => onPageChange(page - 1)}>
        {t('pagination.previous')}
      </Button>
      <span className="text-muted-foreground text-sm">
        {t('pagination.page')} {page} {t('pagination.of')} {totalPages}
      </span>
      <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => onPageChange(page + 1)}>
        {t('pagination.next')}
      </Button>
    </div>
  );
}

function InfoCard({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="rounded-xl border bg-background p-4">
      <p className="text-muted-foreground text-xs uppercase tracking-wide">{label}</p>
      <p className="mt-1 text-base font-semibold">{value}</p>
    </div>
  );
}

function FeatureObservabilitySkeleton() {
  return (
    <div className="space-y-8">
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }).map((_, index) => (
          <Skeleton key={index} className="h-28 rounded-2xl" />
        ))}
      </div>
      <Skeleton className="h-72 rounded-2xl" />
      <Skeleton className="h-72 rounded-2xl" />
    </div>
  );
}

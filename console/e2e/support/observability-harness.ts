import type { Page } from '@playwright/test';

export interface ObservabilityHarnessConfig {
  title: string;
  featureKey: string;
  summaryUrl: string;
  rulesUrl: string;
  tracesUrl: string;
}

export async function mountObservabilityHarness(page: Page, config: ObservabilityHarnessConfig) {
  await page.setContent('<!doctype html><html><body><main id="root"></main></body></html>');

  await page.evaluate((harness) => {
    type BrowserConfig = ObservabilityHarnessConfig;

    const config = harness as BrowserConfig;
    const root = document.getElementById('root');
    if (!root) {
      throw new Error('Harness root is missing');
    }

    root.innerHTML = '';
    root.className = 'min-h-screen bg-background p-6 text-foreground';

    const shell = document.createElement('section');
    shell.className = 'mx-auto max-w-6xl space-y-6 rounded-2xl border border-border bg-card p-6';

    const header = document.createElement('header');
    header.className = 'space-y-2';

    const title = document.createElement('h1');
    title.textContent = config.title;
    title.className = 'text-2xl font-semibold';
    header.appendChild(title);

    const subtitle = document.createElement('p');
    subtitle.textContent = config.featureKey;
    subtitle.className = 'text-muted-foreground text-sm';
    header.appendChild(subtitle);

    shell.appendChild(header);

    const tablist = document.createElement('div');
    tablist.setAttribute('role', 'tablist');
    tablist.className = 'flex gap-2';

    const summaryTab = createTab('Summary', true);
    const rulesTab = createTab('Rules');
    const tracesTab = createTab('Traces');

    tablist.append(summaryTab.button, rulesTab.button, tracesTab.button);
    shell.appendChild(tablist);

    const status = document.createElement('div');
    status.dataset.testid = 'observability-status';
    status.setAttribute('role', 'status');
    status.className = 'text-sm';
    shell.appendChild(status);

    const panels = {
      summary: document.createElement('section'),
      rules: document.createElement('section'),
      traces: document.createElement('section'),
    };

    panels.summary.dataset.testid = 'observability-summary';
    panels.rules.dataset.testid = 'observability-rules';
    panels.traces.dataset.testid = 'observability-traces';

    const summaryCards = document.createElement('div');
    summaryCards.className = 'grid gap-3 md:grid-cols-2 xl:grid-cols-4';
    summaryCards.dataset.testid = 'summary-cards';
    panels.summary.appendChild(summaryCards);

    const rulesList = document.createElement('div');
    rulesList.className = 'space-y-3';
    rulesList.dataset.testid = 'rules-list';
    panels.rules.appendChild(rulesList);

    const tracesToolbar = document.createElement('div');
    tracesToolbar.className = 'flex flex-wrap items-center gap-3';
    const traceSearch = document.createElement('input');
    traceSearch.type = 'search';
    traceSearch.placeholder = 'Search request, rule or trace';
    traceSearch.setAttribute('aria-label', 'Search request, rule or trace');
    traceSearch.className =
      'h-10 min-w-72 rounded-md border border-input bg-background px-3 text-sm';
    tracesToolbar.appendChild(traceSearch);

    const traceFilters = document.createElement('div');
    traceFilters.className = 'flex flex-wrap gap-2';
    const traceFilterButtons = new Map<string, HTMLButtonElement>();
    for (const [filter, label] of [
      ['all', 'All'],
      ['hit', 'Redis hit'],
      ['miss', 'Redis miss'],
      ['computed', 'Computed'],
      ['disabled', 'Disabled'],
    ] as const) {
      const button = createFilterButton(label, filter === 'all');
      traceFilterButtons.set(filter, button);
      traceFilters.appendChild(button);
    }
    tracesToolbar.appendChild(traceFilters);
    panels.traces.appendChild(tracesToolbar);

    const tracesTable = document.createElement('div');
    tracesTable.className = 'space-y-2';
    tracesTable.dataset.testid = 'traces-list';
    panels.traces.appendChild(tracesTable);

    const detailsPanel = document.createElement('div');
    detailsPanel.dataset.testid = 'trace-details';
    detailsPanel.className = 'rounded-xl border border-border/70 bg-muted/15 p-4 text-sm';
    panels.traces.appendChild(detailsPanel);

    type ObservabilityTab = 'summary' | 'rules' | 'traces';
    let currentTraces: TraceRow[] = [];
    let currentFilter: 'all' | 'hit' | 'miss' | 'computed' | 'disabled' = 'all';
    let rulesLoaded = false;
    let tracesLoaded = false;

    const summaryData = {
      totalEvaluations: 0,
      usedRedisRate: 0,
      averageDurationMs: 0,
      p95DurationMs: 0,
    };
    let rulesData: RuleRow[] = [];

    void loadSummary();

    summaryTab.button.addEventListener('click', () => switchTab('summary'));
    rulesTab.button.addEventListener('click', () => switchTab('rules'));
    tracesTab.button.addEventListener('click', () => switchTab('traces'));
    traceSearch.addEventListener('input', () => renderTraces());
    traceFilterButtons.forEach((button, filter) => {
      button.addEventListener('click', () => {
        currentFilter = filter as typeof currentFilter;
        traceFilterButtons.forEach((item, itemFilter) => {
          item.dataset.active = String(itemFilter === currentFilter);
        });
        renderTraces();
      });
    });

    root.appendChild(shell);
    shell.append(panels.summary, panels.rules, panels.traces);
    switchTab('summary');

    async function switchTab(nextTab: ObservabilityTab) {
      summaryTab.button.dataset.active = String(nextTab === 'summary');
      rulesTab.button.dataset.active = String(nextTab === 'rules');
      tracesTab.button.dataset.active = String(nextTab === 'traces');
      panels.summary.hidden = nextTab !== 'summary';
      panels.rules.hidden = nextTab !== 'rules';
      panels.traces.hidden = nextTab !== 'traces';

      if (nextTab === 'rules' && !rulesLoaded) {
        await loadRules();
      }

      if (nextTab === 'traces' && !tracesLoaded) {
        await loadTraces();
      }
    }

    async function loadSummary() {
      status.textContent = 'Loading summary...';
      const response = await fetch(config.summaryUrl);
      const data = (await response.json()) as Partial<typeof summaryData>;
      Object.assign(summaryData, data);
      status.textContent = 'Summary loaded';
      renderSummary();
    }

    async function loadRules() {
      status.textContent = 'Loading rules...';
      const response = await fetch(config.rulesUrl);
      rulesData = (await response.json()) as RuleRow[];
      rulesLoaded = true;
      status.textContent = 'Rules loaded';
      renderRules();
    }

    async function loadTraces() {
      status.textContent = 'Loading traces...';
      const response = await fetch(config.tracesUrl);
      currentTraces = (await response.json()) as TraceRow[];
      tracesLoaded = true;
      status.textContent = 'Traces loaded';
      renderTraces();
    }

    function renderSummary() {
      summaryCards.innerHTML = '';
      for (const card of [
        { label: 'Total evaluations', value: String(summaryData.totalEvaluations) },
        { label: 'Used Redis rate', value: formatPercent(summaryData.usedRedisRate) },
        { label: 'Average duration', value: formatNumber(summaryData.averageDurationMs) },
        { label: 'P95 duration', value: formatNumber(summaryData.p95DurationMs) },
      ]) {
        const article = document.createElement('article');
        article.className = 'rounded-xl border border-border/70 bg-muted/15 p-4';

        const label = document.createElement('p');
        label.textContent = card.label;
        label.className = 'text-muted-foreground text-xs uppercase tracking-wide';

        const value = document.createElement('p');
        value.textContent = card.value;
        value.className = 'mt-2 text-xl font-semibold';

        article.append(label, value);
        summaryCards.appendChild(article);
      }
    }

    function renderRules() {
      rulesList.innerHTML = '';
      for (const rule of rulesData) {
        const article = document.createElement('article');
        article.dataset.ruleId = rule.ruleId;
        article.className = 'space-y-3 rounded-xl border border-border/70 bg-muted/10 p-4';

        const heading = document.createElement('div');
        heading.className = 'flex flex-wrap items-center gap-2';

        const title = document.createElement('h3');
        title.textContent = rule.name;
        title.className = 'font-medium';
        heading.appendChild(title);

        const badge = document.createElement('span');
        badge.textContent = rule.cacheStatus;
        badge.className = 'rounded-full border border-border/70 px-2 py-0.5 text-xs';
        heading.appendChild(badge);

        article.appendChild(heading);

        const metrics = document.createElement('p');
        metrics.textContent = `${rule.ruleId} · ${formatNumber(rule.durationMs)} · compile cache ${rule.expressionCompileCacheHit ? 'hit' : 'miss'} · ${rule.matched ? 'matched' : 'not matched'}`;
        metrics.className = 'text-muted-foreground text-sm';
        article.appendChild(metrics);

        const calls = document.createElement('div');
        calls.className = 'space-y-2';
        for (const call of rule.externalCalls ?? []) {
          const row = document.createElement('div');
          row.className = 'rounded-lg border border-border/60 bg-background/60 px-3 py-2 text-sm';
          row.textContent = `${call.apiKey} · ${call.cacheStatus} · ${formatNumber(call.durationMs)} · ${call.httpStatus}`;
          calls.appendChild(row);
        }
        article.appendChild(calls);

        rulesList.appendChild(article);
      }
    }

    function renderTraces() {
      tracesTable.innerHTML = '';
      const query = traceSearch.value.trim().toLowerCase();
      const filtered = currentTraces.filter((trace) => {
        const matchesFilter = currentFilter === 'all' ? true : trace.cacheStatus === currentFilter;
        const matchesQuery =
          query.length === 0
            ? true
            : [trace.requestId, trace.ruleId, trace.traceId, trace.outcome]
                .join(' ')
                .toLowerCase()
                .includes(query);
        return matchesFilter && matchesQuery;
      });

      if (filtered.length === 0) {
        const empty = document.createElement('p');
        empty.className = 'text-muted-foreground text-sm';
        empty.textContent = 'No traces match the current filters.';
        tracesTable.appendChild(empty);
        detailsPanel.textContent = '';
        return;
      }

      for (const trace of filtered) {
        const article = document.createElement('article');
        article.dataset.requestId = trace.requestId;
        article.className = 'rounded-xl border border-border/70 bg-muted/10 p-4';

        const header = document.createElement('div');
        header.className = 'flex flex-wrap items-center justify-between gap-3';

        const left = document.createElement('div');
        const requestId = document.createElement('h3');
        requestId.textContent = trace.requestId;
        requestId.className = 'font-medium';
        left.appendChild(requestId);

        const meta = document.createElement('p');
        meta.textContent = `${trace.ruleId} · ${trace.cacheStatus} · ${trace.outcome} · usedRedis=${trace.usedRedis}`;
        meta.className = 'text-muted-foreground text-sm';
        left.appendChild(meta);
        header.appendChild(left);

        const expand = document.createElement('button');
        expand.type = 'button';
        expand.textContent = 'Details';
        expand.className = 'rounded-md border border-border/70 px-3 py-1 text-sm';
        header.appendChild(expand);

        article.appendChild(header);

        const body = document.createElement('div');
        body.hidden = true;
        body.className = 'mt-3 space-y-2 rounded-lg border border-border/60 bg-background/70 p-3';
        body.textContent = trace.steps
          .map((step) => `${step.component}:${step.cacheStatus}:${formatNumber(step.durationMs)}`)
          .join(' | ');
        article.appendChild(body);

        expand.addEventListener('click', () => {
          body.hidden = !body.hidden;
          detailsPanel.textContent = JSON.stringify(trace, null, 2);
        });

        tracesTable.appendChild(article);
      }
    }

    function createTab(label: string, selected = false) {
      const button = document.createElement('button');
      button.type = 'button';
      button.setAttribute('role', 'tab');
      button.textContent = label;
      button.className = 'rounded-md border border-border/70 px-3 py-2 text-sm';
      button.dataset.active = String(selected);
      return { button };
    }

    function createFilterButton(label: string, selected = false) {
      const button = document.createElement('button');
      button.type = 'button';
      button.textContent = label;
      button.className = 'rounded-md border border-border/70 px-3 py-1.5 text-sm';
      button.dataset.active = String(selected);
      return button;
    }

    function formatNumber(value: number) {
      return Number.isFinite(value) ? `${value.toFixed(1)}ms` : '0.0ms';
    }

    function formatPercent(value: number) {
      return `${Math.round(value * 100)}%`;
    }
  }, config);
}

interface RuleRow {
  ruleId: string;
  name: string;
  cacheStatus: 'hit' | 'miss' | 'computed' | 'disabled' | 'not_applicable';
  durationMs: number;
  matched: boolean;
  expressionCompileCacheHit: boolean;
  externalCalls?: {
    apiKey: string;
    cacheStatus: 'hit' | 'miss' | 'computed' | 'disabled';
    durationMs: number;
    httpStatus: number;
  }[];
}

interface TraceRow {
  traceId: string;
  requestId: string;
  ruleId: string;
  cacheStatus: 'hit' | 'miss' | 'computed' | 'disabled';
  usedRedis: boolean;
  outcome: string;
  totalDurationMs: number;
  steps: {
    component: string;
    cacheStatus: string;
    durationMs: number;
  }[];
}

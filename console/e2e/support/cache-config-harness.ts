import type { Page } from '@playwright/test';

export interface CacheFieldHarnessConfig {
  key: string;
  label: string;
  enabledLabel: string;
  ttlLabel: string;
  initialEnabled: boolean;
  initialTtl: number;
  payloadEnabledKey: string;
  payloadTtlKey: string;
}

export interface CacheConfigHarnessConfig {
  title: string;
  description?: string;
  saveUrl: string;
  saveMethod?: 'POST' | 'PUT' | 'PATCH';
  submitLabel?: string;
  fields: CacheFieldHarnessConfig[];
  wrapper?: {
    kind: 'rule';
    externalApiKey: string;
    baseBinding?: Record<string, unknown>;
  };
}

export async function mountCacheConfigHarness(page: Page, config: CacheConfigHarnessConfig) {
  await page.setContent('<!doctype html><html><body><main id="root"></main></body></html>');

  await page.evaluate((harness) => {
    type BrowserConfig = CacheConfigHarnessConfig;

    const config = harness as BrowserConfig;
    const root = document.getElementById('root');
    if (!root) {
      throw new Error('Harness root is missing');
    }

    const state = new Map<string, { enabled: boolean; ttl: string }>();
    for (const field of config.fields) {
      state.set(field.key, {
        enabled: field.initialEnabled,
        ttl: String(field.initialTtl),
      });
    }

    root.innerHTML = '';
    root.className = 'min-h-screen bg-background p-6 text-foreground';

    const container = document.createElement('section');
    container.setAttribute('aria-label', config.title);
    container.className =
      'mx-auto max-w-4xl space-y-6 rounded-2xl border border-border bg-card p-6 shadow-sm';

    const heading = document.createElement('div');
    heading.className = 'space-y-2';

    const title = document.createElement('h1');
    title.textContent = config.title;
    title.className = 'text-2xl font-semibold';
    heading.appendChild(title);

    if (config.description) {
      const description = document.createElement('p');
      description.textContent = config.description;
      description.className = 'text-muted-foreground text-sm';
      heading.appendChild(description);
    }

    container.appendChild(heading);

    const form = document.createElement('form');
    form.className = 'space-y-5';

    const fields = new Map<string, { checkbox: HTMLInputElement; ttlInput: HTMLInputElement }>();

    for (const field of config.fields) {
      const fieldset = document.createElement('fieldset');
      fieldset.className = 'space-y-3 rounded-xl border border-border/70 p-4';

      const legend = document.createElement('legend');
      legend.textContent = field.label;
      legend.className = 'px-1 text-sm font-medium';
      fieldset.appendChild(legend);

      const toggleRow = document.createElement('div');
      toggleRow.className = 'flex items-center gap-3';

      const checkbox = document.createElement('input');
      checkbox.type = 'checkbox';
      checkbox.id = `${field.key}-enabled`;
      checkbox.checked = state.get(field.key)?.enabled ?? false;
      checkbox.setAttribute('aria-label', field.enabledLabel);
      checkbox.className = 'h-4 w-4';
      toggleRow.appendChild(checkbox);

      const toggleLabel = document.createElement('label');
      toggleLabel.htmlFor = checkbox.id;
      toggleLabel.textContent = field.enabledLabel;
      toggleLabel.className = 'text-sm';
      toggleRow.appendChild(toggleLabel);

      fieldset.appendChild(toggleRow);

      const ttlRow = document.createElement('div');
      ttlRow.className = 'grid gap-2 md:grid-cols-[minmax(0,1fr)_140px] md:items-center';

      const ttlLabel = document.createElement('label');
      ttlLabel.htmlFor = `${field.key}-ttl`;
      ttlLabel.textContent = field.ttlLabel;
      ttlLabel.className = 'text-sm';
      ttlRow.appendChild(ttlLabel);

      const ttlInput = document.createElement('input');
      ttlInput.type = 'number';
      ttlInput.min = '0';
      ttlInput.step = '1';
      ttlInput.id = `${field.key}-ttl`;
      ttlInput.value = state.get(field.key)?.ttl ?? String(field.initialTtl);
      ttlInput.disabled = !checkbox.checked;
      ttlInput.setAttribute('aria-label', field.ttlLabel);
      ttlInput.className =
        'h-10 rounded-md border border-input bg-background px-3 text-sm disabled:opacity-50';
      ttlRow.appendChild(ttlInput);

      fieldset.appendChild(ttlRow);

      const status = document.createElement('p');
      status.dataset.stateFor = field.key;
      status.className = 'text-muted-foreground text-xs';
      fieldset.appendChild(status);

      checkbox.addEventListener('change', () => {
        const current = state.get(field.key);
        if (!current) return;
        current.enabled = checkbox.checked;
        ttlInput.disabled = !checkbox.checked;
        if (!checkbox.checked) {
          current.ttl = '0';
          ttlInput.value = '0';
        }
        renderPreview();
      });

      ttlInput.addEventListener('input', () => {
        const current = state.get(field.key);
        if (!current) return;
        current.ttl = ttlInput.value;
        renderPreview();
      });

      fields.set(field.key, { checkbox, ttlInput });
      form.appendChild(fieldset);
    }

    const preview = document.createElement('pre');
    preview.dataset.testid = 'payload-preview';
    preview.className = 'overflow-auto rounded-xl border border-border/70 bg-muted/20 p-4 text-xs';

    const status = document.createElement('div');
    status.dataset.testid = 'save-status';
    status.setAttribute('role', 'status');
    status.className = 'min-h-6 text-sm';

    const actions = document.createElement('div');
    actions.className = 'flex items-center gap-3';

    const saveButton = document.createElement('button');
    saveButton.type = 'submit';
    saveButton.textContent = config.submitLabel ?? 'Save cache config';
    saveButton.className =
      'inline-flex h-10 items-center justify-center rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground';
    actions.appendChild(saveButton);

    form.appendChild(actions);
    form.appendChild(preview);
    form.appendChild(status);

    form.addEventListener('submit', async (event) => {
      event.preventDefault();
      const payload = buildPayload(config, state);

      const response = await fetch(config.saveUrl, {
        method: config.saveMethod ?? 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });

      status.textContent = response.ok ? 'Saved' : 'Save failed';
    });

    function renderPreview() {
      const payload = buildPayload(config, state);
      preview.textContent = JSON.stringify(payload, null, 2);
      for (const field of config.fields) {
        const current = state.get(field.key);
        const label = form.querySelector<HTMLElement>(`[data-state-for="${field.key}"]`);
        if (!current || !label) continue;
        label.textContent = current.enabled
          ? `${field.label} is enabled (${current.ttl || '0'}s)`
          : `${field.label} is disabled`;
      }
    }

    renderPreview();

    container.appendChild(form);
    root.appendChild(container);

    function buildPayload(
      harnessConfig: BrowserConfig,
      values: Map<string, { enabled: boolean; ttl: string }>,
    ) {
      const payload: Record<string, unknown> = {};

      if (harnessConfig.wrapper?.kind === 'rule') {
        const field = harnessConfig.fields[0];
        if (!field) {
          throw new Error('Rule harness requires one cache field');
        }

        const current = values.get(field.key);
        payload.externalApiBindings = [
          {
            externalApiKey: harnessConfig.wrapper.externalApiKey,
            ...(harnessConfig.wrapper.baseBinding ?? {}),
            [field.payloadEnabledKey]: current?.enabled ?? false,
            [field.payloadTtlKey]: current?.enabled ? normalizeTtl(current?.ttl) : 0,
          },
        ];
        return payload;
      }

      for (const field of harnessConfig.fields) {
        const current = values.get(field.key);
        payload[field.payloadEnabledKey] = current?.enabled ?? false;
        payload[field.payloadTtlKey] = current?.enabled ? normalizeTtl(current?.ttl) : 0;
      }

      return payload;
    }

    function normalizeTtl(value: string | undefined) {
      const parsed = Number.parseInt(value ?? '', 10);
      return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
    }
  }, config);
}

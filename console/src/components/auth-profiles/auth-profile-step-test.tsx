import { FlaskConical } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { DraftState } from './auth-profile-builder-utils';
import { applyTestPreset } from './auth-profile-builder-utils';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

interface TestResult {
  ok: boolean;
  attempted: boolean;
  httpStatus?: number;
  cached?: boolean;
  details?: Record<string, unknown>;
}

interface StepTestProps {
  draft: DraftState;
  onChange: React.Dispatch<React.SetStateAction<DraftState>>;
  testResult: TestResult | null;
  onRunTest: () => void;
  isTesting: boolean;
}

export function StepTest({ draft, onChange, testResult, onRunTest, isTesting }: StepTestProps) {
  const { t } = useTranslation('settings');

  return (
    <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-1">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-sm font-semibold">{t('authProfiles.testSectionTitle')}</p>
          <p className="text-muted-foreground text-sm">
            {t('authProfiles.testSectionDescription')}
          </p>
        </div>
        <Button
          type="button"
          variant="outline"
          className="min-w-28 shrink-0"
          onClick={onRunTest}
          disabled={isTesting}
        >
          <FlaskConical className="mr-2 h-4 w-4" />
          {isTesting ? t('authProfiles.testing') : t('authProfiles.test')}
        </Button>
      </div>

      {draft.type === 'oidc_standard' ? (
        <div className="space-y-2">
          <Label>{t('authProfiles.oidcForm.testToken')}</Label>
          <Input
            value={draft.test.bearerToken}
            onChange={(event) =>
              onChange((current) => ({
                ...current,
                test: { ...current.test, bearerToken: event.target.value },
              }))
            }
            placeholder={t('authProfiles.oidcForm.testTokenPlaceholder')}
          />
          <p className="text-muted-foreground text-xs">
            {t('authProfiles.oidcForm.testTokenHelp')}
          </p>
        </div>
      ) : (
        <>
          <div className="grid gap-4 lg:grid-cols-3">
            <JsonEditor
              label={t('authProfiles.testHeadersLabel')}
              value={draft.test.headers}
              onChange={(value) =>
                onChange((current) => ({ ...current, test: { ...current.test, headers: value } }))
              }
              helpText={t('authProfiles.testHeadersHelp')}
            />
            <JsonEditor
              label={t('authProfiles.testQueryLabel')}
              value={draft.test.query}
              onChange={(value) =>
                onChange((current) => ({ ...current, test: { ...current.test, query: value } }))
              }
              helpText={t('authProfiles.testQueryHelp')}
            />
            <JsonEditor
              label={t('authProfiles.testBodyLabel')}
              value={draft.test.body}
              onChange={(value) =>
                onChange((current) => ({ ...current, test: { ...current.test, body: value } }))
              }
              helpText={t('authProfiles.testBodyHelp')}
            />
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => onChange((current) => applyTestPreset(current, 'bearer'))}
            >
              {t('authProfiles.testPresets.bearer')}
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => onChange((current) => applyTestPreset(current, 'api_key'))}
            >
              {t('authProfiles.testPresets.apiKey')}
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => onChange((current) => applyTestPreset(current, 'custom_header'))}
            >
              {t('authProfiles.testPresets.customHeader')}
            </Button>
          </div>
        </>
      )}

      {testResult ? (
        <div className="bg-muted/25 space-y-2 rounded-xl border px-4 py-3 text-sm">
          <div className="flex flex-wrap items-center gap-3">
            <span
              className={
                !testResult.attempted
                  ? 'text-content-subtle'
                  : testResult.ok
                    ? 'text-success'
                    : 'text-warning'
              }
            >
              {!testResult.attempted
                ? t('authProfiles.testNotAttemptedBadge')
                : testResult.ok
                  ? t('authProfiles.testOkBadge')
                  : t('authProfiles.testRejectedBadge')}
            </span>
            {typeof testResult.httpStatus === 'number' ? (
              <span>HTTP {testResult.httpStatus}</span>
            ) : null}
            <span>
              {testResult.cached
                ? t('authProfiles.testCached')
                : t('authProfiles.testNotCached')}
            </span>
          </div>
          {typeof testResult.details?.reason === 'string' ? (
            <p className="text-muted-foreground text-sm">{testResult.details.reason}</p>
          ) : null}
          {testResult.details ? (
            <pre className="bg-background max-w-full min-w-0 overflow-auto whitespace-pre-wrap break-all rounded-lg border p-3 text-xs">
              {JSON.stringify(testResult.details, null, 2)}
            </pre>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function JsonEditor({
  label,
  value,
  onChange,
  helpText,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  helpText: string;
}) {
  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      <textarea
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring min-h-[160px] w-full rounded-md border px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none"
      />
      <p className="text-muted-foreground text-xs">{helpText}</p>
    </div>
  );
}

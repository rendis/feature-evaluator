import { useTranslation } from 'react-i18next';

import type { DraftState, SuccessRuleDraft, SuccessRuleType } from './auth-profile-builder-utils';

import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';

interface StepValidationProps {
  draft: DraftState;
  onChange: React.Dispatch<React.SetStateAction<DraftState>>;
}

export function StepValidation({ draft, onChange }: StepValidationProps) {
  const { t } = useTranslation('settings');

  return (
    <div className="min-h-0 flex-1 space-y-6 overflow-y-auto px-1">
      <SuccessRuleEditor
        value={draft.custom.successRule}
        onChange={(successRule) =>
          onChange((current) => ({
            ...current,
            custom: { ...current.custom, successRule },
          }))
        }
      />

      <div className="flex flex-col gap-4 md:flex-row md:items-center">
        <div className="flex items-center gap-3">
          <Switch
            checked={draft.cacheEnabled}
            onCheckedChange={(checked) =>
              onChange((current) => ({ ...current, cacheEnabled: checked }))
            }
          />
          <div>
            <p className="text-sm font-medium">{t('authProfiles.cacheCustom')}</p>
            <p className="text-muted-foreground text-xs">{t('authProfiles.cacheHelp')}</p>
          </div>
        </div>

        <div className="flex w-full items-center gap-2 md:ml-auto md:max-w-64">
          <Label htmlFor="auth-profile-cache-ttl" className="shrink-0 text-sm">
            {t('authProfiles.cacheTTL')}
          </Label>
          <Input
            id="auth-profile-cache-ttl"
            value={draft.cacheTTLSeconds}
            disabled={!draft.cacheEnabled}
            onChange={(event) =>
              onChange((current) => ({ ...current, cacheTTLSeconds: event.target.value }))
            }
            placeholder={t('authProfiles.cacheTTLPlaceholder')}
          />
        </div>
      </div>
    </div>
  );
}

function SuccessRuleEditor({
  value,
  onChange,
}: {
  value: SuccessRuleDraft;
  onChange: (value: SuccessRuleDraft) => void;
}) {
  const { t } = useTranslation('settings');

  return (
    <div className="space-y-3">
      <div>
        <p className="text-sm font-semibold">{t('authProfiles.successRule.title')}</p>
        <p className="text-muted-foreground text-sm">{t('authProfiles.successRule.description')}</p>
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label>{t('authProfiles.successRule.label')}</Label>
          <Select
            value={value.type}
            onValueChange={(next) => onChange({ ...value, type: next as SuccessRuleType })}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="any_2xx">{t('authProfiles.successRule.types.any_2xx')}</SelectItem>
              <SelectItem value="status">{t('authProfiles.successRule.types.status')}</SelectItem>
              <SelectItem value="json_field">
                {t('authProfiles.successRule.types.json_field')}
              </SelectItem>
              <SelectItem value="response_header">
                {t('authProfiles.successRule.types.response_header')}
              </SelectItem>
              <SelectItem value="text_contains">
                {t('authProfiles.successRule.types.text_contains')}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        {value.type === 'status' ? (
          <div className="space-y-2">
            <Label>{t('authProfiles.successRule.status')}</Label>
            <Input
              value={value.status}
              onChange={(event) => onChange({ ...value, status: event.target.value })}
              placeholder="200"
            />
          </div>
        ) : null}

        {value.type === 'json_field' ? (
          <>
            <div className="space-y-2">
              <Label>{t('authProfiles.successRule.jsonPath')}</Label>
              <Input
                value={value.path}
                onChange={(event) => onChange({ ...value, path: event.target.value })}
                placeholder="valid"
              />
            </div>
            <div className="space-y-2">
              <Label>{t('authProfiles.successRule.expectedValue')}</Label>
              <Input
                value={value.value}
                onChange={(event) => onChange({ ...value, value: event.target.value })}
                placeholder="true"
              />
            </div>
          </>
        ) : null}

        {value.type === 'response_header' ? (
          <>
            <div className="space-y-2">
              <Label>{t('authProfiles.successRule.responseHeader')}</Label>
              <Input
                value={value.header}
                onChange={(event) => onChange({ ...value, header: event.target.value })}
                placeholder="x-decision"
              />
            </div>
            <div className="space-y-2">
              <Label>{t('authProfiles.successRule.expectedValue')}</Label>
              <Input
                value={value.value}
                onChange={(event) => onChange({ ...value, value: event.target.value })}
                placeholder="allow"
              />
            </div>
          </>
        ) : null}

        {value.type === 'text_contains' ? (
          <div className="space-y-2">
            <Label>{t('authProfiles.successRule.expectedText')}</Label>
            <Input
              value={value.value}
              onChange={(event) => onChange({ ...value, value: event.target.value })}
              placeholder="ok"
            />
          </div>
        ) : null}
      </div>
    </div>
  );
}

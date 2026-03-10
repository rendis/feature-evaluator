import { useTranslation } from 'react-i18next';

import type { DraftState } from './auth-profile-builder-utils';

import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

interface APIKeyEditorProps {
  draft: DraftState;
  onChange: React.Dispatch<React.SetStateAction<DraftState>>;
}

export function APIKeyEditor({ draft, onChange }: APIKeyEditorProps) {
  const { t } = useTranslation('settings');

  return (
    <div className="space-y-4">
      <div>
        <p className="text-sm font-semibold">{t('authProfiles.apiKeyForm.title')}</p>
        <p className="text-muted-foreground text-sm">{t('authProfiles.apiKeyForm.description')}</p>
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label>{t('authProfiles.apiKeyForm.location')}</Label>
          <Select
            value={draft.apiKey.location}
            onValueChange={(next) =>
              onChange((current) => ({
                ...current,
                apiKey: {
                  ...current.apiKey,
                  location: next as DraftState['apiKey']['location'],
                },
              }))
            }
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="header">
                {t('authProfiles.apiKeyForm.locationOptions.header')}
              </SelectItem>
              <SelectItem value="query">
                {t('authProfiles.apiKeyForm.locationOptions.query')}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-2">
          <Label>
            {draft.apiKey.location === 'header'
              ? t('authProfiles.apiKeyForm.headerName')
              : t('authProfiles.apiKeyForm.queryParamName')}
          </Label>
          <Input
            value={draft.apiKey.name}
            onChange={(event) =>
              onChange((current) => ({
                ...current,
                apiKey: { ...current.apiKey, name: event.target.value },
              }))
            }
            placeholder={draft.apiKey.location === 'header' ? 'X-Api-Key' : 'api_key'}
          />
        </div>
        <div className="space-y-2">
          <Label>{t('authProfiles.apiKeyForm.prefix')}</Label>
          <Input
            value={draft.apiKey.prefix}
            onChange={(event) =>
              onChange((current) => ({
                ...current,
                apiKey: { ...current.apiKey, prefix: event.target.value },
              }))
            }
            placeholder="Bearer"
          />
          <p className="text-muted-foreground text-xs">{t('authProfiles.apiKeyForm.prefixHelp')}</p>
        </div>
        <div className="space-y-2">
          <Label>{t('authProfiles.apiKeyForm.secret')}</Label>
          <Input
            type="password"
            value={draft.apiKey.secret}
            onChange={(event) =>
              onChange((current) => ({
                ...current,
                apiKey: { ...current.apiKey, secret: event.target.value },
              }))
            }
            placeholder={t('authProfiles.apiKeyForm.secretPlaceholder')}
          />
        </div>
      </div>
    </div>
  );
}

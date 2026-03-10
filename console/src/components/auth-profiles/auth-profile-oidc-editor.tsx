import { useTranslation } from 'react-i18next';

import type { DraftState } from './auth-profile-builder-utils';

import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

interface OIDCEditorProps {
  draft: DraftState;
  onChange: React.Dispatch<React.SetStateAction<DraftState>>;
}

export function OIDCEditor({ draft, onChange }: OIDCEditorProps) {
  const { t } = useTranslation('settings');

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <p className="text-sm font-semibold">{t('authProfiles.oidcForm.title')}</p>
        <p className="text-muted-foreground text-sm">{t('authProfiles.oidcForm.description')}</p>
        <div className="bg-muted/25 rounded-xl border border-border/70 px-4 py-3 text-sm">
          {t('authProfiles.oidcForm.discoveryNotice')}
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label>{t('authProfiles.oidcForm.issuer')}</Label>
          <Input
            value={draft.oidc.issuer}
            onChange={(event) =>
              onChange((current) => ({
                ...current,
                oidc: { ...current.oidc, issuer: event.target.value },
              }))
            }
            placeholder="https://idp.example.com/realms/main"
          />
        </div>
        <div className="space-y-2">
          <Label>{t('authProfiles.oidcForm.audience')}</Label>
          <Input
            value={draft.oidc.audience}
            onChange={(event) =>
              onChange((current) => ({
                ...current,
                oidc: { ...current.oidc, audience: event.target.value },
              }))
            }
            placeholder="feature-evaluator"
          />
        </div>
      </div>
    </div>
  );
}

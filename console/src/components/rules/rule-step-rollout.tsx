import { useTranslation } from 'react-i18next';

import { RolloutSection } from './rule-form-rollout';

import type { RuleDraft } from './rule-builder-utils';
import type { Dispatch, SetStateAction } from 'react';

import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';

interface StepRolloutProps {
  draft: RuleDraft;
  onChange: Dispatch<SetStateAction<RuleDraft>>;
}

export function StepRollout({ draft, onChange }: StepRolloutProps) {
  const { t } = useTranslation('rules');

  return (
    <div className="min-h-0 flex-1 space-y-6 overflow-y-auto px-1 py-1">
      <div className="flex items-center gap-3">
        <Label htmlFor="rollout-enabled">{t('form.rolloutToggle')}</Label>
        <Switch
          id="rollout-enabled"
          checked={draft.rolloutEnabled}
          onCheckedChange={(v) => onChange((c) => ({ ...c, rolloutEnabled: v }))}
          aria-label={t('form.rolloutToggle')}
        />
      </div>

      <RolloutSection
        enabled={draft.rolloutEnabled}
        percentage={draft.rolloutLimit}
        onChange={(v) => onChange((c) => ({ ...c, rolloutLimit: v }))}
      />
    </div>
  );
}

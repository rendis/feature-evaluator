import { X } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { FeatureDraft } from './feature-builder-utils';
import type { Dispatch, SetStateAction } from 'react';

import { TagCombobox } from '@/components/shared/tag-combobox';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';


const AVAILABLE_ENVIRONMENTS = ['dev', 'uat', 'production'] as const;

interface StepConfigProps {
  draft: FeatureDraft;
  onChange: Dispatch<SetStateAction<FeatureDraft>>;
}

export function StepConfig({ draft, onChange }: StepConfigProps) {
  const { t } = useTranslation('features');

  const toggleEnvironment = (env: string) => {
    onChange((c) => ({
      ...c,
      environments: c.environments.includes(env)
        ? c.environments.filter((e) => e !== env)
        : [...c.environments, env],
    }));
  };

  return (
    <div className="min-h-0 flex-1 space-y-6 overflow-y-auto px-1 py-1">
      <div className="space-y-2">
        <Label>{t('fields.tags')}</Label>
        <TagCombobox
          value={draft.tags}
          onChange={(tags) => onChange((c) => ({ ...c, tags }))}
        />
      </div>

      <div className="border-t pt-5">
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="activeFrom">{t('fields.activeFrom')}</Label>
            <div className="flex gap-1">
              <Input
                id="activeFrom"
                type="datetime-local"
                value={draft.activeFrom}
                onChange={(e) => onChange((c) => ({ ...c, activeFrom: e.target.value }))}
                className="flex-1"
              />
              {draft.activeFrom ? (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="shrink-0"
                  onClick={() => onChange((c) => ({ ...c, activeFrom: '' }))}
                >
                  <X className="h-4 w-4" />
                </Button>
              ) : null}
            </div>
          </div>
          <div className="space-y-2">
            <Label htmlFor="activeUntil">{t('fields.activeUntil')}</Label>
            <div className="flex gap-1">
              <Input
                id="activeUntil"
                type="datetime-local"
                value={draft.activeUntil}
                onChange={(e) => onChange((c) => ({ ...c, activeUntil: e.target.value }))}
                className="flex-1"
              />
              {draft.activeUntil ? (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="shrink-0"
                  onClick={() => onChange((c) => ({ ...c, activeUntil: '' }))}
                >
                  <X className="h-4 w-4" />
                </Button>
              ) : null}
            </div>
          </div>
          <p className="text-muted-foreground text-xs sm:col-span-2">{t('schedule.helper')}</p>
        </div>
      </div>

      <div className="border-t pt-5">
        <div className="space-y-2">
          <Label>{t('fields.environments')}</Label>
          <div className="flex flex-wrap gap-2">
            {AVAILABLE_ENVIRONMENTS.map((env) => (
              <Badge
                key={env}
                variant={draft.environments.includes(env) ? 'default' : 'outline'}
                className="cursor-pointer select-none"
                onClick={() => toggleEnvironment(env)}
              >
                {t(`environments.${env}`)}
              </Badge>
            ))}
          </div>
          <p className="text-muted-foreground text-xs">{t('environments.helper')}</p>
        </div>
      </div>
    </div>
  );
}

import { Check } from 'lucide-react';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';

import type { DraftState, UseCaseCard } from './auth-profile-builder-utils';
import { getUseCaseCards } from './auth-profile-builder-utils';

import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';

interface StepProfileProps {
  draft: DraftState;
  derivedKey: string;
  isEditing: boolean;
  hasSecret: boolean;
  onChange: React.Dispatch<React.SetStateAction<DraftState>>;
  onTypeChange: (type: DraftState['type']) => void;
}

export function StepProfile({
  draft,
  derivedKey,
  isEditing,
  hasSecret: _hasSecret,
  onChange,
  onTypeChange,
}: StepProfileProps) {
  const { t } = useTranslation('settings');
  const useCaseCards = useMemo(() => getUseCaseCards(t), [t]);
  const currentUseCase =
    useCaseCards.find((option) => option.type === draft.type) ?? useCaseCards[0];

  return (
    <div className="min-h-0 flex-1 space-y-6 overflow-y-auto px-1">
      <div className="flex items-center gap-4">
        <TooltipProvider delayDuration={120}>
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="inline-flex">
                <Switch
                  id="auth-profile-active"
                  aria-label={
                    draft.active ? t('authProfiles.active') : t('authProfiles.inactive')
                  }
                  checked={draft.active}
                  onCheckedChange={(checked) =>
                    onChange((current) => ({ ...current, active: checked }))
                  }
                />
              </span>
            </TooltipTrigger>
            <TooltipContent>
              {draft.active ? t('authProfiles.active') : t('authProfiles.inactive')}
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>

      <div className="grid gap-4 md:grid-cols-[minmax(0,1.4fr)_minmax(220px,0.9fr)] md:items-start">
        <div className="space-y-2">
          <Label htmlFor="auth-profile-name">{t('authProfiles.name')}</Label>
          <Input
            id="auth-profile-name"
            value={draft.name}
            onChange={(event) =>
              onChange((current) => ({ ...current, name: event.target.value }))
            }
            placeholder={t('authProfiles.namePlaceholder')}
          />
          <div className="h-4" />
        </div>

        <div className="space-y-2">
          <Label>{t('authProfiles.key')}</Label>
          <div className="text-muted-foreground border-border/70 bg-muted/35 flex h-10 items-center rounded-md border px-3 font-mono text-sm">
            {derivedKey || t('authProfiles.generatedKey')}
          </div>
          <p className="text-muted-foreground min-h-4 text-xs">
            {t('authProfiles.generatedKeyHelp')}
          </p>
        </div>
      </div>

      <section className="space-y-4">
        {isEditing ? (
          <div className="space-y-3">
            <div>
              <p className="text-sm font-semibold">{t('authProfiles.currentTypeTitle')}</p>
              <p className="text-muted-foreground text-sm">
                {t('authProfiles.currentTypeDescription')}
              </p>
            </div>
            <ReadOnlyUseCaseCard option={currentUseCase} />
          </div>
        ) : (
          <>
            <div>
              <p className="text-sm font-semibold">{t('authProfiles.useCaseTitle')}</p>
              <p className="text-muted-foreground text-sm">
                {t('authProfiles.useCaseDescription')}
              </p>
            </div>
            <div className="grid gap-3 md:grid-cols-3">
              {useCaseCards.map((option) => {
                const Icon = option.icon;
                const selected = draft.type === option.type;
                return (
                  <button
                    key={option.type}
                    type="button"
                    className={`rounded-xl border px-4 py-4 text-left transition ${
                      selected
                        ? 'border-primary bg-primary/5 shadow-sm'
                        : 'border-border hover:border-primary/40'
                    }`}
                    onClick={() => onTypeChange(option.type)}
                  >
                    <div className="mb-3 flex items-center justify-between">
                      <Icon className="text-muted-foreground h-4 w-4" />
                      {selected ? <Check className="text-primary h-4 w-4" /> : null}
                    </div>
                    <p className="text-sm font-medium">{option.title}</p>
                    <p className="text-muted-foreground mt-1 text-sm">{option.description}</p>
                  </button>
                );
              })}
            </div>
          </>
        )}
      </section>
    </div>
  );
}

function ReadOnlyUseCaseCard({ option }: { option: UseCaseCard }) {
  const Icon = option.icon;

  return (
    <div className="rounded-xl border border-border/70 bg-muted/15 px-4 py-4">
      <div className="flex items-start gap-3">
        <div className="bg-background flex h-9 w-9 items-center justify-center rounded-lg border border-border/70">
          <Icon className="text-primary h-4 w-4" />
        </div>
        <div className="space-y-1">
          <p className="text-sm font-medium">{option.title}</p>
          <p className="text-muted-foreground text-sm">{option.description}</p>
        </div>
      </div>
    </div>
  );
}

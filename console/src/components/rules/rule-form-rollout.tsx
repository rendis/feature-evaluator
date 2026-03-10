import { useTranslation } from 'react-i18next';

import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Slider } from '@/components/ui/slider';
import { cn } from '@/lib/utils';

interface RolloutSectionProps {
  enabled: boolean;
  percentage: number;
  onChange: (value: number) => void;
}

export function RolloutSection({ enabled, percentage, onChange }: RolloutSectionProps) {
  const { t } = useTranslation('rules');

  return (
    <div className={cn('space-y-4', !enabled && 'opacity-60')} aria-disabled={!enabled}>
      <div className="space-y-1">
        <Label htmlFor="rollout-limit">{t('rollout.limitLabel')}</Label>
        <p className="text-muted-foreground text-xs">{t('rollout.limitDescription')}</p>
      </div>
      <div className="space-y-3">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
          <Slider
            value={[percentage]}
            onValueChange={([v]) => onChange(v)}
            min={0}
            max={100}
            step={1}
            disabled={!enabled}
            className="flex-1"
          />
          <div className="flex items-center gap-1">
            <Input
              id="rollout-limit"
              type="number"
              min={0}
              max={100}
              value={percentage}
              disabled={!enabled}
              onChange={(e) => {
                const v = parseInt(e.target.value, 10);
                if (!isNaN(v) && v >= 0 && v <= 100) onChange(v);
              }}
              className="w-20 text-center"
            />
            <span className="text-muted-foreground text-sm">%</span>
          </div>
        </div>
        <p className="text-muted-foreground text-xs">
          {percentage === 100 ? t('rollout.allUsers') : t('rollout.partial', { percentage })}
        </p>
      </div>
    </div>
  );
}

import {
  CheckCircle,
  Crown,
  Diamond,
  Flame,
  Gem,
  Lock,
  Rocket,
  Shield,
  Star,
  Trophy,
  Zap,
  type LucideIcon,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { Label } from '@/components/ui/label';
import { cn } from '@/lib/utils';

const BUILTIN_ICONS: { key: string; icon: LucideIcon }[] = [
  { key: 'crown', icon: Crown },
  { key: 'star', icon: Star },
  { key: 'diamond', icon: Diamond },
  { key: 'shield', icon: Shield },
  { key: 'rocket', icon: Rocket },
  { key: 'lightning', icon: Zap },
  { key: 'gem', icon: Gem },
  { key: 'fire', icon: Flame },
  { key: 'lock', icon: Lock },
  { key: 'check-circle', icon: CheckCircle },
  { key: 'zap', icon: Zap },
  { key: 'trophy', icon: Trophy },
];

interface IconPickerProps {
  value: string;
  onChange: (icon: string) => void;
  color?: string;
}

export function IconPicker({ value, onChange, color }: IconPickerProps) {
  const { t } = useTranslation('tiers');

  const selectedKey = value.startsWith('builtin:') ? value.replace('builtin:', '') : '';

  return (
    <div className="space-y-2">
      <Label>{t('icons.builtin')}</Label>
      <div className="grid grid-cols-6 gap-2">
        {BUILTIN_ICONS.map(({ key, icon: Icon }) => {
          const isSelected = selectedKey === key;
          return (
            <button
              key={key}
              type="button"
              className={cn(
                'flex h-10 w-10 items-center justify-center rounded-lg border-2 transition-all',
                isSelected
                  ? 'border-foreground scale-110'
                  : 'border-border hover:border-foreground/50 hover:scale-105',
              )}
              style={isSelected && color ? { backgroundColor: `${color}26` } : undefined}
              onClick={() => onChange(`builtin:${key}`)}
            >
              <Icon
                className="h-5 w-5"
                style={color ? { color } : undefined}
              />
            </button>
          );
        })}
      </div>
    </div>
  );
}

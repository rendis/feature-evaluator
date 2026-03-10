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
import { createElement } from 'react';

import type { TierRef } from '@/api/types';

import { cn } from '@/lib/utils';

const BUILTIN_ICONS: Record<string, LucideIcon> = {
  crown: Crown,
  star: Star,
  diamond: Diamond,
  shield: Shield,
  rocket: Rocket,
  lightning: Zap,
  gem: Gem,
  fire: Flame,
  lock: Lock,
  'check-circle': CheckCircle,
  zap: Zap,
  trophy: Trophy,
};

export function getBuiltinIcon(icon: string): LucideIcon | null {
  const iconKey = icon.startsWith('builtin:') ? icon.replace('builtin:', '') : null;
  return iconKey ? (BUILTIN_ICONS[iconKey] ?? null) : null;
}

interface TierBadgeProps {
  tier: TierRef;
  size?: 'sm' | 'default';
}

export function TierBadge({ tier, size = 'default' }: TierBadgeProps) {
  const icon = getBuiltinIcon(tier.icon);
  const iconSize = size === 'sm' ? 'h-2.5 w-2.5' : 'h-3 w-3';

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-full font-medium',
        size === 'sm' ? 'px-2 py-0.5 text-[10px]' : 'px-2.5 py-0.5 text-xs',
      )}
      style={{
        backgroundColor: `${tier.color}26`,
        color: tier.color,
        border: `1px solid ${tier.color}4D`,
      }}
    >
      {icon ? createElement(icon, { className: iconSize }) : null}
      {tier.name}
    </span>
  );
}

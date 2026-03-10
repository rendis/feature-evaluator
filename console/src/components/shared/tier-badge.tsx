import type React from 'react';

import { findTier } from '@/lib/tier-icons';
import { cn } from '@/lib/utils';

interface TierBadgeProps {
  tier: { key: string; name: string; color: string };
  size?: 'sm' | 'default';
}

export function TierBadge({ tier, size = 'default' }: TierBadgeProps) {
  const tierDef = findTier(tier.key);
  const iconSize = size === 'sm' ? 'h-2.5 w-2.5' : 'h-3 w-3';
  const color = tierDef?.color ?? tier.color;
  const spriteId = tierDef?.spriteId ?? `tier-${tier.key}`;

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-full font-medium',
        size === 'sm' ? 'px-2 py-0.5 text-[10px]' : 'px-2.5 py-0.5 text-xs',
      )}
      style={{
        backgroundColor: `${color}26`,
        color: color,
        border: `1px solid ${color}4D`,
      }}
    >
      <svg
        className={iconSize}
        style={{
          '--icon-fill': `${color}26`,
          '--icon-stroke': color,
        } as React.CSSProperties}
      >
        <use href={`${import.meta.env.BASE_URL}assets/tiers-sprite.svg#${spriteId}`} />
      </svg>
      {tier.name}
    </span>
  );
}

import { X } from 'lucide-react';

import type { Tag } from '@/api/types';

import { cn } from '@/lib/utils';

interface TagBadgeProps {
  tag: Tag;
  onRemove?: () => void;
  size?: 'sm' | 'default';
}

export function TagBadge({ tag, onRemove, size = 'default' }: TagBadgeProps) {
  const color = tag.color;

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
      {tag.name}
      {onRemove ? (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            onRemove();
          }}
          className="ml-0.5 rounded-full p-0.5 hover:bg-foreground/10"
        >
          <X className={cn(size === 'sm' ? 'h-2.5 w-2.5' : 'h-3 w-3')} />
        </button>
      ) : null}
    </span>
  );
}

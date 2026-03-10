import { Check } from 'lucide-react';

import { cn } from '@/lib/utils';

const PRESET_COLORS = [
  '#EF4444',
  '#F97316',
  '#F59E0B',
  '#22C55E',
  '#14B8A6',
  '#06B6D4',
  '#3B82F6',
  '#6366F1',
  '#8B5CF6',
  '#EC4899',
  '#F43F5E',
  '#78716C',
] as const;

interface ColorPickerProps {
  value: string;
  onChange: (color: string) => void;
}

export function ColorPicker({ value, onChange }: ColorPickerProps) {
  return (
    <div className="grid grid-cols-4 gap-2">
      {PRESET_COLORS.map((color) => (
        <button
          key={color}
          type="button"
          className={cn(
            'flex h-7 w-7 items-center justify-center rounded-md border-2 transition-all',
            value === color ? 'border-foreground scale-110' : 'border-transparent hover:scale-105',
          )}
          style={{ backgroundColor: color }}
          onClick={() => onChange(color)}
        >
          {value === color ? <Check className="text-content-inverse h-3.5 w-3.5" /> : null}
        </button>
      ))}
    </div>
  );
}

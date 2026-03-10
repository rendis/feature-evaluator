import { Check, ChevronsUpDown } from 'lucide-react';
import { useState } from 'react';

import { Button } from '@/components/ui/button';
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from '@/components/ui/command';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { TIER_CATEGORIES, findTier, getTiersByCategory } from '@/lib/tier-icons';
import { cn } from '@/lib/utils';

function TierIcon({ spriteId, color, className }: { spriteId: string; color: string; className?: string }) {
  return (
    <svg
      className={cn('h-4 w-4 shrink-0', className)}
      style={{
        '--icon-fill': `${color}26`,
        '--icon-stroke': color,
      } as React.CSSProperties}
    >
      <use href={`${import.meta.env.BASE_URL}assets/tiers-sprite.svg#${spriteId}`} />
    </svg>
  );
}

interface TierPickerProps {
  value: string;
  onChange: (key: string) => void;
}

export function TierPicker({ value, onChange }: TierPickerProps) {
  const [open, setOpen] = useState(false);
  const selected = value ? findTier(value) : undefined;

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full justify-between"
        >
          {selected ? (
            <span className="flex items-center gap-2">
              <TierIcon spriteId={selected.spriteId} color={selected.color} />
              <span>{selected.name}</span>
            </span>
          ) : (
            <span className="text-muted-foreground">Select tier...</span>
          )}
          <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent
        className="w-[--radix-popover-trigger-width] p-0"
        align="start"
        onWheel={(e) => e.stopPropagation()}
      >
        <Command>
          <CommandInput placeholder="Search tier..." />
          <CommandList className="overscroll-contain">
            <CommandEmpty>No tier found.</CommandEmpty>
            {TIER_CATEGORIES.map((category) => {
              const tiers = getTiersByCategory(category);
              return (
                <CommandGroup key={category} heading={category}>
                  {tiers.map((tier) => (
                    <CommandItem
                      key={tier.key}
                      value={`${tier.name} ${tier.key}`}
                      onSelect={() => {
                        onChange(tier.key === value ? '' : tier.key);
                        setOpen(false);
                      }}
                    >
                      <TierIcon spriteId={tier.spriteId} color={tier.color} />
                      <span>{tier.name}</span>
                      <Check
                        className={cn(
                          'ml-auto h-4 w-4',
                          value === tier.key ? 'opacity-100' : 'opacity-0',
                        )}
                      />
                    </CommandItem>
                  ))}
                </CommandGroup>
              );
            })}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

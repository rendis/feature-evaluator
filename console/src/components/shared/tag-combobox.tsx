import { useQuery } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { ColorPicker } from './color-picker';
import { TagBadge } from './tag-badge';

import type { Tag } from '@/api/types';

import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { useCreateTag } from '@/mutations/tag-mutations';
import { tagQueries } from '@/queries/tag-queries';

interface TagComboboxProps {
  value: Tag[];
  onChange: (tags: Tag[]) => void;
}

function CreateTagInline({ search, onCreated }: { search: string; onCreated: (tag: Tag) => void }) {
  const { t } = useTranslation('features');
  const [creating, setCreating] = useState(false);
  const [newColor, setNewColor] = useState('#3B82F6');
  const createTag = useCreateTag();

  const handleCreate = () => {
    if (!search.trim()) return;
    createTag.mutate({ name: search.trim(), color: newColor }, {
      onSuccess: (tag) => { onCreated(tag); setCreating(false); setNewColor('#3B82F6'); },
    });
  };

  if (!creating) {
    return (
      <CommandItem onSelect={() => setCreating(true)}>
        <Plus className="h-4 w-4" />
        <span>{t('tags.createNew')} &ldquo;{search.trim()}&rdquo;</span>
      </CommandItem>
    );
  }

  return (
    <div className="space-y-3 p-2">
      <p className="text-xs font-medium text-muted-foreground">{t('tags.selectColor')}</p>
      <ColorPicker value={newColor} onChange={setNewColor} />
      <button
        type="button"
        onClick={handleCreate}
        disabled={createTag.isPending}
        className="flex w-full items-center justify-center gap-1 rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
      >
        <Plus className="h-3 w-3" />
        {t('tags.createNew')} &ldquo;{search.trim()}&rdquo;
      </button>
    </div>
  );
}

export function TagCombobox({ value, onChange }: TagComboboxProps) {
  const { t } = useTranslation('features');
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => setDebouncedSearch(search), 300);
    return () => clearTimeout(debounceRef.current);
  }, [search]);

  const { data: tags = [] } = useQuery(tagQueries.list(debouncedSearch));
  const selectedKeys = new Set(value.map((tg) => tg.key));
  const available = tags.filter((tg) => !selectedKeys.has(tg.key));

  const handleSelect = (tag: Tag) => { onChange([...value, tag]); setSearch(''); };
  const handleRemove = (key: string) => { onChange(value.filter((tg) => tg.key !== key)); };
  const handleCreated = (tag: Tag) => { onChange([...value, tag]); setSearch(''); };

  return (
    <div className="space-y-2">
      {value.length > 0 ? (
        <div className="flex flex-wrap gap-1.5">
          {value.map((tag) => (
            <TagBadge key={tag.key} tag={tag} onRemove={() => handleRemove(tag.key)} />
          ))}
        </div>
      ) : null}
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <button type="button" className="flex h-9 w-full items-center rounded-md border border-input bg-background px-3 text-sm text-muted-foreground ring-offset-background hover:bg-accent hover:text-accent-foreground">
            <Plus className="mr-2 h-4 w-4" />
            {t('fields.tags')}
          </button>
        </PopoverTrigger>
        <PopoverContent className="w-[280px] p-0" align="start">
          <Command shouldFilter={false}>
            <CommandInput placeholder={t('tags.searchPlaceholder')} value={search} onValueChange={setSearch} />
            <CommandList>
              <CommandEmpty>{t('tags.noResults')}</CommandEmpty>
              {available.length > 0 ? (
                <CommandGroup>
                  {available.map((tag) => (
                    <CommandItem key={tag.key} onSelect={() => handleSelect(tag)}><TagBadge tag={tag} size="sm" /></CommandItem>
                  ))}
                </CommandGroup>
              ) : null}
              {search.trim() ? (
                <>
                  <CommandSeparator />
                  <CommandGroup><CreateTagInline search={search} onCreated={handleCreated} /></CommandGroup>
                </>
              ) : null}
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    </div>
  );
}

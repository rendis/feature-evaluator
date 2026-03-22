import { ArrowRight, ChevronDown, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { DraftState, MappingRow, MappingSourceType, MappingTargetType, StaticHeader } from './auth-profile-builder-utils';
import { combineMappingRows, createMappingRow, createStaticHeader, splitMappingRows } from './auth-profile-builder-utils';

import { Button } from '@/components/ui/button';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

interface CustomEditorProps {
  draft: DraftState;
  onChange: React.Dispatch<React.SetStateAction<DraftState>>;
}

export function CustomEditor({ draft, onChange }: CustomEditorProps) {
  const { t } = useTranslation('settings');
  const custom = draft.custom;
  const mappingRows = combineMappingRows(custom.headerMappings, custom.bodyMappings);

  return (
    <div className="space-y-5">
      <div>
        <p className="text-sm font-semibold">{t('authProfiles.customForm.title')}</p>
        <p className="text-muted-foreground text-sm">{t('authProfiles.customForm.description')}</p>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <div className="space-y-2 md:col-span-2">
          <Label>{t('authProfiles.customForm.url')}</Label>
          <Input
            value={custom.url}
            onChange={(event) =>
              onChange((current) => ({
                ...current,
                custom: { ...current.custom, url: event.target.value },
              }))
            }
            placeholder="https://validator.example.com/check"
          />
        </div>
        <div className="space-y-2">
          <Label>{t('authProfiles.customForm.method')}</Label>
          <Select
            value={custom.method}
            onValueChange={(next) =>
              onChange((current) => ({
                ...current,
                custom: { ...current.custom, method: next as 'GET' | 'POST' },
              }))
            }
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="GET">GET</SelectItem>
              <SelectItem value="POST">POST</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-2">
          <Label>{t('authProfiles.customForm.timeout')}</Label>
          <Input
            value={custom.timeout}
            onChange={(event) =>
              onChange((current) => ({
                ...current,
                custom: { ...current.custom, timeout: event.target.value },
              }))
            }
            placeholder="5000"
          />
        </div>
        <div className="space-y-2">
          <Label>{t('authProfiles.customForm.outboundHeader')}</Label>
          <Input
            value={custom.outboundAuthHeaderName}
            onChange={(event) =>
              onChange((current) => ({
                ...current,
                custom: { ...current.custom, outboundAuthHeaderName: event.target.value },
              }))
            }
            placeholder={t('authProfiles.customForm.outboundHeaderPlaceholder')}
          />
        </div>
        <div className="space-y-2">
          <Label>{t('authProfiles.customForm.outboundApiKey')}</Label>
          <Input
            type="password"
            value={custom.outboundApiKey}
            onChange={(event) =>
              onChange((current) => ({
                ...current,
                custom: { ...current.custom, outboundApiKey: event.target.value },
              }))
            }
          />
        </div>
      </div>

      <StaticHeadersSection
        headers={custom.requestHeaders}
        onAdd={() =>
          onChange((current) => ({
            ...current,
            custom: {
              ...current.custom,
              requestHeaders: [...current.custom.requestHeaders, createStaticHeader()],
            },
          }))
        }
        onChange={(rows) =>
          onChange((current) => ({
            ...current,
            custom: { ...current.custom, requestHeaders: rows },
          }))
        }
      />

      <MappingTable
        label={t('authProfiles.mapping.title')}
        rows={mappingRows}
        addLabel={t('authProfiles.mapping.add')}
        onAdd={() =>
          onChange((current) => ({
            ...current,
            custom: {
              ...current.custom,
              headerMappings: [...current.custom.headerMappings, createMappingRow('header')],
            },
          }))
        }
        onChange={(rows) =>
          onChange((current) => ({
            ...current,
            custom: {
              ...current.custom,
              ...splitMappingRows(rows),
            },
          }))
        }
      />
    </div>
  );
}

function StaticHeadersSection({
  headers,
  onAdd,
  onChange,
}: {
  headers: StaticHeader[];
  onAdd: () => void;
  onChange: (rows: StaticHeader[]) => void;
}) {
  const { t } = useTranslation('settings');

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-semibold">{t('authProfiles.requestHeaders.title')}</p>
          <p className="text-muted-foreground text-xs">
            {t('authProfiles.requestHeaders.description')}
          </p>
        </div>
        <Button type="button" variant="outline" size="sm" className="w-44 justify-center" onClick={onAdd}>
          {t('authProfiles.requestHeaders.add')}
        </Button>
      </div>
      {headers.map((header) => (
        <div
          key={header.id}
          className="grid gap-3 rounded-2xl border border-border/70 bg-muted/15 p-3 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_40px] md:items-center"
        >
          <Input
            value={header.key}
            onChange={(event) =>
              onChange(headers.map((h) => (h.id === header.id ? { ...h, key: event.target.value } : h)))
            }
            placeholder={t('authProfiles.requestHeaders.keyPlaceholder')}
            className="font-mono text-sm"
          />
          <Input
            value={header.value}
            onChange={(event) =>
              onChange(headers.map((h) => (h.id === header.id ? { ...h, value: event.target.value } : h)))
            }
            placeholder={t('authProfiles.requestHeaders.valuePlaceholder')}
            className="text-sm"
          />
          <div className="flex items-center justify-end md:justify-center">
            <Button
              type="button"
              variant="ghost"
              size="icon"
              aria-label={t('authProfiles.requestHeaders.removeAria')}
              onClick={() => onChange(headers.filter((h) => h.id !== header.id))}
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          </div>
        </div>
      ))}
    </div>
  );
}

function MappingTable({
  label,
  rows,
  addLabel,
  onAdd,
  onChange,
}: {
  label: string;
  rows: MappingRow[];
  addLabel: string;
  onAdd: () => void;
  onChange: (rows: MappingRow[]) => void;
}) {
  const { t } = useTranslation('settings');

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-semibold">{label}</p>
          <p className="text-muted-foreground text-xs">{t('authProfiles.mapping.description')}</p>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="w-44 justify-center"
          onClick={onAdd}
        >
          {addLabel}
        </Button>
      </div>
      <div className="space-y-3">
        {rows.map((row) => (
          <div
            key={row.id}
            className="grid gap-3 rounded-2xl border border-border/70 bg-muted/15 p-3 md:grid-cols-[minmax(0,1fr)_40px_minmax(0,1fr)_40px] md:items-start"
          >
            <MappingEndpoint
              type={row.sourceType}
              value={row.sourceValue}
              stripPrefix={row.sourceStripPrefix}
              headerLabel={t('authProfiles.mapping.header')}
              bodyLabel={t('authProfiles.mapping.body')}
              onTypeChange={(next) =>
                onChange(
                  rows.map((item) =>
                    item.id === row.id
                      ? {
                          ...item,
                          sourceType: next as MappingSourceType,
                          sourceStripPrefix:
                            next === 'request_header' ? item.sourceStripPrefix : '',
                        }
                      : item,
                  ),
                )
              }
              onValueChange={(next) =>
                onChange(
                  rows.map((item) => (item.id === row.id ? { ...item, sourceValue: next } : item)),
                )
              }
              onStripPrefixChange={(next) =>
                onChange(
                  rows.map((item) =>
                    item.id === row.id ? { ...item, sourceStripPrefix: next } : item,
                  ),
                )
              }
              placeholder={
                row.sourceType === 'request_header'
                  ? t('authProfiles.mapping.sourceHeaderPlaceholder')
                  : t('authProfiles.mapping.sourceBodyPlaceholder')
              }
              caption={t('authProfiles.mapping.origin')}
            />

            <div className="text-muted-foreground hidden justify-center pt-[52px] md:flex">
              <ArrowRight className="h-5 w-5" />
            </div>

            <MappingEndpoint
              type={row.targetType}
              value={row.targetValue}
              headerLabel={t('authProfiles.mapping.header')}
              bodyLabel={t('authProfiles.mapping.body')}
              onTypeChange={(next) =>
                onChange(
                  rows.map((item) =>
                    item.id === row.id ? { ...item, targetType: next as MappingTargetType } : item,
                  ),
                )
              }
              onValueChange={(next) =>
                onChange(
                  rows.map((item) => (item.id === row.id ? { ...item, targetValue: next } : item)),
                )
              }
              placeholder={
                row.targetType === 'header'
                  ? t('authProfiles.mapping.targetHeaderPlaceholder')
                  : t('authProfiles.mapping.targetBodyPlaceholder')
              }
              caption={t('authProfiles.mapping.destination')}
            />

            <div className="flex items-start justify-end md:justify-center">
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label={t('authProfiles.mapping.removeAria')}
                onClick={() => onChange(rows.filter((item) => item.id !== row.id))}
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function MappingEndpoint({
  type,
  value,
  stripPrefix,
  headerLabel,
  bodyLabel,
  onTypeChange,
  onValueChange,
  onStripPrefixChange,
  placeholder,
  caption,
}: {
  type: MappingSourceType | MappingTargetType;
  value: string;
  stripPrefix?: string;
  headerLabel: string;
  bodyLabel: string;
  onTypeChange: (value: string) => void;
  onValueChange: (value: string) => void;
  onStripPrefixChange?: (value: string) => void;
  placeholder: string;
  caption: string;
}) {
  const { t } = useTranslation('settings');
  const isHeader = type === 'request_header' || type === 'header';
  const supportsStripPrefix = type === 'request_header' && !!onStripPrefixChange;

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <div className="inline-flex rounded-full border border-border/80 bg-background/60 p-1 shadow-[inset_0_1px_2px_rgba(255,255,255,0.04)]">
          <button
            type="button"
            className={`rounded-full px-3 py-1 text-xs font-medium transition ${
              isHeader ? 'bg-muted text-foreground shadow-sm' : 'text-muted-foreground'
            }`}
            onClick={() =>
              onTypeChange(
                type === 'request_body' || type === 'body' ? type.replace('body', 'header') : type,
              )
            }
          >
            {headerLabel}
          </button>
          <button
            type="button"
            className={`rounded-full px-3 py-1 text-xs font-medium transition ${
              !isHeader ? 'bg-muted text-foreground shadow-sm' : 'text-muted-foreground'
            }`}
            onClick={() =>
              onTypeChange(
                type === 'request_header' || type === 'request_body' ? 'request_body' : 'body',
              )
            }
          >
            {bodyLabel}
          </button>
        </div>
      </div>

      {isHeader ? (
        <HeaderCombobox value={value} onChange={onValueChange} placeholder={placeholder} />
      ) : (
        <div className="rounded-2xl border border-dashed border-border/70 bg-background/40 px-3 py-2 transition focus-within:border-primary/60 focus-within:bg-background/70">
          <Input
            value={value}
            onChange={(event) => onValueChange(event.target.value)}
            placeholder={placeholder}
            className="h-auto rounded-none border-0 bg-transparent px-0 py-0 text-sm shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
          />
        </div>
      )}

      <p className="text-muted-foreground text-center text-[11px] uppercase tracking-wide">
        {caption}
      </p>

      {supportsStripPrefix ? (
        <div className="space-y-1">
          <p className="text-muted-foreground text-[11px]">
            {t('authProfiles.mapping.stripPrefix')}
          </p>
          <Input
            value={stripPrefix ?? ''}
            onChange={(event) => onStripPrefixChange?.(event.target.value)}
            placeholder={t('authProfiles.mapping.stripPrefixPlaceholder')}
            className="h-8 text-xs"
          />
        </div>
      ) : (
        <div className="h-11" />
      )}
    </div>
  );
}

const COMMON_HEADERS = [
  'Authorization',
  'X-Api-Key',
  'X-Auth-Token',
  'X-Session-Token',
  'X-CSRF-Token',
  'X-Request-ID',
  'X-Forwarded-For',
  'Cookie',
  'Content-Type',
  'Accept',
] as const;

function HeaderCombobox({
  value,
  onChange,
  placeholder,
}: {
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
}) {
  const { t } = useTranslation('settings');
  const [open, setOpen] = useState(false);

  return (
    <div className="flex rounded-2xl border border-dashed border-border/70 bg-background/40 transition focus-within:border-primary/60 focus-within:bg-background/70">
      <Input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        className="h-auto flex-1 rounded-none border-0 bg-transparent px-3 py-2 text-sm shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
      />
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <button
            type="button"
            className="text-muted-foreground hover:text-foreground flex shrink-0 items-center px-2 transition"
            aria-label={t('authProfiles.mapping.commonHeaders')}
          >
            <ChevronDown className="h-3.5 w-3.5" />
          </button>
        </PopoverTrigger>
        <PopoverContent className="w-[220px] p-0" align="end" sideOffset={6}>
          <Command>
            <CommandInput placeholder={t('authProfiles.mapping.searchHeaders')} />
            <CommandList>
              <CommandEmpty>{t('authProfiles.mapping.noHeaders')}</CommandEmpty>
              <CommandGroup>
                {COMMON_HEADERS.map((header) => (
                  <CommandItem
                    key={header}
                    value={header}
                    onSelect={() => {
                      onChange(header);
                      setOpen(false);
                    }}
                  >
                    <span className="font-mono text-xs">{header}</span>
                  </CommandItem>
                ))}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    </div>
  );
}

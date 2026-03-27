import { useQuery, useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute, Link } from '@tanstack/react-router';
import { Plus, ToggleLeft } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { ListFeaturesParams } from '@/api/features';
import type { Tag } from '@/api/types';

import { FeatureList } from '@/components/features/feature-list';
import { EmptyState } from '@/components/shared/empty-state';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { PageHeader } from '@/components/shared/page-header';
import { PermissionButton } from '@/components/shared/permission-button';
import { TagBadge } from '@/components/shared/tag-badge';
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
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useDebounce } from '@/hooks/use-debounce';
import { featureQueries } from '@/queries/feature-queries';
import { tagQueries } from '@/queries/tag-queries';

export const Route = createFileRoute('/_authenticated/features/')({
  component: FeaturesPage,
  pendingComponent: () => <LoadingSkeleton rows={5} />,
  errorComponent: ({ error }) => <ApiErrorState error={error} />,
});

export function FeaturesPage() {
  const { t } = useTranslation('features');
  const [params, setParams] = useState<ListFeaturesParams>({ page: 1, pageSize: 20 });
  const [search, setSearch] = useState('');
  const debouncedSearch = useDebounce(search, 300);
  const { data } = useSuspenseQuery(featureQueries.summaryList(params));

  /* eslint-disable react-hooks/set-state-in-effect -- debounced search intentionally drives list query state */
  useEffect(() => {
    const normalizedSearch = debouncedSearch.trim() || undefined;
    setParams((previous) =>
      previous.search === normalizedSearch
        ? previous
        : { ...previous, page: 1, search: normalizedSearch },
    );
  }, [debouncedSearch]);
  /* eslint-enable react-hooks/set-state-in-effect */

  const features = data.data;
  const pagination = data.pagination;

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('title')}
        description={t('description')}
        actions={
          <PermissionButton permission="features.write" asChild>
            <Link to="/features/new">
              <Plus className="mr-2 h-4 w-4" />
              {t('createFeature')}
            </Link>
          </PermissionButton>
        }
      />

      <FeaturesFilters
        params={params}
        search={search}
        onSearchChange={setSearch}
        onChange={setParams}
      />

      {features.length === 0 ? (
        <EmptyState
          icon={<ToggleLeft className="h-10 w-10" />}
          title={t('empty.title')}
          description={t('empty.description')}
          action={
            <PermissionButton permission="features.write" asChild>
              <Link to="/features/new">{t('createFeature')}</Link>
            </PermissionButton>
          }
        />
      ) : (
        <>
          <FeatureList features={features} />
          <FeaturesPagination
            page={pagination.page}
            totalPages={pagination.totalPages}
            onPageChange={(page) => setParams((p) => ({ ...p, page }))}
          />
        </>
      )}
    </div>
  );
}

interface FeaturesFiltersProps {
  params: ListFeaturesParams;
  search: string;
  onSearchChange: (value: string) => void;
  onChange: (params: ListFeaturesParams) => void;
}

function FeaturesFilters({ params, search, onSearchChange, onChange }: FeaturesFiltersProps) {
  const { t } = useTranslation('features');

  return (
    <div className="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center">
      <Input
        placeholder={t('search')}
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
        className="sm:max-w-xs"
      />
      <div className="flex gap-2">
        <Button
          variant={params.enabled === undefined ? 'default' : 'outline'}
          size="sm"
          onClick={() => onChange({ ...params, enabled: undefined, page: 1 })}
        >
          {t('filters.all')}
        </Button>
        <Button
          variant={params.enabled === true ? 'default' : 'outline'}
          size="sm"
          onClick={() => onChange({ ...params, enabled: true, page: 1 })}
        >
          {t('filters.enabled')}
        </Button>
        <Button
          variant={params.enabled === false ? 'default' : 'outline'}
          size="sm"
          onClick={() => onChange({ ...params, enabled: false, page: 1 })}
        >
          {t('filters.disabled')}
        </Button>
      </div>
      <Select
        value={params.environment ?? 'all'}
        onValueChange={(v) =>
          onChange({ ...params, environment: v === 'all' ? undefined : v, page: 1 })
        }
      >
        <SelectTrigger className="w-[150px]">
          <SelectValue placeholder={t('filters.environment')} />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">{t('filters.all')}</SelectItem>
          <SelectItem value="dev">{t('environments.dev')}</SelectItem>
          <SelectItem value="uat">{t('environments.uat')}</SelectItem>
          <SelectItem value="production">{t('environments.production')}</SelectItem>
        </SelectContent>
      </Select>
      <TagFilterSelect
        selected={params.tags ?? []}
        onChange={(tags) =>
          onChange({ ...params, tags: tags.length > 0 ? tags : undefined, page: 1 })
        }
      />
    </div>
  );
}

function TagFilterSelect({
  selected,
  onChange,
}: {
  selected: string[];
  onChange: (tags: string[]) => void;
}) {
  const { t } = useTranslation('features');
  const [open, setOpen] = useState(false);
  const { data: tags = [] } = useQuery(tagQueries.list());

  const selectedSet = new Set(selected);
  const selectedTags = tags.filter((tg: Tag) => selectedSet.has(tg.key));

  const toggleTag = (key: string) => {
    if (selectedSet.has(key)) {
      onChange(selected.filter((k) => k !== key));
    } else {
      onChange([...selected, key]);
    }
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="gap-1">
          {t('filters.tags')}
          {selectedTags.length > 0 ? (
            <span className="ml-1 flex gap-1">
              {selectedTags.map((tg: Tag) => (
                <TagBadge key={tg.key} tag={tg} size="sm" />
              ))}
            </span>
          ) : null}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[220px] p-0" align="start">
        <Command>
          <CommandInput placeholder={t('tags.searchPlaceholder')} />
          <CommandList>
            <CommandEmpty>{t('tags.noResults')}</CommandEmpty>
            <CommandGroup>
              {tags.map((tg: Tag) => (
                <CommandItem key={tg.key} onSelect={() => toggleTag(tg.key)}>
                  <div className="flex items-center gap-2">
                    <div
                      className={`flex h-4 w-4 items-center justify-center rounded-sm border ${selectedSet.has(tg.key) ? 'border-primary bg-primary' : 'border-muted-foreground'}`}
                    >
                      {selectedSet.has(tg.key) ? (
                        <svg
                          className="h-3 w-3 text-primary-foreground"
                          viewBox="0 0 24 24"
                          fill="none"
                          stroke="currentColor"
                          strokeWidth="3"
                          strokeLinecap="round"
                          strokeLinejoin="round"
                        >
                          <polyline points="20 6 9 17 4 12" />
                        </svg>
                      ) : null}
                    </div>
                    <TagBadge tag={tg} size="sm" />
                  </div>
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

interface FeaturesPaginationProps {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}

function FeaturesPagination({ page, totalPages, onPageChange }: FeaturesPaginationProps) {
  const { t } = useTranslation();

  if (totalPages <= 1) return null;

  return (
    <div className="flex items-center justify-center gap-4">
      <Button
        variant="outline"
        size="sm"
        disabled={page <= 1}
        onClick={() => onPageChange(page - 1)}
      >
        {t('pagination.previous')}
      </Button>
      <span className="text-muted-foreground text-sm">
        {t('pagination.page')} {page} {t('pagination.of')} {totalPages}
      </span>
      <Button
        variant="outline"
        size="sm"
        disabled={page >= totalPages}
        onClick={() => onPageChange(page + 1)}
      >
        {t('pagination.next')}
      </Button>
    </div>
  );
}

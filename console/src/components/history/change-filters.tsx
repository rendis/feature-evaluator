import { useTranslation } from 'react-i18next';

import type { ListChangelogParams } from '@/api/changelog';

import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';

interface ChangeFiltersProps {
  params: ListChangelogParams;
  onChange: (params: ListChangelogParams) => void;
}

const ENTITY_TYPES = ['feature', 'rule', 'segment', 'pack'] as const;
const ACTIONS = ['create', 'update', 'delete', 'toggle', 'reorder'] as const;

export function ChangeFilters({ params, onChange }: ChangeFiltersProps) {
  const { t } = useTranslation('history');

  const update = (patch: Partial<ListChangelogParams>) => {
    onChange({ ...params, ...patch, page: 1 });
  };

  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-5">
      <Select
        value={params.entityType ?? '_all'}
        onValueChange={(v) => update({ entityType: v === '_all' ? undefined : v })}
      >
        <SelectTrigger>
          <SelectValue placeholder={t('filters.entityType')} />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="_all">{t('filters.allEntities')}</SelectItem>
          {ENTITY_TYPES.map((et) => (
            <SelectItem key={et} value={et}>
              {t(`entityType.${et}`)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Select
        value={params.action ?? '_all'}
        onValueChange={(v) => update({ action: v === '_all' ? undefined : v })}
      >
        <SelectTrigger>
          <SelectValue placeholder={t('filters.action')} />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="_all">{t('filters.allActions')}</SelectItem>
          {ACTIONS.map((a) => (
            <SelectItem key={a} value={a}>
              {t(`action.${a}`)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Input
        placeholder={t('filters.actor')}
        value={params.actor ?? ''}
        onChange={(e) => update({ actor: e.target.value })}
      />
      <Input
        type="date"
        placeholder={t('filters.from')}
        value={params.from ?? ''}
        onChange={(e) => update({ from: e.target.value })}
      />
      <Input
        type="date"
        placeholder={t('filters.to')}
        value={params.to ?? ''}
        onChange={(e) => update({ to: e.target.value })}
      />
    </div>
  );
}

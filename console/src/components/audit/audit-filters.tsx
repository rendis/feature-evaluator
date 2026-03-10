import { useTranslation } from 'react-i18next';

import type { ListAuditErrorsParams } from '@/api/audit';

import { Input } from '@/components/ui/input';

interface AuditFiltersProps {
  params: ListAuditErrorsParams;
  onChange: (params: ListAuditErrorsParams) => void;
}

export function AuditFilters({ params, onChange }: AuditFiltersProps) {
  const { t } = useTranslation('audit');

  const update = (patch: Partial<ListAuditErrorsParams>) => {
    onChange({ ...params, ...patch, page: 1 });
  };

  return (
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-5">
      <Input
        placeholder={t('filters.featureKey')}
        value={params.featureKey ?? ''}
        onChange={(e) => update({ featureKey: e.target.value })}
      />
      <Input
        placeholder={t('filters.tenantId')}
        value={params.tenantId ?? ''}
        onChange={(e) => update({ tenantId: e.target.value })}
      />
      <Input
        placeholder={t('filters.errorType')}
        value={params.errorType ?? ''}
        onChange={(e) => update({ errorType: e.target.value })}
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

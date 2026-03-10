import { useTranslation } from 'react-i18next';

import type { AuditError } from '@/api/types';

import { Badge } from '@/components/ui/badge';
import { useLocaleFormatters } from '@/hooks/use-locale-formatters';

interface AuditTableProps {
  errors: AuditError[];
}

function AuditMobileCard({ error }: { error: AuditError }) {
  const { formatDate } = useLocaleFormatters();

  return (
    <div className="space-y-2 rounded-lg border p-4">
      <div className="flex items-center justify-between">
        <Badge variant="destructive">{error.errorType}</Badge>
        <span className="text-muted-foreground text-xs">{formatDate(error.createdAt)}</span>
      </div>
      <p className="text-sm font-medium">{error.featureKey}</p>
      <p className="text-muted-foreground text-sm">{error.message}</p>
      <div className="text-muted-foreground flex gap-3 text-xs">
        {error.tenantId ? <span>tenant: {error.tenantId}</span> : null}
        {error.ruleId ? <span>rule: {error.ruleId}</span> : null}
      </div>
    </div>
  );
}

function AuditDesktopRow({ error }: { error: AuditError }) {
  const { formatDate } = useLocaleFormatters();

  return (
    <tr className="border-b last:border-0">
      <td className="px-4 py-3 text-sm">{formatDate(error.createdAt)}</td>
      <td className="px-4 py-3 text-sm font-medium">{error.featureKey}</td>
      <td className="px-4 py-3">
        <Badge variant="destructive">{error.errorType}</Badge>
      </td>
      <td className="max-w-xs truncate px-4 py-3 text-sm">{error.message}</td>
      <td className="px-4 py-3 text-sm">{error.tenantId || '—'}</td>
      <td className="text-muted-foreground px-4 py-3 text-xs">{error.requestId}</td>
    </tr>
  );
}

export function AuditTable({ errors }: AuditTableProps) {
  const { t } = useTranslation('audit');

  return (
    <>
      <div className="space-y-3 md:hidden">
        {errors.map((error) => (
          <AuditMobileCard key={error.id} error={error} />
        ))}
      </div>

      <div className="hidden overflow-x-auto rounded-lg border md:block">
        <table className="w-full text-left">
          <thead className="border-b bg-muted/50">
            <tr>
              <th className="px-4 py-3 text-xs font-medium">{t('columns.date')}</th>
              <th className="px-4 py-3 text-xs font-medium">{t('columns.featureKey')}</th>
              <th className="px-4 py-3 text-xs font-medium">{t('columns.errorType')}</th>
              <th className="px-4 py-3 text-xs font-medium">{t('columns.message')}</th>
              <th className="px-4 py-3 text-xs font-medium">{t('columns.tenantId')}</th>
              <th className="px-4 py-3 text-xs font-medium">{t('columns.requestId')}</th>
            </tr>
          </thead>
          <tbody>
            {errors.map((error) => (
              <AuditDesktopRow key={error.id} error={error} />
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}

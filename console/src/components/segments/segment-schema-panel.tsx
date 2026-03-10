import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';

import type { SegmentSchema } from '@/api/types';

import { flattenSchemaFields } from './segment-data-utils';

interface SegmentSchemaPanelProps {
  schemaData: SegmentSchema;
}

export function SegmentSchemaPanel({ schemaData }: SegmentSchemaPanelProps) {
  const { t } = useTranslation('segments');
  const fields = useMemo(() => flattenSchemaFields(schemaData.schema), [schemaData.schema]);

  return (
    <div className="min-w-0 space-y-4">
      <div className="grid gap-3 md:grid-cols-3">
        <SchemaMeta label={t('schema.keyPath')} value={schemaData.recordKeyPath} monospace />
        <SchemaMeta label={t('schema.sourceType')} value={schemaData.sourceType ?? '-'} />
        <SchemaMeta label={t('schema.recordCount')} value={String(schemaData.recordCount)} />
      </div>

      <div className="max-w-full overflow-x-auto rounded-md border">
        <table className="w-max min-w-full text-sm">
          <thead>
            <tr className="border-b bg-muted/50">
              <th className="px-4 py-3 text-left font-medium">{t('schema.path')}</th>
              <th className="px-4 py-3 text-left font-medium">{t('schema.type')}</th>
              <th className="px-4 py-3 text-left font-medium">{t('schema.required')}</th>
              <th className="px-4 py-3 text-left font-medium">{t('schema.nullable')}</th>
            </tr>
          </thead>
          <tbody>
            {fields.map((field) => (
              <tr key={field.path} className="border-b last:border-0">
                <td className="px-4 py-3 font-mono text-xs break-all">{field.path}</td>
                <td className="px-4 py-3">{field.type}</td>
                <td className="px-4 py-3">{field.required ? t('common.yes', { ns: 'common', defaultValue: 'Yes' }) : t('common.no', { ns: 'common', defaultValue: 'No' })}</td>
                <td className="px-4 py-3">{field.nullable ? t('common.yes', { ns: 'common', defaultValue: 'Yes' }) : t('common.no', { ns: 'common', defaultValue: 'No' })}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="space-y-2">
        <p className="text-sm font-medium">{t('schema.raw')}</p>
        <pre className="max-h-96 overflow-auto rounded-md border bg-muted/30 p-4 text-xs">{JSON.stringify(schemaData.schema, null, 2)}</pre>
      </div>
    </div>
  );
}

function SchemaMeta({
  label,
  value,
  monospace = false,
}: {
  label: string;
  value: string;
  monospace?: boolean;
}) {
  return (
    <div className="rounded-md border bg-muted/20 p-3">
      <p className="text-muted-foreground text-xs">{label}</p>
      <p className={monospace ? 'font-mono text-sm' : 'text-sm'}>{value}</p>
    </div>
  );
}

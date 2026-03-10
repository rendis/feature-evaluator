import { Link } from '@tanstack/react-router';
import { useTranslation } from 'react-i18next';

import type { Segment } from '@/api/types';

import { useLocaleFormatters } from '@/hooks/use-locale-formatters';
import { useMobile } from '@/hooks/use-mobile';

interface SegmentListProps {
  segments: Segment[];
}

export function SegmentList({ segments }: SegmentListProps) {
  const isMobile = useMobile();

  if (isMobile) {
    return (
      <div className="space-y-3">
        {segments.map((s) => (
          <SegmentCard key={s.key} segment={s} />
        ))}
      </div>
    );
  }

  return <SegmentDesktopTable segments={segments} />;
}

function SegmentCard({ segment }: { segment: Segment }) {
  const { t } = useTranslation('segments');
  const { formatDate } = useLocaleFormatters();
  return (
    <Link
      to="/segments/$segmentKey"
      params={{ segmentKey: segment.key }}
      className="block rounded-md border p-4 transition-colors hover:bg-muted/50"
    >
      <p className="font-medium">{segment.name}</p>
      <p className="text-muted-foreground font-mono text-xs">{segment.key}</p>
      {segment.description ? (
        <p className="text-muted-foreground mt-1 text-sm">{segment.description}</p>
      ) : null}
      <p className="text-muted-foreground mt-1 text-xs">
        {t('data.recordCountLabel', { count: segment.recordCount })}
      </p>
      {segment.lastImportAt ? (
        <p className="text-muted-foreground mt-1 text-xs">
          {t('fields.lastImport')}: {formatDate(segment.lastImportAt)}
        </p>
      ) : null}
    </Link>
  );
}

function SegmentDesktopTable({ segments }: { segments: Segment[] }) {
  const { t } = useTranslation('segments');
  const { formatDate } = useLocaleFormatters();
  return (
    <div className="rounded-md border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/50">
            <th className="px-4 py-3 text-left font-medium">{t('fields.name')}</th>
            <th className="px-4 py-3 text-left font-medium">{t('fields.key')}</th>
            <th className="px-4 py-3 text-left font-medium">{t('data.title')}</th>
            <th className="px-4 py-3 text-left font-medium">{t('fields.description')}</th>
            <th className="px-4 py-3 text-left font-medium">{t('fields.lastImport')}</th>
          </tr>
        </thead>
        <tbody>
          {segments.map((s) => (
            <tr key={s.key} className="border-b last:border-0">
              <td className="px-4 py-3">
                <Link
                  to="/segments/$segmentKey"
                  params={{ segmentKey: s.key }}
                  className="hover:text-primary font-medium transition-colors"
                >
                  {s.name}
                </Link>
              </td>
              <td className="text-muted-foreground px-4 py-3 font-mono text-xs">{s.key}</td>
              <td className="text-muted-foreground px-4 py-3 text-xs">{s.recordCount}</td>
              <td className="text-muted-foreground max-w-xs truncate px-4 py-3">{s.description}</td>
              <td className="text-muted-foreground px-4 py-3 text-xs">
                {s.lastImportAt ? formatDate(s.lastImportAt) : '-'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

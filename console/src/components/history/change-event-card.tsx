import { useTranslation } from 'react-i18next';

import type { ChangeEntry } from '@/api/changelog';

import { Badge } from '@/components/ui/badge';
import { useLocaleFormatters } from '@/hooks/use-locale-formatters';

interface ChangeEventCardProps {
  entry: ChangeEntry;
  onClick?: () => void;
}

function actionVariant(action: string): 'success' | 'destructive' | 'secondary' | 'outline' {
  switch (action) {
    case 'create':
      return 'success';
    case 'delete':
      return 'destructive';
    case 'update':
    case 'toggle':
      return 'secondary';
    default:
      return 'outline';
  }
}

export function ChangeEventCard({ entry, onClick }: ChangeEventCardProps) {
  const { t } = useTranslation('history');
  const { formatDateTime } = useLocaleFormatters();
  const hasChanges = entry.fieldChanges && entry.fieldChanges.length > 0;

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={!hasChanges}
      className="group flex w-full items-start gap-3 rounded-md border p-3 text-left transition-colors hover:bg-muted/50 disabled:cursor-default disabled:hover:bg-transparent"
    >
      <div className="flex-1 min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={actionVariant(entry.action)}>{t(`action.${entry.action}`)}</Badge>
          <Badge variant="outline">{t(`entityType.${entry.entityType}`)}</Badge>
          <span className="truncate font-medium text-sm">{entry.entityKey}</span>
          {entry.parentKey ? (
            <span className="text-muted-foreground text-xs">({entry.parentKey})</span>
          ) : null}
        </div>
        <div className="text-muted-foreground mt-1 flex items-center gap-3 text-xs">
          <span>{entry.actor}</span>
          <span>{formatDateTime(entry.createdAt)}</span>
          {hasChanges ? (
            <span className="text-primary group-hover:underline">
              {t('changes.viewDiff', { count: entry.fieldChanges?.length ?? 0 })}
            </span>
          ) : null}
        </div>
      </div>
    </button>
  );
}

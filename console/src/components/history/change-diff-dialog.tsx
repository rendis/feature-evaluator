import { useTranslation } from 'react-i18next';

import type { ChangeEntry } from '@/api/changelog';

import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { useLocaleFormatters } from '@/hooks/use-locale-formatters';

interface ChangeDiffDialogProps {
  entry: ChangeEntry | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) return '---';
  if (typeof value === 'object') return JSON.stringify(value, null, 2);
  return String(value);
}

export function ChangeDiffDialog({ entry, open, onOpenChange }: ChangeDiffDialogProps) {
  const { t } = useTranslation('history');
  const { formatDateTime } = useLocaleFormatters();

  if (!entry) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{t('diff.title')}</DialogTitle>
          <DialogDescription>
            {t(`action.${entry.action}`)} {t(`entityType.${entry.entityType}`)} &mdash;{' '}
            {entry.entityKey}
          </DialogDescription>
        </DialogHeader>

        <div className="text-muted-foreground flex flex-wrap items-center gap-3 text-xs">
          <span>{t('diff.by', { actor: entry.actor })}</span>
          <Badge variant="outline" className="text-xs">
            {entry.actorType}
          </Badge>
          <span>{formatDateTime(entry.createdAt)}</span>
        </div>

        {entry.fieldChanges && entry.fieldChanges.length > 0 ? (
          <div className="space-y-3">
            {entry.fieldChanges.map((fc, i) => (
              <div key={i} className="rounded-md border">
                <div className="border-b bg-muted/50 px-3 py-2">
                  <span className="font-mono text-sm font-medium">{fc.field}</span>
                </div>
                <div className="grid grid-cols-2 gap-px bg-border">
                  <div className="bg-danger-soft/35 p-3">
                    <p className="text-muted-foreground mb-1 text-xs">{t('diff.old')}</p>
                    <pre className="whitespace-pre-wrap break-all font-mono text-xs">
                      {formatValue(fc.oldValue)}
                    </pre>
                  </div>
                  <div className="bg-success-soft/35 p-3">
                    <p className="text-muted-foreground mb-1 text-xs">{t('diff.new')}</p>
                    <pre className="whitespace-pre-wrap break-all font-mono text-xs">
                      {formatValue(fc.newValue)}
                    </pre>
                  </div>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-muted-foreground text-sm">{t('diff.noChanges')}</p>
        )}
      </DialogContent>
    </Dialog>
  );
}

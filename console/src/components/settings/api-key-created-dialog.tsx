import { AlertTriangle, Check, Copy } from 'lucide-react';
import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';

// eslint-disable-next-line @typescript-eslint/no-empty-function
const noop = () => {};

interface ApiKeyCreatedDialogProps {
  open: boolean;
  rawKey: string;
  onDismiss: () => void;
}

export function ApiKeyCreatedDialog({ open, rawKey, onDismiss }: ApiKeyCreatedDialogProps) {
  const { t } = useTranslation('settings');
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async () => {
    await navigator.clipboard.writeText(rawKey);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [rawKey]);

  return (
    <Dialog open={open} onOpenChange={noop}>
      <DialogContent
        className="sm:max-w-md"
        onPointerDownOutside={(e) => e.preventDefault()}
        onEscapeKeyDown={(e) => e.preventDefault()}
        onInteractOutside={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>{t('apiKeys.keyCreated')}</DialogTitle>
          <DialogDescription>{t('apiKeys.keyWarning')}</DialogDescription>
        </DialogHeader>

        <div className="bg-warning/10 border-warning/30 flex items-start gap-2 rounded-md border p-3">
          <AlertTriangle className="text-warning mt-0.5 h-4 w-4 shrink-0" />
          <p className="text-sm">{t('apiKeys.keyWarning')}</p>
        </div>

        <div className="flex items-center gap-2">
          <code className="bg-muted flex-1 overflow-x-auto rounded-md p-3 font-mono text-sm break-all">
            {rawKey}
          </code>
          <Button variant="outline" size="icon" onClick={handleCopy} className="shrink-0">
            {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
          </Button>
        </div>

        <DialogFooter>
          <Button onClick={onDismiss}>{t('apiKeys.keyCopied')}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

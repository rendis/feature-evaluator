import { AlertCircle } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { getVisibleErrorMessage } from '@/lib/display-error';
import { Button } from '@/components/ui/button';

interface ErrorStateProps {
  message?: string;
  onRetry?: () => void;
}

export function ErrorState({ message, onRetry }: ErrorStateProps) {
  const { t } = useTranslation();

  return (
    <div className="fe-empty-surface flex flex-col items-center justify-center px-6 py-16 text-center">
      <AlertCircle className="text-destructive mb-4 h-10 w-10" />
      <h3 className="text-content-strong text-lg font-semibold">
        {message ?? t('errors.unknown')}
      </h3>
      {onRetry ? (
        <Button variant="outline" onClick={onRetry} className="mt-4">
          {t('actions.retry')}
        </Button>
      ) : null}
    </div>
  );
}

export function ApiErrorState({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  const { t } = useTranslation();

  return (
    <ErrorState
      message={getVisibleErrorMessage(error, t('errors.unknown'))}
      onRetry={onRetry}
    />
  );
}

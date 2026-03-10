import { useCallback, useMemo, useState, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

import { LoadingModal } from '@/components/shared/loading-modal';
import {
  GlobalLoadingContext,
  type GlobalLoadingOptions,
  type LoadingEntry,
} from '@/hooks/global-loading-context';

export function GlobalLoadingProvider({ children }: { children: ReactNode }) {
  const { t } = useTranslation('common');
  const [entries, setEntries] = useState<LoadingEntry[]>([]);

  const show = useCallback((id: symbol, options: GlobalLoadingOptions) => {
    setEntries((previous) => [...previous.filter((entry) => entry.id !== id), { id, options }]);
  }, []);

  const hide = useCallback((id: symbol) => {
    setEntries((previous) => previous.filter((entry) => entry.id !== id));
  }, []);

  const value = useMemo(() => ({ hide, show }), [hide, show]);
  const activeEntry = entries.at(-1);

  return (
    <GlobalLoadingContext.Provider value={value}>
      {children}
      <LoadingModal
        open={!!activeEntry}
        title={
          activeEntry?.options.title ??
          t('loadingModal.savingTitle', { defaultValue: 'Guardando cambios' })
        }
        description={
          activeEntry?.options.description ??
          t('loadingModal.description', {
            defaultValue: 'Espera un momento mientras completamos la operacion.',
          })
        }
      />
    </GlobalLoadingContext.Provider>
  );
}

import { useContext, useEffect, useMemo, useRef } from 'react';
import { useTranslation } from 'react-i18next';

import {
  GlobalLoadingContext,
  type GlobalLoadingOptions,
  type SubmissionLoadingAction,
} from '@/hooks/global-loading-context';

export function useGlobalLoadingModal(active: boolean, options: GlobalLoadingOptions = {}) {
  const context = useContext(GlobalLoadingContext);
  const idRef = useRef<symbol>(Symbol('global-loading-modal'));
  const normalizedOptions = useMemo(
    () => ({
      description: options.description,
      title: options.title,
    }),
    [options.description, options.title],
  );

  if (!context) {
    throw new Error('useGlobalLoadingModal must be used within GlobalLoadingProvider');
  }

  useEffect(() => {
    const id = idRef.current;

    if (active) {
      context.show(id, normalizedOptions);
    } else {
      context.hide(id);
    }

    return () => context.hide(id);
  }, [active, context, normalizedOptions]);
}

export function useSubmissionLoadingModal(
  active: boolean,
  action: SubmissionLoadingAction = 'save',
) {
  const { t } = useTranslation('common');

  const titleByAction: Record<SubmissionLoadingAction, string> = {
    create: t('loadingModal.creatingTitle', { defaultValue: 'Creando registro' }),
    save: t('loadingModal.savingTitle', { defaultValue: 'Guardando cambios' }),
    update: t('loadingModal.updatingTitle', { defaultValue: 'Actualizando registro' }),
  };

  useGlobalLoadingModal(active, {
    title: titleByAction[action],
    description: t('loadingModal.description', {
      defaultValue: 'Espera un momento mientras completamos la operacion.',
    }),
  });
}

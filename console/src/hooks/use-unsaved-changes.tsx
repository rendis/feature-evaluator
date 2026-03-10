import { useBlocker, useNavigate } from '@tanstack/react-router';
import { useCallback, useEffect, useRef, useState } from 'react';

import { UnsavedChangesDialog } from '@/components/shared/unsaved-changes-dialog';

interface UseUnsavedChangesOptions {
  isDirty: boolean;
  backTo: string;
  backParams?: Record<string, string>;
  blockNavigation?: boolean;
}

export function useUnsavedChanges({
  isDirty,
  backTo,
  backParams,
  blockNavigation = false,
}: UseUnsavedChangesOptions) {
  const navigate = useNavigate();
  const [showDialog, setShowDialog] = useState(false);

  // Sync to refs via effect (not during render) so callbacks stay stable
  const isDirtyRef = useRef(isDirty);
  const backToRef = useRef(backTo);
  const backParamsRef = useRef(backParams);

  useEffect(() => {
    isDirtyRef.current = isDirty;
    backToRef.current = backTo;
    backParamsRef.current = backParams;
  });

  const navigationBlocker = useBlocker({
    shouldBlockFn: () => isDirtyRef.current,
    enableBeforeUnload: isDirty,
    disabled: !blockNavigation,
    withResolver: true,
  });

  // Browser beforeunload
  useEffect(() => {
    if (blockNavigation || !isDirty) return;
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault();
    };
    window.addEventListener('beforeunload', handler);
    return () => window.removeEventListener('beforeunload', handler);
  }, [blockNavigation, isDirty]);

  const goBack = useCallback(() => {
    void navigate({ to: backToRef.current, params: backParamsRef.current });
  }, [navigate]);

  const handleBack = useCallback(() => {
    if (blockNavigation) {
      goBack();
      return;
    }

    if (isDirtyRef.current) {
      setShowDialog(true);
    } else {
      goBack();
    }
  }, [blockNavigation, goBack]);

  const confirmLeave = useCallback(() => {
    setShowDialog(false);

    if (navigationBlocker.status === 'blocked') {
      navigationBlocker.proceed?.();
      return;
    }

    goBack();
  }, [goBack, navigationBlocker]);

  const cancelLeave = useCallback(() => {
    setShowDialog(false);

    if (navigationBlocker.status === 'blocked') {
      navigationBlocker.reset?.();
    }
  }, [navigationBlocker]);

  const dialogOpen = showDialog || navigationBlocker.status === 'blocked';

  const UnsavedDialog = useCallback(
    () => <UnsavedChangesDialog open={dialogOpen} onLeave={confirmLeave} onStay={cancelLeave} />,
    [dialogOpen, confirmLeave, cancelLeave],
  );

  const markClean = useCallback(() => {
    isDirtyRef.current = false;
  }, []);

  return { handleBack, UnsavedDialog, markClean };
}

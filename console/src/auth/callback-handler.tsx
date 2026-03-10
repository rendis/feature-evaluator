import { useNavigate } from '@tanstack/react-router';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { ApiClientError, setTokenProvider } from '@/api/client';
import { membersApi } from '@/api/members';
import { useAuth } from '@/hooks/use-auth';

type CallbackState = 'processing' | 'error' | 'access-denied' | 'service-unavailable';

async function verifyMembership(): Promise<'ok' | 'access-denied' | 'service-unavailable'> {
  let retries = 2;
  while (retries > 0) {
    try {
      await membersApi.me();
      return 'ok';
    } catch (err) {
      if (err instanceof ApiClientError && (err.status === 403 || err.status === 404)) {
        return 'access-denied';
      }
      retries--;
      if (retries > 0) await new Promise((r) => setTimeout(r, 1000));
    }
  }
  return 'service-unavailable';
}

export function CallbackHandler() {
  const { t } = useTranslation('auth');
  const { userManager } = useAuth();
  const navigate = useNavigate();
  const [state, setState] = useState<CallbackState>('processing');

  useEffect(() => {
    const processCallback = async () => {
      if (!userManager) return;
      try {
        const user = await userManager.signinRedirectCallback();
        setTokenProvider(() => user.access_token);
        const result = await verifyMembership();
        if (result === 'ok') {
          void navigate({ to: '/' });
        } else {
          setState(result);
        }
      } catch {
        setState('error');
      }
    };

    void processCallback();
  }, [userManager, navigate]);

  if (state === 'processing') {
    return (
      <div className="flex h-screen flex-col items-center justify-center gap-4">
        <div className="border-primary h-8 w-8 animate-spin rounded-full border-4 border-t-transparent" />
        <p className="text-muted-foreground">{t('callback.processing')}</p>
      </div>
    );
  }

  if (state === 'access-denied') {
    return (
      <div className="flex h-screen flex-col items-center justify-center gap-4">
        <h1 className="text-2xl font-bold">{t('accessDenied.title')}</h1>
        <p className="text-muted-foreground">{t('accessDenied.description')}</p>
      </div>
    );
  }

  if (state === 'service-unavailable') {
    return (
      <div className="flex h-screen flex-col items-center justify-center gap-4">
        <h1 className="text-2xl font-bold">{t('serviceUnavailable.title')}</h1>
        <p className="text-muted-foreground">{t('serviceUnavailable.description')}</p>
      </div>
    );
  }

  return (
    <div className="flex h-screen flex-col items-center justify-center gap-4">
      <h1 className="text-2xl font-bold">{t('callback.error')}</h1>
    </div>
  );
}

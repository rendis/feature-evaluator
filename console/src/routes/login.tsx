import { createFileRoute, Navigate } from '@tanstack/react-router';
import { LogIn } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { FullScreenSpinner } from '@/components/shared/full-screen-spinner';
import { Button } from '@/components/ui/button';
import { useAuth } from '@/hooks/use-auth';

export const Route = createFileRoute('/login')({
  component: LoginPage,
});

function LoginPage() {
  const { t } = useTranslation('auth');
  const { isAuthenticated, isLoading, login } = useAuth();

  if (isLoading) {
    return <FullScreenSpinner />;
  }

  if (isAuthenticated) {
    return <Navigate to="/" />;
  }

  return (
    <div className="flex h-screen flex-col items-center justify-center gap-6">
      <div className="text-center">
        <h1 className="text-3xl font-bold">{t('login.title')}</h1>
        <p className="text-muted-foreground mt-2">{t('login.subtitle')}</p>
      </div>
      <Button size="lg" onClick={() => void login()}>
        <LogIn className="mr-2 h-4 w-4" />
        {t('login.button')}
      </Button>
    </div>
  );
}

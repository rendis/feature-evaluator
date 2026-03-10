import { createFileRoute, Link } from '@tanstack/react-router';
import { ShieldAlert } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { Button } from '@/components/ui/button';

export const Route = createFileRoute('/auth/access-denied')({
  component: AccessDeniedPage,
});

function AccessDeniedPage() {
  const { t } = useTranslation('auth');

  return (
    <div className="flex h-screen flex-col items-center justify-center gap-4 text-center">
      <ShieldAlert className="text-destructive h-12 w-12" />
      <h1 className="text-2xl font-bold">{t('accessDenied.title')}</h1>
      <p className="text-muted-foreground max-w-sm">{t('accessDenied.description')}</p>
      <Button variant="outline" asChild>
        <Link to="/login">{t('accessDenied.backToLogin')}</Link>
      </Button>
    </div>
  );
}

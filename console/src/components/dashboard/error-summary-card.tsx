import { useSuspenseQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { AlertTriangle } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { dashboardQueries } from '@/queries/dashboard-queries';

export function ErrorSummaryCard() {
  const { t } = useTranslation('dashboard');
  const { data } = useSuspenseQuery(dashboardQueries.errorSummary());

  return (
    <div className="rounded-lg border">
      <div className="flex items-center justify-between border-b px-4 py-3">
        <div className="flex items-center gap-2">
          <AlertTriangle className="text-destructive h-4 w-4" />
          <h3 className="text-sm font-semibold">{t('errors.title')}</h3>
        </div>
        <Badge variant={data.total > 0 ? 'destructive' : 'secondary'}>
          {data.total}
        </Badge>
      </div>
      <div className="p-4">
        {data.byType.length === 0 ? (
          <p className="text-muted-foreground text-sm">{t('errors.empty')}</p>
        ) : (
          <div className="space-y-2">
            {data.byType.map((entry) => (
              <div key={entry.type} className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">{entry.type}</span>
                <span className="font-medium">{entry.count}</span>
              </div>
            ))}
          </div>
        )}
        <Button variant="outline" asChild className="mt-4 w-full">
          <Link to="/audit">{t('errors.viewAll')}</Link>
        </Button>
      </div>
    </div>
  );
}

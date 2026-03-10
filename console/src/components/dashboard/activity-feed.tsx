import { useSuspenseQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';

import type { RecentActivity } from '@/api/dashboard';

import { Badge } from '@/components/ui/badge';
import { useLocaleFormatters } from '@/hooks/use-locale-formatters';
import { dashboardQueries } from '@/queries/dashboard-queries';

const ACTIVITY_VARIANT: Record<string, 'default' | 'secondary' | 'outline'> = {
  feature_created: 'default',
  feature_updated: 'secondary',
  feature_toggled: 'outline',
  rule_created: 'default',
  rule_updated: 'secondary',
};

function ActivityItem({ item }: { item: RecentActivity }) {
  const { t } = useTranslation('dashboard');
  const { formatDate } = useLocaleFormatters();
  const variant = ACTIVITY_VARIANT[item.type] ?? 'outline';

  return (
    <div className="flex items-start gap-3 border-b py-3 last:border-0">
      <Badge variant={variant} className="mt-0.5 shrink-0">
        {t(`activity.types.${item.type}`)}
      </Badge>
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium">{item.description}</p>
        <p className="text-muted-foreground text-xs">
          {item.featureKey} &middot; {item.createdBy} &middot;{' '}
          {formatDate(item.createdAt)}
        </p>
      </div>
    </div>
  );
}

export function ActivityFeed() {
  const { t } = useTranslation('dashboard');
  const { data: activities } = useSuspenseQuery(dashboardQueries.activity());

  if (activities.length === 0) {
    return (
      <div className="text-muted-foreground rounded-lg border border-dashed p-8 text-center text-sm">
        {t('activity.empty')}
      </div>
    );
  }

  return (
    <div className="rounded-lg border">
      <div className="border-b px-4 py-3">
        <h3 className="text-sm font-semibold">{t('activity.title')}</h3>
      </div>
      <div className="divide-y px-4">
        {activities.map((item) => (
          <ActivityItem key={item.id} item={item} />
        ))}
      </div>
    </div>
  );
}

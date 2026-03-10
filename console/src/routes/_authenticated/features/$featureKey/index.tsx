import { useQuery, useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute, Link } from '@tanstack/react-router';
import { Plus } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { Tag } from '@/api/types';

import { FeatureDetailHeader } from '@/components/features/feature-detail-header';
import { ChangeTimeline } from '@/components/history/change-timeline';
import { RuleList } from '@/components/rules/rule-list';
import { PendingSchedules } from '@/components/schedules/pending-schedules';
import { EmptyState } from '@/components/shared/empty-state';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { PermissionButton } from '@/components/shared/permission-button';
import { TagBadge } from '@/components/shared/tag-badge';
import { changelogQueries } from '@/queries/changelog-queries';
import { featureQueries } from '@/queries/feature-queries';
import { ruleQueries } from '@/queries/rule-queries';

export const Route = createFileRoute('/_authenticated/features/$featureKey/')({
  component: FeatureDetailPage,
  pendingComponent: () => <LoadingSkeleton rows={8} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function FeatureDetailPage() {
  const { featureKey } = Route.useParams();
  const { t } = useTranslation(['features', 'rules']);
  const { data: feature } = useSuspenseQuery(featureQueries.detail(featureKey));
  const { data: rules = [] } = useSuspenseQuery(ruleQueries.list(featureKey));

  return (
    <div className="space-y-8">
      <FeatureDetailHeader feature={feature} />

      <FeatureMetadataSection feature={feature} />

      <PendingSchedules featureKey={featureKey} />

      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">
            {t('title', { ns: 'rules' })}
            <span className="text-muted-foreground ml-2 text-sm font-normal">
              ({t('detail.rulesCount', { count: rules.length })})
            </span>
          </h2>
          <PermissionButton permission="features.write" size="sm" asChild>
            <Link to="/features/$featureKey/rules/new" params={{ featureKey }}>
              <Plus className="mr-1 h-3 w-3" />
              {t('createRule', { ns: 'rules' })}
            </Link>
          </PermissionButton>
        </div>

        {rules.length === 0 ? (
          <EmptyState
            title={t('empty.title', { ns: 'rules' })}
            description={t('empty.description', { ns: 'rules' })}
            action={
              <PermissionButton permission="features.write" size="sm" asChild>
                <Link to="/features/$featureKey/rules/new" params={{ featureKey }}>
                  {t('empty.cta', { ns: 'rules' })}
                </Link>
              </PermissionButton>
            }
          />
        ) : (
          <RuleList featureKey={featureKey} rules={rules} />
        )}
      </div>

      <RecentChangesSection featureKey={featureKey} />
    </div>
  );
}

function RecentChangesSection({ featureKey }: { featureKey: string }) {
  const { t } = useTranslation('history');
  const { data } = useQuery(changelogQueries.byEntity('feature', featureKey, { pageSize: 10 }));

  const entries = data?.data ?? [];

  if (entries.length === 0) return null;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">{t('recent.title')}</h2>
        <Link to="/history" className="text-primary text-sm hover:underline">
          {t('recent.viewAll')}
        </Link>
      </div>
      <ChangeTimeline entries={entries} />
    </div>
  );
}

function FeatureMetadataSection({ feature }: { feature: { description: string; tags: Tag[]; defaultValue: unknown; valueType: string } }) {
  const { t } = useTranslation('features');

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {feature.description ? (
        <div>
          <p className="text-muted-foreground text-xs">{t('fields.description')}</p>
          <p className="text-sm">{feature.description}</p>
        </div>
      ) : null}
      <div>
        <p className="text-muted-foreground text-xs">{t('fields.defaultValue')}</p>
        <p className="font-mono text-sm">{String(feature.defaultValue)}</p>
      </div>
      {feature.tags.length > 0 ? (
        <div>
          <p className="text-muted-foreground mb-1 text-xs">{t('fields.tags')}</p>
          <div className="flex flex-wrap gap-1">
            {feature.tags.map((tag) => (
              <TagBadge key={tag.key} tag={tag} size="sm" />
            ))}
          </div>
        </div>
      ) : null}
    </div>
  );
}

import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { ShieldAlert } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { FeatureDetailHeader } from '@/components/features/feature-detail-header';
import { FeatureObservabilityView } from '@/components/features/feature-observability-view';
import { EmptyState } from '@/components/shared/empty-state';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { featureQueries } from '@/queries/feature-queries';
import { usePermissions } from '@/hooks/use-permissions';

export const Route = createFileRoute('/_authenticated/features/$featureKey/observability')({
  component: FeatureObservabilityPage,
  pendingComponent: () => <LoadingSkeleton rows={8} />,
  errorComponent: ({ error }) => <ApiErrorState error={error} />,
});

interface FeatureObservabilityPageProps {
  featureKey?: string;
}

export function FeatureObservabilityPage({ featureKey: explicitFeatureKey }: FeatureObservabilityPageProps = {}) {
  const routeParams = Route.useParams();
  const featureKey = explicitFeatureKey ?? routeParams.featureKey;
  const { can } = usePermissions();

  if (!can('audit.read')) {
    return <ObservabilityAccessDenied />;
  }

  return <FeatureObservabilityContent featureKey={featureKey} />;
}

function FeatureObservabilityContent({ featureKey }: { featureKey: string }) {
  const { data: feature } = useSuspenseQuery(featureQueries.detail(featureKey));
  return (
    <div className="space-y-8">
      <FeatureDetailHeader feature={feature} />
      <FeatureObservabilityView feature={feature} featureKey={featureKey} />
    </div>
  );
}

function ObservabilityAccessDenied() {
  const { t } = useTranslation('auth');

  return (
    <EmptyState
      icon={<ShieldAlert className="h-10 w-10" />}
      title={t('accessDenied.title', { defaultValue: 'Access denied' })}
      description={t('accessDenied.description', {
        defaultValue: 'You do not have permission to view feature observability.',
      })}
    />
  );
}

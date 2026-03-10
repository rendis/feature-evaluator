import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';

import { FeatureBuilder } from '@/components/features/feature-builder';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { featureQueries } from '@/queries/feature-queries';

export const Route = createFileRoute('/_authenticated/features/$featureKey/edit')({
  component: EditFeaturePage,
  pendingComponent: () => <LoadingSkeleton rows={6} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function EditFeaturePage() {
  const { featureKey } = Route.useParams();
  const { data: feature } = useSuspenseQuery(featureQueries.detail(featureKey));

  return <FeatureBuilder key={feature.key} feature={feature} />;
}

import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';

import { ExternalApiBuilder } from '@/components/external-apis/external-api-builder';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { externalApiQueries } from '@/queries/external-api-queries';

export const Route = createFileRoute('/_authenticated/settings/external-apis/$key')({
  component: ExternalApiDetailPage,
  pendingComponent: () => <LoadingSkeleton rows={8} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function ExternalApiDetailPage() {
  const { key } = Route.useParams();
  const { data: externalApi } = useSuspenseQuery(externalApiQueries.detail(key));

  return <ExternalApiBuilder key={externalApi.key} externalApi={externalApi} />;
}

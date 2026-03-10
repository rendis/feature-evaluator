import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';

import { AuthProfileBuilder } from '@/components/auth-profiles/auth-profile-builder';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { authProfileQueries } from '@/queries/auth-profile-queries';

export const Route = createFileRoute('/_authenticated/settings/auth-profiles/$key')({
  component: AuthProfileDetailPage,
  pendingComponent: () => <LoadingSkeleton rows={8} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function AuthProfileDetailPage() {
  const { key } = Route.useParams();
  const { data: profile } = useSuspenseQuery(authProfileQueries.detail(key));

  return <AuthProfileBuilder key={profile.key} profile={profile} />;
}

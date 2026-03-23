import { createFileRoute } from '@tanstack/react-router';

import { SecurityPolicyRoutePage } from '@/components/security-policies/security-policy-page';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';

export const Route = createFileRoute('/_authenticated/settings/security')({
  component: SecurityPolicyRoutePage,
  pendingComponent: () => <LoadingSkeleton rows={4} />,
  errorComponent: ({ error }) => <ApiErrorState error={error} />,
});

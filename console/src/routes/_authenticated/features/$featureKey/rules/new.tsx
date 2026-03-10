import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';

import { RuleBuilder } from '@/components/rules/rule-builder';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { featureQueries } from '@/queries/feature-queries';
import { ruleQueries } from '@/queries/rule-queries';

export const Route = createFileRoute('/_authenticated/features/$featureKey/rules/new')({
  component: NewRulePage,
  pendingComponent: () => <LoadingSkeleton rows={6} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function NewRulePage() {
  const { featureKey } = Route.useParams();
  const { data: feature } = useSuspenseQuery(featureQueries.detail(featureKey));
  const { data: rules = [] } = useSuspenseQuery(ruleQueries.list(featureKey));
  const nextPriority = rules.length > 0 ? Math.max(...rules.map((r) => r.priority)) + 1 : 1;

  return <RuleBuilder feature={feature} nextPriority={nextPriority} />;
}

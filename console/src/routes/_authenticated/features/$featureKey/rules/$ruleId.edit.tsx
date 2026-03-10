import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { useTranslation } from 'react-i18next';

import { RuleBuilder } from '@/components/rules/rule-builder';
import { ApiErrorState, ErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { featureQueries } from '@/queries/feature-queries';
import { ruleQueries } from '@/queries/rule-queries';

export const Route = createFileRoute('/_authenticated/features/$featureKey/rules/$ruleId/edit')({
  component: EditRulePage,
  pendingComponent: () => <LoadingSkeleton rows={6} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function EditRulePage() {
  const { featureKey, ruleId } = Route.useParams();
  const { t } = useTranslation('rules');
  const { data: feature } = useSuspenseQuery(featureQueries.detail(featureKey));
  const { data: rules = [] } = useSuspenseQuery(ruleQueries.list(featureKey));
  const rule = rules.find((r) => r.id === ruleId);

  if (!rule) {
    return <ErrorState message={t('errors.notFound', { ns: 'common' })} />;
  }

  return <RuleBuilder key={rule.id} feature={feature} rule={rule} />;
}

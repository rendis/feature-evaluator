import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { useTranslation } from 'react-i18next';

import { RuleBuilder } from '@/components/rules/rule-builder';
import { createClonedDraft } from '@/components/rules/rule-builder-utils';
import { ApiErrorState, ErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { featureQueries } from '@/queries/feature-queries';
import { ruleQueries } from '@/queries/rule-queries';

interface NewRuleSearch {
  cloneFrom?: string;
}

export const Route = createFileRoute('/_authenticated/features/$featureKey/rules/new')({
  validateSearch: (search: Record<string, unknown>): NewRuleSearch => ({
    cloneFrom:
      typeof search.cloneFrom === 'string' && search.cloneFrom.trim().length > 0
        ? search.cloneFrom
        : undefined,
  }),
  component: NewRulePage,
  pendingComponent: () => <LoadingSkeleton rows={6} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

export function NewRulePage() {
  const { featureKey } = Route.useParams();
  const { cloneFrom } = Route.useSearch();
  const { t } = useTranslation('rules');
  const { data: feature } = useSuspenseQuery(featureQueries.detail(featureKey));
  const { data: rules = [] } = useSuspenseQuery(ruleQueries.list(featureKey));
  const nextPriority = rules.length > 0 ? Math.max(...rules.map((r) => r.priority)) + 1 : 1;
  const sourceRule = cloneFrom ? rules.find((rule) => rule.id === cloneFrom) : undefined;

  if (cloneFrom && !sourceRule) {
    return (
      <ErrorState
        message={t('clone.sourceNotFound', {
          defaultValue: 'No encontramos la regla que quieres clonar.',
        })}
      />
    );
  }

  const initialDraft = sourceRule
    ? createClonedDraft(
        sourceRule,
        feature,
        nextPriority,
        t('clone.nameTemplate', {
          defaultValue: 'Copia de {{name}}',
          name: sourceRule.name,
        }),
      )
    : undefined;

  return (
    <RuleBuilder
      key={cloneFrom ? `clone:${cloneFrom}` : 'new'}
      feature={feature}
      nextPriority={nextPriority}
      initialDraft={initialDraft}
    />
  );
}

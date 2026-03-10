import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';


import { ExpressionModeSwitch } from './expression-editor/expression-mode-switch';

import type { ConditionGroup } from './expression-editor/types';

import { Label } from '@/components/ui/label';
import { useMediaQuery } from '@/hooks/use-media-query';
import { expressionQueries } from '@/queries/expression-queries';

interface ExpressionSectionProps {
  expression: string;
  conditionGroup: ConditionGroup;
  onExpressionChange: (expr: string) => void;
  onGroupChange: (group: ConditionGroup) => void;
}

export function ExpressionSection({
  expression,
  conditionGroup,
  onExpressionChange,
  onGroupChange,
}: ExpressionSectionProps) {
  const { t } = useTranslation('rules');
  const isDesktop = useMediaQuery('(min-width: 1024px)');
  const { data: schema } = useQuery(expressionQueries.schema());

  const fields = schema?.fields.map((f) => f.name) ?? [
    'user.id',
    'user.email',
    'user.plan',
    'user.active',
    'tenant.id',
    'campus.id',
    'program.id',
    'authenticated',
  ];

  return (
    <div className="space-y-3">
      <Label>{t('fields.expression')}</Label>
      <ExpressionModeSwitch
        expression={expression}
        conditionGroup={conditionGroup}
        onExpressionChange={onExpressionChange}
        onGroupChange={onGroupChange}
        fields={fields}
        showTextMode={isDesktop}
      />
    </div>
  );
}

import { useQuery } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';


import { ConditionGroupEditor } from './condition-group';
import { emptyGroup, serializeExpression } from './types';

import type { ConditionGroup } from './types';

import { expressionQueries } from '@/queries/expression-queries';

interface ExpressionEditorProps {
  value: ConditionGroup;
  onChange: (group: ConditionGroup, expressionStr: string) => void;
}

export function ExpressionEditor({ value, onChange }: ExpressionEditorProps) {
  const { t } = useTranslation('rules');
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

  const group = value.conditions.length > 0 ? value : emptyGroup();

  const handleChange = (updated: ConditionGroup) => {
    const expr = serializeExpression(updated);
    onChange(updated, expr);
  };

  return (
    <div className="space-y-3">
      <p className="text-sm font-medium">{t('expression.visualMode')}</p>
      <ConditionGroupEditor group={group} onChange={handleChange} fields={fields} />
    </div>
  );
}

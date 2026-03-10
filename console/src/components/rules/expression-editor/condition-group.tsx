import { Plus } from 'lucide-react';
import { useTranslation } from 'react-i18next';


import { ConditionRow } from './condition-row';
import { emptyCondition, emptyGroup, isConditionGroup } from './types';

import type { Condition, ConditionGroup as ConditionGroupType, Connector } from './types';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';

const MAX_DEPTH = 3;

interface ConditionGroupProps {
  group: ConditionGroupType;
  onChange: (updated: ConditionGroupType) => void;
  depth?: number;
  fields: string[];
}

export function ConditionGroupEditor({ group, onChange, depth = 0, fields }: ConditionGroupProps) {
  const { t } = useTranslation('rules');

  const updateItem = (index: number, item: Condition | ConditionGroupType) => {
    const next = [...group.conditions];
    next[index] = item;
    onChange({ ...group, conditions: next });
  };

  const removeItem = (index: number) => {
    const next = group.conditions.filter((_, i) => i !== index);
    onChange({ ...group, conditions: next.length > 0 ? next : [emptyCondition()] });
  };

  const addCondition = () => {
    onChange({ ...group, conditions: [...group.conditions, emptyCondition()] });
  };

  const addGroup = () => {
    onChange({ ...group, conditions: [...group.conditions, emptyGroup()] });
  };

  const toggleConnector = () => {
    const next: Connector = group.connector === 'and' ? 'or' : 'and';
    onChange({ ...group, connector: next });
  };

  return (
    <div className={depth > 0 ? 'ml-4 rounded-md border border-dashed p-3' : ''}>
      <div className="space-y-2">
        {group.conditions.map((item, i) => (
          <div key={i}>
            {i > 0 ? (
              <button
                type="button"
                onClick={toggleConnector}
                className="my-2 flex items-center justify-center"
              >
                <Badge variant="secondary" className="cursor-pointer select-none uppercase">
                  {group.connector === 'and' ? t('expression.and') : t('expression.or')}
                </Badge>
              </button>
            ) : null}
            {isConditionGroup(item) ? (
              <ConditionGroupEditor
                group={item}
                onChange={(updated) => updateItem(i, updated)}
                depth={depth + 1}
                fields={fields}
              />
            ) : (
              <ConditionRow
                condition={item}
                onChange={(updated) => updateItem(i, updated)}
                onRemove={() => removeItem(i)}
                fields={fields}
              />
            )}
          </div>
        ))}
      </div>
      <div className="mt-3 flex gap-2">
        <Button type="button" variant="outline" size="sm" onClick={addCondition}>
          <Plus className="mr-1 h-3 w-3" />
          {t('expression.addCondition')}
        </Button>
        {depth < MAX_DEPTH - 1 ? (
          <Button type="button" variant="outline" size="sm" onClick={addGroup}>
            <Plus className="mr-1 h-3 w-3" />
            {t('expression.addGroup')}
          </Button>
        ) : (
          <span className="text-muted-foreground self-center text-xs">
            {t('expression.maxDepthReached')}
          </span>
        )}
      </div>
    </div>
  );
}

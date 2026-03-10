import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';


import { ConditionGroupEditor } from './condition-group';
import { ExpressionTester } from './expression-tester';
import { ExpressionTextEditor } from './expression-text-editor';
import { emptyGroup, serializeExpression } from './types';

import type { ConditionGroup } from './types';

import { Button } from '@/components/ui/button';

interface ExpressionModeSwitchProps {
  expression: string;
  conditionGroup: ConditionGroup;
  onExpressionChange: (expr: string) => void;
  onGroupChange: (group: ConditionGroup) => void;
  fields: string[];
  showTextMode: boolean;
}

export function ExpressionModeSwitch({
  expression,
  conditionGroup,
  onExpressionChange,
  onGroupChange,
  fields,
  showTextMode,
}: ExpressionModeSwitchProps) {
  const { t } = useTranslation('rules');
  const [mode, setMode] = useState<'visual' | 'text'>('visual');

  const switchToText = () => {
    const expr = serializeExpression(conditionGroup);
    onExpressionChange(expr);
    setMode('text');
  };

  const switchToVisual = () => {
    toast.info(t('expression.parseWarning', { defaultValue: 'Text mode expression kept as-is' }));
    setMode('visual');
  };

  const handleVisualChange = (group: ConditionGroup) => {
    onGroupChange(group);
    onExpressionChange(serializeExpression(group));
  };

  const group = conditionGroup.conditions.length > 0 ? conditionGroup : emptyGroup();

  return (
    <div className="space-y-3">
      {showTextMode ? (
        <div className="flex gap-2">
          <Button
            type="button"
            variant={mode === 'visual' ? 'default' : 'outline'}
            size="sm"
            onClick={() => mode !== 'visual' && switchToVisual()}
          >
            {t('expression.visualMode')}
          </Button>
          <Button
            type="button"
            variant={mode === 'text' ? 'default' : 'outline'}
            size="sm"
            onClick={() => mode !== 'text' && switchToText()}
          >
            {t('expression.textMode')}
          </Button>
        </div>
      ) : null}

      {mode === 'visual' ? (
        <ConditionGroupEditor group={group} onChange={handleVisualChange} fields={fields} />
      ) : (
        <ExpressionTextEditor value={expression} onChange={onExpressionChange} />
      )}

      {expression ? <ExpressionTester expression={expression} /> : null}
    </div>
  );
}

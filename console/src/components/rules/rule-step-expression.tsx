import { useCallback } from 'react';

import { RuleConditionsCanvas } from './rule-conditions-canvas';

import type { RuleDraft } from './rule-builder-utils';
import type { RuleConditionsValue } from './rule-conditions-canvas';
import type { Feature } from '@/api/types';
import type { Dispatch, SetStateAction } from 'react';

interface StepExpressionProps {
  draft: RuleDraft;
  feature: Feature;
  onChange: Dispatch<SetStateAction<RuleDraft>>;
}

export function StepExpression({ draft, feature, onChange }: StepExpressionProps) {
  const handleConditionsChange = useCallback(
    (value: RuleConditionsValue) => {
      onChange((c) => ({
        ...c,
        expression: value.expression,
        metadata: value.metadata,
        sourceBindings: value.sourceBindings,
        externalApiBindings: value.externalApiBindings,
      }));
    },
    [onChange],
  );

  return (
    <div className="min-h-0 flex-1 overflow-y-auto px-1 py-1">
      <RuleConditionsCanvas
        feature={feature}
        initialExpression={draft.expression}
        initialMetadata={draft.metadata}
        initialSourceBindings={draft.sourceBindings}
        initialExternalApiBindings={draft.externalApiBindings}
        onChange={handleConditionsChange}
      />
    </div>
  );
}

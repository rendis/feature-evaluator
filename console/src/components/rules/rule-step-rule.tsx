import { useTranslation } from 'react-i18next';

import type { RuleDraft } from './rule-builder-utils';
import type { ValueType } from '@/api/types';
import type { Dispatch, SetStateAction } from 'react';

import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';

interface StepRuleProps {
  draft: RuleDraft;
  valueType: ValueType;
  onChange: Dispatch<SetStateAction<RuleDraft>>;
}

export function StepRule({ draft, valueType, onChange }: StepRuleProps) {
  const { t } = useTranslation('rules');

  return (
    <div className="min-h-0 flex-1 space-y-6 overflow-y-auto px-1 py-1">
      <div className="flex items-center gap-3">
        <Label htmlFor="rule-enabled">{t('fields.enabled')}</Label>
        <Switch
          id="rule-enabled"
          checked={draft.enabled}
          onCheckedChange={(v) => onChange((c) => ({ ...c, enabled: v }))}
          aria-label={t('fields.enabled')}
        />
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="name">{t('fields.name')}</Label>
          <Input
            id="name"
            value={draft.name}
            onChange={(e) => onChange((c) => ({ ...c, name: e.target.value }))}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="priority">{t('fields.priority')}</Label>
          <Input
            id="priority"
            type="number"
            min={1}
            max={10000}
            value={draft.priority}
            onChange={(e) => {
              const v = parseInt(e.target.value, 10);
              if (!isNaN(v)) onChange((c) => ({ ...c, priority: v }));
            }}
          />
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="value">{t('fields.value')}</Label>
        <RuleValueInput
          value={draft.value}
          valueType={valueType}
          onChange={(v) => onChange((c) => ({ ...c, value: v }))}
        />
      </div>

    </div>
  );
}

function RuleValueInput({
  value,
  valueType,
  onChange,
}: {
  value: string;
  valueType: ValueType;
  onChange: (value: string) => void;
}) {
  if (valueType === 'boolean') {
    return (
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger id="value">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="true">true</SelectItem>
          <SelectItem value="false">false</SelectItem>
        </SelectContent>
      </Select>
    );
  }
  if (valueType === 'json') {
    return (
      <textarea
        id="value"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        rows={4}
        className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      />
    );
  }
  return (
    <Input
      id="value"
      type={valueType === 'number' ? 'number' : 'text'}
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  );
}


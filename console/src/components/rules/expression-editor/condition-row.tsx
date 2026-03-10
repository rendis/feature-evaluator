import { Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { Condition, Operator } from './types';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';


const OPERATORS: { value: Operator; label: string }[] = [
  { value: '==', label: '==' },
  { value: '!=', label: '!=' },
  { value: '>', label: '>' },
  { value: '>=', label: '>=' },
  { value: '<', label: '<' },
  { value: '<=', label: '<=' },
  { value: 'contains', label: 'contains' },
  { value: 'startsWith', label: 'startsWith' },
  { value: 'endsWith', label: 'endsWith' },
  { value: 'matches', label: 'matches' },
  { value: 'in', label: 'in' },
  { value: 'not in', label: 'not in' },
];

interface ConditionRowProps {
  condition: Condition;
  onChange: (updated: Condition) => void;
  onRemove: () => void;
  fields: string[];
}

export function ConditionRow({ condition, onChange, onRemove, fields }: ConditionRowProps) {
  const { t } = useTranslation('rules');

  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
      <Select value={condition.field} onValueChange={(v) => onChange({ ...condition, field: v })}>
        <SelectTrigger className="sm:w-[200px]">
          <SelectValue placeholder={t('expression.selectField')} />
        </SelectTrigger>
        <SelectContent>
          {fields.map((f) => (
            <SelectItem key={f} value={f}>
              {f}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Select
        value={condition.operator}
        onValueChange={(v) => onChange({ ...condition, operator: v as Operator })}
      >
        <SelectTrigger className="sm:w-[140px]">
          <SelectValue placeholder={t('expression.selectOperator')} />
        </SelectTrigger>
        <SelectContent>
          {OPERATORS.map((op) => (
            <SelectItem key={op.value} value={op.value}>
              {op.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <Input
        value={condition.value}
        onChange={(e) => onChange({ ...condition, value: e.target.value })}
        placeholder={t('expression.enterValue')}
        className="sm:flex-1"
      />
      <Button variant="ghost" size="icon" onClick={onRemove} className="shrink-0">
        <Trash2 className="h-4 w-4" />
      </Button>
    </div>
  );
}

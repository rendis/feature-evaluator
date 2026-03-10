import { useTranslation } from 'react-i18next';

import type { FeatureDraft } from './feature-builder-utils';
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


interface StepValueProps {
  draft: FeatureDraft;
  isEditing: boolean;
  onChange: Dispatch<SetStateAction<FeatureDraft>>;
}

export function StepValue({ draft, isEditing, onChange }: StepValueProps) {
  const { t } = useTranslation('features');

  return (
    <div className="min-h-0 flex-1 space-y-6 overflow-y-auto px-1 py-1">
      <div className="grid gap-4 md:grid-cols-2 md:items-start">
        <div className="space-y-2">
          <Label>{t('fields.valueType')}</Label>
          <Select
            value={draft.valueType}
            onValueChange={(v) =>
              onChange((c) => ({
                ...c,
                valueType: v as ValueType,
                defaultValue: v === 'boolean' ? 'false' : '',
              }))
            }
            disabled={isEditing}
          >
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {(['boolean', 'string', 'number', 'json'] as const).map((vt) => (
                <SelectItem key={vt} value={vt}>
                  {t(`valueTypes.${vt}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label>{t('fields.defaultValue')}</Label>
          <DefaultValueInput
            value={draft.defaultValue}
            valueType={draft.valueType}
            onChange={(v) => onChange((c) => ({ ...c, defaultValue: v }))}
          />
        </div>
      </div>
    </div>
  );
}

function DefaultValueInput({
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
        <SelectTrigger>
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
        value={value}
        onChange={(e) => onChange(e.target.value)}
        rows={4}
        className="flex w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      />
    );
  }

  return (
    <Input
      type={valueType === 'number' ? 'number' : 'text'}
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  );
}

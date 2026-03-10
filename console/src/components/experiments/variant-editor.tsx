import { Plus, Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { Variant } from '@/api/types';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

interface VariantEditorProps {
  variants: Variant[];
  onChange: (variants: Variant[]) => void;
  disabled?: boolean;
}

export function VariantEditor({ variants, onChange, disabled }: VariantEditorProps) {
  const { t } = useTranslation('experiments');

  const addVariant = () => {
    onChange([...variants, { key: '', value: '', weight: 50 }]);
  };

  const removeVariant = (index: number) => {
    onChange(variants.filter((_, i) => i !== index));
  };

  const updateVariant = (index: number, field: keyof Variant, value: string | number) => {
    const updated = variants.map((v, i) =>
      i === index ? { ...v, [field]: field === 'weight' ? Number(value) : value } : v,
    );
    onChange(updated);
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <Label>{t('form.variants')}</Label>
        <Button type="button" variant="outline" size="sm" onClick={addVariant} disabled={disabled}>
          <Plus className="mr-1 h-3 w-3" />
          {t('form.addVariant')}
        </Button>
      </div>
      {variants.map((variant, index) => (
        <div key={index} className="flex items-end gap-2">
          <div className="flex-1 space-y-1">
            <Label className="text-xs">{t('form.variantKey')}</Label>
            <Input
              value={variant.key}
              onChange={(e) => updateVariant(index, 'key', e.target.value)}
              placeholder="control"
              className="font-mono text-sm"
              disabled={disabled}
            />
          </div>
          <div className="flex-1 space-y-1">
            <Label className="text-xs">{t('form.variantValue')}</Label>
            <Input
              value={String(variant.value ?? '')}
              onChange={(e) => updateVariant(index, 'value', e.target.value)}
              disabled={disabled}
            />
          </div>
          <div className="w-20 space-y-1">
            <Label className="text-xs">{t('form.weight')}</Label>
            <Input
              type="number"
              min={0}
              max={100}
              value={variant.weight}
              onChange={(e) => updateVariant(index, 'weight', e.target.value)}
              disabled={disabled}
            />
          </div>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="text-destructive shrink-0"
            onClick={() => removeVariant(index)}
            disabled={disabled || variants.length <= 2}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      ))}
    </div>
  );
}

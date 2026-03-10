import { X } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { FeatureDraft } from './feature-builder-utils';
import type { InputHeader, InputValueType } from '@/api/types';
import type { Dispatch, SetStateAction } from 'react';

import { Button } from '@/components/ui/button';
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
import { slugifyResourceKey } from '@/lib/resource-key';


interface StepInputsProps {
  draft: FeatureDraft;
  onChange: Dispatch<SetStateAction<FeatureDraft>>;
}

export function StepInputs({ draft, onChange }: StepInputsProps) {
  const { t } = useTranslation('features');

  const addHeader = () => {
    onChange((c) => ({
      ...c,
      headers: [
        ...c.headers,
        {
          headerName: '',
          expressionKey: '',
          label: '',
          type: 'string' as InputValueType,
          required: false,
          description: '',
        },
      ],
    }));
  };

  const updateHeader = <K extends keyof InputHeader>(index: number, key: K, value: InputHeader[K]) => {
    onChange((c) => ({
      ...c,
      headers: c.headers.map((h, i) => {
        if (i !== index) return h;
        const updated = { ...h, [key]: value };
        if (key === 'headerName') {
          updated.expressionKey = slugifyResourceKey(String(value));
        }
        return updated;
      }),
    }));
  };

  const removeHeader = (index: number) => {
    onChange((c) => ({
      ...c,
      headers: c.headers.filter((_, i) => i !== index),
    }));
  };

  return (
    <div className="min-h-0 flex-1 space-y-6 overflow-y-auto px-1 py-1">
      <div className="space-y-3">
        <div className="flex items-center justify-between gap-3">
          <div>
            <p className="text-sm font-medium">
              {t('form.headersTitle', { defaultValue: 'Headers esperados' })}
            </p>
            <p className="text-muted-foreground text-xs">
              {t('form.headersHelp', {
                defaultValue:
                  'Cada header se expone en expresiones como headers.mi_header.',
              })}
            </p>
          </div>
          <Button type="button" variant="outline" size="sm" onClick={addHeader}>
            {t('form.addHeader', { defaultValue: 'Agregar header' })}
          </Button>
        </div>

        <div className="space-y-3">
          {draft.headers.length === 0 ? (
            <div className="text-muted-foreground rounded-xl border border-dashed px-4 py-5 text-sm">
              {t('form.noHeaders', { defaultValue: 'No hay headers configurados todavia.' })}
            </div>
          ) : null}

          {draft.headers.map((header, index) => (
            <div
              key={index}
              className="border-border/70 bg-muted/20 rounded-xl border p-4"
            >
              <div className="grid items-start gap-3 md:grid-cols-2 xl:grid-cols-[1.4fr_1.2fr_0.9fr_auto_auto]">
                <div className="space-y-1">
                  <Label>Header</Label>
                  <Input
                    value={header.headerName}
                    onChange={(e) => updateHeader(index, 'headerName', e.target.value)}
                    placeholder="X-Student-ID"
                  />
                  <p className="text-muted-foreground min-h-4 text-xs font-mono">
                    {header.expressionKey
                      ? `${t('form.expressionKey', { defaultValue: 'Clave' })}: ${header.expressionKey}`
                      : '\u00A0'}
                  </p>
                </div>
                <div className="space-y-1">
                  <Label>{t('form.headerType', { defaultValue: 'Tipo' })}</Label>
                  <Select
                    value={header.type}
                    onValueChange={(v) => updateHeader(index, 'type', v as InputValueType)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="string">string</SelectItem>
                      <SelectItem value="number">number</SelectItem>
                      <SelectItem value="boolean">boolean</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="pt-6">
                  <div className="flex h-10 items-center gap-2">
                    <Switch
                      checked={header.required}
                      onCheckedChange={(checked) => updateHeader(index, 'required', checked)}
                      aria-label="Header requerido"
                    />
                    <span className="text-sm">
                      {t('form.headerRequired', { defaultValue: 'Requerido' })}
                    </span>
                  </div>
                </div>
                <div className="pt-6">
                  <div className="flex h-10 items-center justify-end">
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => removeHeader(index)}
                      aria-label={`Eliminar header ${header.headerName || index + 1}`}
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              </div>
              <div className="mt-3 space-y-1">
                <Label>{t('fields.description')}</Label>
                <Input
                  value={header.description ?? ''}
                  onChange={(e) => updateHeader(index, 'description', e.target.value)}
                  placeholder="Usado para identificar al alumno en el request."
                />
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="space-y-2 border-t pt-5">
        <Label htmlFor="request-body-example">
          {t('form.requestBodyLabel', { defaultValue: 'Ejemplo valido del request body' })}
        </Label>
        <textarea
          id="request-body-example"
          value={draft.requestBodyExampleText}
          onChange={(e) => onChange((c) => ({ ...c, requestBodyExampleText: e.target.value }))}
          rows={10}
          className="flex w-full rounded-xl border border-input bg-background px-4 py-3 font-mono text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          placeholder={`{\n  "student_id": "std_123",\n  "course": {\n    "code": "math_01"\n  }\n}`}
        />
        <p className="text-muted-foreground text-xs">
          {t('form.requestBodyHelp', {
            defaultValue:
              'No escribes schema. Guardamos un ejemplo y extraemos el catalogo de fields para el builder.',
          })}
        </p>
      </div>
    </div>
  );
}

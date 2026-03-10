import { Plus, Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { ExperimentMetric } from '@/api/types';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

interface MetricEditorProps {
  metrics: ExperimentMetric[];
  onChange: (metrics: ExperimentMetric[]) => void;
  disabled?: boolean;
}

export function MetricEditor({ metrics, onChange, disabled }: MetricEditorProps) {
  const { t } = useTranslation('experiments');

  const addMetric = () => {
    onChange([...metrics, { key: '', name: '', description: '' }]);
  };

  const removeMetric = (index: number) => {
    onChange(metrics.filter((_, i) => i !== index));
  };

  const updateMetric = (index: number, field: keyof ExperimentMetric, value: string) => {
    const updated = metrics.map((m, i) => (i === index ? { ...m, [field]: value } : m));
    onChange(updated);
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <Label>{t('form.metrics')}</Label>
        <Button type="button" variant="outline" size="sm" onClick={addMetric} disabled={disabled}>
          <Plus className="mr-1 h-3 w-3" />
          {t('form.addMetric')}
        </Button>
      </div>
      {metrics.length === 0 && (
        <p className="text-muted-foreground text-sm">{t('form.noMetrics')}</p>
      )}
      {metrics.map((metric, index) => (
        <div key={index} className="flex items-end gap-2">
          <div className="flex-1 space-y-1">
            <Label className="text-xs">{t('form.metricKey')}</Label>
            <Input
              value={metric.key}
              onChange={(e) => updateMetric(index, 'key', e.target.value)}
              placeholder="signup"
              className="font-mono text-sm"
              disabled={disabled}
            />
          </div>
          <div className="flex-1 space-y-1">
            <Label className="text-xs">{t('form.metricName')}</Label>
            <Input
              value={metric.name}
              onChange={(e) => updateMetric(index, 'name', e.target.value)}
              disabled={disabled}
            />
          </div>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="text-destructive shrink-0"
            onClick={() => removeMetric(index)}
            disabled={disabled}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      ))}
    </div>
  );
}

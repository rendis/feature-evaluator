import { useNavigate } from '@tanstack/react-router';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import type { Experiment, ExperimentMetric, Variant } from '@/api/types';

import { MetricEditor } from '@/components/experiments/metric-editor';
import { VariantEditor } from '@/components/experiments/variant-editor';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { useSubmissionLoadingModal } from '@/hooks/use-global-loading';
import { useCreateExperiment, useUpdateExperiment } from '@/mutations/experiment-mutations';

interface ExperimentFormProps {
  experiment?: Experiment;
  featureKeys: string[];
}

type CreateExperimentMutation = ReturnType<typeof useCreateExperiment>;
type UpdateExperimentMutation = ReturnType<typeof useUpdateExperiment>;

interface SubmitExperimentParams {
  createExperiment: CreateExperimentMutation;
  updateExperiment: UpdateExperimentMutation;
  description: string;
  featureKey: string;
  lookupCacheEnabled: boolean;
  lookupCacheTTLSeconds: string;
  isEditing: boolean;
  metrics: ExperimentMetric[];
  name: string;
  navigate: ReturnType<typeof useNavigate>;
  t: ReturnType<typeof useTranslation<'experiments'>>['t'];
  variants: Variant[];
}

function submitExperiment({
  createExperiment,
  updateExperiment,
  description,
  featureKey,
  lookupCacheEnabled,
  lookupCacheTTLSeconds,
  isEditing,
  metrics,
  name,
  navigate,
  t,
  variants,
}: SubmitExperimentParams) {
  const callbacks = {
    onSuccess: () => {
      toast.success(t('form.success'));
      navigate({ to: '/experiments' });
    },
    onError: () => toast.error(t('form.error')),
  };

  if (isEditing) {
    updateExperiment.mutate(
      {
        name,
        description,
        lookupCacheEnabled,
        lookupCacheTTLSeconds: lookupCacheEnabled
          ? normalizePositiveInt(lookupCacheTTLSeconds, 300)
          : 0,
        variants,
        metrics,
      },
      callbacks,
    );
    return;
  }

  if (!featureKey) {
    return;
  }

  createExperiment.mutate(
    {
      featureKey,
      name,
      description,
      lookupCacheEnabled,
      lookupCacheTTLSeconds: lookupCacheEnabled
        ? normalizePositiveInt(lookupCacheTTLSeconds, 300)
        : 0,
      variants,
      metrics,
    },
    callbacks,
  );
}

function createSubmitHandler(params: SubmitExperimentParams) {
  return (event: React.FormEvent) => {
    event.preventDefault();
    if (!params.name.trim()) return;
    submitExperiment(params);
  };
}

function getFormAction(isEditing: boolean) {
  return isEditing ? 'update' : 'create';
}

function isDraftExperiment(experiment?: Experiment) {
  return !experiment || experiment.status === 'draft';
}

function ExperimentFormContent({
  featureKey,
  featureKeys,
  isDraft,
  isEditing,
  isPending,
  lookupCacheEnabled,
  lookupCacheTTLSeconds,
  name,
  navigate,
  onDescriptionChange,
  onFeatureKeyChange,
  onNameChange,
  onLookupCacheEnabledChange,
  onLookupCacheTTLChange,
  onSubmit,
  onMetricsChange,
  onVariantsChange,
  description,
  metrics,
  variants,
}: {
  featureKey: string;
  featureKeys: string[];
  isDraft: boolean;
  isEditing: boolean;
  isPending: boolean;
  lookupCacheEnabled: boolean;
  lookupCacheTTLSeconds: string;
  name: string;
  navigate: ReturnType<typeof useNavigate>;
  onDescriptionChange: (value: string) => void;
  onFeatureKeyChange: (value: string) => void;
  onMetricsChange: (value: ExperimentMetric[]) => void;
  onNameChange: (value: string) => void;
  onLookupCacheEnabledChange: (value: boolean) => void;
  onLookupCacheTTLChange: (value: string) => void;
  onSubmit: (event: React.FormEvent) => void;
  onVariantsChange: (value: Variant[]) => void;
  description: string;
  metrics: ExperimentMetric[];
  variants: Variant[];
}) {
  const { t } = useTranslation('experiments');

  return (
    <form onSubmit={onSubmit} className="space-y-6">
      {!isEditing ? (
        <FeatureKeySelect featureKeys={featureKeys} value={featureKey} onChange={onFeatureKeyChange} />
      ) : null}
      <FormFields
        name={name}
        onNameChange={onNameChange}
        description={description}
        onDescriptionChange={onDescriptionChange}
      />
      <CacheSettings
        enabled={lookupCacheEnabled}
        ttlSeconds={lookupCacheTTLSeconds}
        onEnabledChange={onLookupCacheEnabledChange}
        onTTLChange={onLookupCacheTTLChange}
      />
      <VariantEditor variants={variants} onChange={onVariantsChange} disabled={!isDraft} />
      <MetricEditor metrics={metrics} onChange={onMetricsChange} disabled={!isDraft} />
      <div className="flex gap-2">
        <Button type="submit" disabled={isPending}>
          {t(isEditing ? 'form.save' : 'form.create')}
        </Button>
        <Button type="button" variant="outline" onClick={() => navigate({ to: '/experiments' })}>
          {t('actions.cancel', { ns: 'common' })}
        </Button>
      </div>
    </form>
  );
}

export function ExperimentForm({ experiment, featureKeys }: ExperimentFormProps) {
  const { t } = useTranslation('experiments');
  const navigate = useNavigate();
  const isEditing = !!experiment;
  const createExperiment = useCreateExperiment();
  const updateExperiment = useUpdateExperiment(experiment?.id ?? '');

  const [name, setName] = useState(experiment?.name ?? '');
  const [description, setDescription] = useState(experiment?.description ?? '');
  const [featureKey, setFeatureKey] = useState(experiment?.featureKey ?? '');
  const [lookupCacheEnabled, setLookupCacheEnabled] = useState(
    experiment?.lookupCacheEnabled ?? (experiment?.lookupCacheTTLSeconds ?? 0) > 0,
  );
  const [lookupCacheTTLSeconds, setLookupCacheTTLSeconds] = useState(
    experiment?.lookupCacheTTLSeconds ? String(experiment.lookupCacheTTLSeconds) : '300',
  );
  const [variants, setVariants] = useState<Variant[]>(
    experiment?.variants ?? [
      { key: 'control', value: '', weight: 50 },
      { key: 'treatment', value: '', weight: 50 },
    ],
  );
  const [metrics, setMetrics] = useState<ExperimentMetric[]>(experiment?.metrics ?? []);

  const isPending = createExperiment.isPending || updateExperiment.isPending;
  const isDraft = isDraftExperiment(experiment);
  useSubmissionLoadingModal(isPending, getFormAction(isEditing));

  const handleSubmit = createSubmitHandler({
    createExperiment,
    updateExperiment,
    description,
    featureKey,
    lookupCacheEnabled,
    lookupCacheTTLSeconds,
    isEditing,
    metrics,
    name,
    navigate,
    t,
    variants,
  });

  return (
    <ExperimentFormContent
      featureKey={featureKey}
      featureKeys={featureKeys}
      isDraft={isDraft}
      isEditing={isEditing}
      isPending={isPending}
      lookupCacheEnabled={lookupCacheEnabled}
      lookupCacheTTLSeconds={lookupCacheTTLSeconds}
      name={name}
      navigate={navigate}
      onDescriptionChange={setDescription}
      onFeatureKeyChange={setFeatureKey}
      onMetricsChange={setMetrics}
      onNameChange={setName}
      onLookupCacheEnabledChange={setLookupCacheEnabled}
      onLookupCacheTTLChange={setLookupCacheTTLSeconds}
      onSubmit={handleSubmit}
      onVariantsChange={setVariants}
      description={description}
      metrics={metrics}
      variants={variants}
    />
  );
}

function FeatureKeySelect({
  featureKeys,
  value,
  onChange,
}: {
  featureKeys: string[];
  value: string;
  onChange: (v: string) => void;
}) {
  const { t } = useTranslation('experiments');
  return (
    <div className="space-y-2">
      <Label htmlFor="exp-feature">{t('form.featureKey')}</Label>
      <select
        id="exp-feature"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="border-input bg-background flex h-9 w-full rounded-md border px-3 py-1 pr-8 text-sm"
        required
      >
        <option value="">{t('form.selectFeature')}</option>
        {featureKeys.map((k) => (
          <option key={k} value={k}>{k}</option>
        ))}
      </select>
    </div>
  );
}

function FormFields({
  name,
  onNameChange,
  description,
  onDescriptionChange,
}: {
  name: string;
  onNameChange: (v: string) => void;
  description: string;
  onDescriptionChange: (v: string) => void;
}) {
  const { t } = useTranslation('experiments');
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="exp-name">{t('form.name')}</Label>
        <Input id="exp-name" value={name} onChange={(e) => onNameChange(e.target.value)} required />
      </div>
      <div className="space-y-2">
        <Label htmlFor="exp-desc">{t('form.description')}</Label>
        <Input id="exp-desc" value={description} onChange={(e) => onDescriptionChange(e.target.value)} />
      </div>
    </>
  );
}

function CacheSettings({
  enabled,
  ttlSeconds,
  onEnabledChange,
  onTTLChange,
}: {
  enabled: boolean;
  ttlSeconds: string;
  onEnabledChange: (value: boolean) => void;
  onTTLChange: (value: string) => void;
}) {
  const { t } = useTranslation('experiments');

  return (
    <div className="rounded-2xl border border-border/70 bg-muted/10 p-4">
      <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div className="flex items-center gap-3">
          <Switch
            checked={enabled}
            aria-label={t('cache.lookup.enabled')}
            onCheckedChange={onEnabledChange}
          />
          <div>
            <p className="text-sm font-medium">{t('cache.lookup.enabled')}</p>
            <p className="text-muted-foreground text-xs">{t('cache.lookup.help')}</p>
          </div>
        </div>
        <div className="flex w-full items-center gap-2 md:max-w-64">
          <Label htmlFor="experiment-cache-ttl" className="shrink-0 text-sm">
            {t('cache.lookup.ttl')}
          </Label>
          <Input
            id="experiment-cache-ttl"
            value={ttlSeconds}
            disabled={!enabled}
            onChange={(event) => onTTLChange(event.target.value)}
            placeholder="300"
          />
        </div>
      </div>
    </div>
  );
}

function normalizePositiveInt(value: string, fallback: number) {
  const parsed = Number.parseInt(value, 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

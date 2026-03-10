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
    updateExperiment.mutate({ name, description, variants, metrics }, callbacks);
    return;
  }

  if (!featureKey) {
    return;
  }

  createExperiment.mutate({ featureKey, name, description, variants, metrics }, callbacks);
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
  name,
  navigate,
  onDescriptionChange,
  onFeatureKeyChange,
  onNameChange,
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
  name: string;
  navigate: ReturnType<typeof useNavigate>;
  onDescriptionChange: (value: string) => void;
  onFeatureKeyChange: (value: string) => void;
  onMetricsChange: (value: ExperimentMetric[]) => void;
  onNameChange: (value: string) => void;
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
      name={name}
      navigate={navigate}
      onDescriptionChange={setDescription}
      onFeatureKeyChange={setFeatureKey}
      onMetricsChange={setMetrics}
      onNameChange={setName}
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

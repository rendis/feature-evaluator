import { useNavigate } from '@tanstack/react-router';
import {
  ArrowLeft,
  ArrowRight,
  Code,
  FileText,
  Save,
  Settings,
  ToggleLeft,
  Trash2,
} from 'lucide-react';
import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';


import {
  buildCreatePayload,
  buildUpdatePayload,
  createDraft,
  serializeSnapshot,
  summarizeDraft,
  validateDraft,
  type FeatureDraft,
} from './feature-builder-utils';
import { StepConfig } from './feature-step-config';
import { StepFeature } from './feature-step-feature';
import { StepInputs } from './feature-step-inputs';
import { StepValue } from './feature-step-value';

import type { Feature } from '@/api/types';

import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { PageHeader } from '@/components/shared/page-header';
import { Button } from '@/components/ui/button';
import { useSubmissionLoadingModal } from '@/hooks/use-global-loading';
import { useUnsavedChanges } from '@/hooks/use-unsaved-changes';
import { getVisibleErrorMessage } from '@/lib/display-error';
import { slugifyResourceKey } from '@/lib/resource-key';
import {
  useCreateFeature,
  useDeleteFeature,
  useUpdateFeature,
} from '@/mutations/feature-mutations';

type BuilderStep = 'feature' | 'value' | 'inputs' | 'config';

interface StepDef {
  id: BuilderStep;
  icon: typeof FileText;
  label: string;
}

interface FeatureBuilderProps {
  feature?: Feature;
}

export function FeatureBuilder({ feature }: FeatureBuilderProps) {
  const { t } = useTranslation('features');
  const navigate = useNavigate();
  const isEditing = !!feature;
  const createFeature = useCreateFeature();
  const updateFeature = useUpdateFeature();
  const deleteFeature = useDeleteFeature();

  const [draft, setDraft] = useState<FeatureDraft>(() => createDraft(feature));
  const [savedSnapshot, setSavedSnapshot] = useState(() => serializeSnapshot(createDraft(feature)));
  const [activeStep, setActiveStep] = useState<BuilderStep>('feature');
  const [deleteOpen, setDeleteOpen] = useState(false);

  const isPending = createFeature.isPending || updateFeature.isPending;
  const isDirty = serializeSnapshot(draft) !== savedSnapshot;

  const { handleBack, UnsavedDialog, markClean } = useUnsavedChanges({
    isDirty,
    backTo: isEditing ? '/features/$featureKey' : '/features',
    backParams: isEditing && feature ? { featureKey: feature.key } : undefined,
    blockNavigation: true,
  });

  useSubmissionLoadingModal(isPending, isEditing ? 'update' : 'create');

  const derivedKey = useMemo(() => slugifyResourceKey(draft.name), [draft.name]);

  const steps: StepDef[] = useMemo(
    () => [
      { id: 'feature', icon: FileText, label: t('builder.steps.feature', { defaultValue: 'Feature' }) },
      { id: 'value', icon: ToggleLeft, label: t('builder.steps.value', { defaultValue: 'Valor' }) },
      { id: 'inputs', icon: Code, label: t('builder.steps.inputs', { defaultValue: 'Inputs' }) },
      { id: 'config', icon: Settings, label: t('builder.steps.config', { defaultValue: 'Configuracion' }) },
    ],
    [t],
  );

  const currentStepIndex = steps.findIndex((s) => s.id === activeStep);
  const previousStep = currentStepIndex > 0 ? steps[currentStepIndex - 1] : null;
  const nextStep = currentStepIndex < steps.length - 1 ? steps[currentStepIndex + 1] : null;

  const save = () => {
    const validationError = validateDraft(draft, t);
    if (validationError) {
      toast.error(validationError);
      return;
    }

    if (isEditing && feature) {
      const payload = buildUpdatePayload(draft);
      updateFeature.mutate(
        { key: feature.key, data: payload },
        {
          onSuccess: () => {
            markClean();
            setSavedSnapshot(serializeSnapshot(draft));
            toast.success(t('form.success'));
            void navigate({
              to: '/features/$featureKey',
              params: { featureKey: feature.key },
            });
          },
          onError: (error) => toast.error(getVisibleErrorMessage(error, t('form.error'))),
        },
      );
      return;
    }

    const payload = buildCreatePayload(draft);
    createFeature.mutate(payload, {
      onSuccess: (created) => {
        markClean();
        setSavedSnapshot(serializeSnapshot(draft));
        toast.success(t('form.success'));
        void navigate({
          to: '/features/$featureKey',
          params: { featureKey: created.key },
        });
      },
      onError: (error) => toast.error(getVisibleErrorMessage(error, t('form.error'))),
    });
  };

  const handleDelete = () => {
    if (!feature) return;
    deleteFeature.mutate(feature.key, {
      onSuccess: () => {
        markClean();
        toast.success(t('delete.success'));
        void navigate({ to: '/features' });
      },
      onError: () => toast.error(t('delete.error')),
    });
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-6">
      <PageHeader
        title={isEditing ? t('form.editTitle') : t('form.createTitle')}
        description={isEditing ? feature.key : undefined}
        actions={
          <div className="flex flex-wrap gap-2">
            <Button type="button" variant="outline" onClick={handleBack}>
              <ArrowLeft className="mr-2 h-4 w-4" />
              {t('actions.back', { ns: 'common' })}
            </Button>
            {isEditing ? (
              <Button type="button" variant="outline" onClick={() => setDeleteOpen(true)}>
                <Trash2 className="mr-2 h-4 w-4" />
                {t('actions.delete', { ns: 'common' })}
              </Button>
            ) : null}
            <Button type="button" onClick={save} disabled={!isDirty || isPending}>
              <Save className="mr-2 h-4 w-4" />
              {t('actions.save', { ns: 'common' })}
            </Button>
          </div>
        }
      />

      <div className="fe-panel-strong flex min-h-0 flex-1 flex-col p-6">
        <div className="mb-6 flex flex-wrap items-center justify-between gap-4">
          <div className="flex flex-wrap items-center gap-3">
            <div className="fe-tablist flex flex-wrap">
              {steps.map((step) => (
                <button
                  key={step.id}
                  type="button"
                  onClick={() => setActiveStep(step.id)}
                  data-active={activeStep === step.id}
                  className="fe-tab-trigger"
                >
                  <step.icon className="h-4 w-4" />
                  {step.label}
                </button>
              ))}
            </div>

            <div className="fe-tablist flex flex-wrap">
              <button
                type="button"
                onClick={() => previousStep && setActiveStep(previousStep.id)}
                disabled={!previousStep}
                className="fe-tab-trigger"
              >
                <ArrowLeft className="h-4 w-4" />
                {t('builder.previous', { defaultValue: 'Anterior' })}
              </button>
              <button
                type="button"
                onClick={() => nextStep && setActiveStep(nextStep.id)}
                disabled={!nextStep}
                className="fe-tab-trigger"
              >
                {t('builder.next', { defaultValue: 'Siguiente' })}
                <ArrowRight className="h-4 w-4" />
              </button>
            </div>
          </div>
        </div>

        <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
          {activeStep === 'feature' ? (
            <StepFeature
              draft={draft}
              derivedKey={derivedKey}
              isEditing={isEditing}
              onChange={setDraft}
            />
          ) : null}
          {activeStep === 'value' ? (
            <StepValue draft={draft} isEditing={isEditing} onChange={setDraft} />
          ) : null}
          {activeStep === 'inputs' ? (
            <StepInputs draft={draft} onChange={setDraft} />
          ) : null}
          {activeStep === 'config' ? (
            <StepConfig draft={draft} onChange={setDraft} />
          ) : null}
        </div>

        <details className="mt-6 border-t pt-5">
          <summary className="cursor-pointer text-sm font-semibold">
            {t('builder.advancedTitle', { defaultValue: 'Detalles avanzados' })}
          </summary>
          <div className="mt-4 grid gap-4 lg:grid-cols-2">
            <div className="space-y-2">
              <p className="text-sm font-medium">
                {t('builder.humanSummary', { defaultValue: 'Resumen' })}
              </p>
              <p className="text-muted-foreground text-sm">{summarizeDraft(draft, t)}</p>
            </div>
            <div className="space-y-2">
              <p className="text-sm font-medium">
                {t('builder.generatedPayload', { defaultValue: 'Payload generado' })}
              </p>
              <pre className="bg-muted/25 max-w-full min-w-0 overflow-auto whitespace-pre-wrap break-all rounded-xl border p-3 text-xs">
                {JSON.stringify(
                  isEditing ? buildUpdatePayload(draft) : buildCreatePayload(draft),
                  null,
                  2,
                )}
              </pre>
            </div>
          </div>
        </details>
      </div>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t('delete.title')}
        description={t('delete.description', { key: feature?.key ?? '' })}
        variant="destructive"
        onConfirm={handleDelete}
        loading={deleteFeature.isPending}
      />

      <UnsavedDialog />
    </div>
  );
}

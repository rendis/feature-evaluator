import { useNavigate } from '@tanstack/react-router';
import {
  ArrowLeft,
  ArrowRight,
  FileText,
  GaugeCircle,
  Save,
  SquareFunction,
  Trash2,
} from 'lucide-react';
import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import {
  buildPayload,
  createDraft,
  serializeSnapshot,
  summarizeDraft,
  validateDraft,
  type RuleDraft,
} from './rule-builder-utils';
import { StepExpression } from './rule-step-expression';
import { StepRollout } from './rule-step-rollout';
import { StepRule } from './rule-step-rule';
import type { Feature, Rule } from '@/api/types';

import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { PageHeader } from '@/components/shared/page-header';
import { Button } from '@/components/ui/button';
import { useSubmissionLoadingModal } from '@/hooks/use-global-loading';
import { useUnsavedChanges } from '@/hooks/use-unsaved-changes';
import { getVisibleErrorMessage } from '@/lib/display-error';
import { useCreateRule, useDeleteRule, useUpdateRule } from '@/mutations/rule-mutations';

type BuilderStep = 'rule' | 'expression' | 'rollout';

interface StepDef {
  id: BuilderStep;
  icon: typeof FileText;
  label: string;
}

interface RuleBuilderProps {
  feature: Feature;
  rule?: Rule;
  nextPriority?: number;
}

export function RuleBuilder({ feature, rule, nextPriority = 1 }: RuleBuilderProps) {
  const { t } = useTranslation('rules');
  const navigate = useNavigate();
  const isEditing = !!rule;
  const createRule = useCreateRule(feature.key);
  const updateRule = useUpdateRule(feature.key);
  const deleteRule = useDeleteRule(feature.key);

  const [draft, setDraft] = useState<RuleDraft>(() => createDraft(rule, feature, nextPriority));
  const [savedSnapshot, setSavedSnapshot] = useState(() =>
    serializeSnapshot(createDraft(rule, feature, nextPriority)),
  );
  const [activeStep, setActiveStep] = useState<BuilderStep>('rule');
  const [deleteOpen, setDeleteOpen] = useState(false);

  const isPending = createRule.isPending || updateRule.isPending;
  const isDirty = serializeSnapshot(draft) !== savedSnapshot;

  const { handleBack, UnsavedDialog, markClean } = useUnsavedChanges({
    isDirty,
    backTo: '/features/$featureKey',
    backParams: { featureKey: feature.key },
    blockNavigation: true,
  });

  useSubmissionLoadingModal(isPending, isEditing ? 'update' : 'create');

  const steps: StepDef[] = useMemo(
    () => [
      { id: 'rule', icon: FileText, label: t('builder.steps.rule', { defaultValue: 'Regla' }) },
      { id: 'expression', icon: SquareFunction, label: t('builder.steps.expression', { defaultValue: 'Expresion' }) },
      { id: 'rollout', icon: GaugeCircle, label: t('builder.steps.rollout', { defaultValue: 'Despliegue' }) },
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

    const payload = buildPayload(draft, feature.valueType);
    const callbacks = {
      onSuccess: () => {
        markClean();
        setSavedSnapshot(serializeSnapshot(draft));
        toast.success(t('form.success'));
        void navigate({
          to: '/features/$featureKey',
          params: { featureKey: feature.key },
        });
      },
      onError: (error: Error) => toast.error(getVisibleErrorMessage(error, t('form.error'))),
    };

    if (isEditing && rule) {
      updateRule.mutate({ ruleId: rule.id, data: payload }, callbacks);
      return;
    }

    createRule.mutate(payload, callbacks);
  };

  const handleDelete = () => {
    if (!rule) return;
    deleteRule.mutate(rule.id, {
      onSuccess: () => {
        markClean();
        toast.success(t('delete.success'));
        void navigate({
          to: '/features/$featureKey',
          params: { featureKey: feature.key },
        });
      },
      onError: () => toast.error(t('delete.error')),
    });
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-6">
      <PageHeader
        title={isEditing ? t('editRule') : t('createRule')}
        description={isEditing ? `${feature.key} / ${rule.name}` : feature.key}
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
          {activeStep === 'rule' ? (
            <StepRule draft={draft} valueType={feature.valueType} onChange={setDraft} />
          ) : null}
          {activeStep === 'expression' ? (
            <StepExpression draft={draft} feature={feature} onChange={setDraft} />
          ) : null}
          {activeStep === 'rollout' ? (
            <StepRollout draft={draft} onChange={setDraft} />
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
                {JSON.stringify(buildPayload(draft, feature.valueType), null, 2)}
              </pre>
            </div>
          </div>
        </details>
      </div>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t('delete.title')}
        description={t('delete.description', { name: rule?.name ?? '' })}
        variant="destructive"
        onConfirm={handleDelete}
        loading={deleteRule.isPending}
      />

      <UnsavedDialog />
    </div>
  );
}

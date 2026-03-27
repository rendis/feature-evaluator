import { useNavigate } from '@tanstack/react-router';
import {
  ArrowLeft,
  ArrowRight,
  Check,
  Lock,
  Play,
  Save,
  Settings,
  Trash2,
  User,
} from 'lucide-react';
import { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import type { TestAuthProfileRequest } from '@/api/authprofiles';
import type { AuthProfile } from '@/api/types';

import {
  applyUseCase,
  buildPayload,
  createDraft,
  parseObjectJson,
  serializeSnapshot,
  shouldReplaceSecrets,
  slugifyName,
  summarizeDraft,
  validateDraft,
  type DraftState,
} from './auth-profile-builder-utils';
import { StepConfig } from './auth-profile-step-config';
import { StepProfile } from './auth-profile-step-profile';
import { StepTest } from './auth-profile-step-test';
import { StepValidation } from './auth-profile-step-validation';

import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { PageHeader } from '@/components/shared/page-header';
import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';
import { useSubmissionLoadingModal } from '@/hooks/use-global-loading';
import { useUnsavedChanges } from '@/hooks/use-unsaved-changes';
import { getVisibleErrorMessage } from '@/lib/display-error';
import {
  useCreateAuthProfile,
  useDeleteAuthProfile,
  useTestAuthProfile,
  useUpdateAuthProfile,
} from '@/mutations/auth-profile-mutations';

type BuilderStep = 'profile' | 'config' | 'validation' | 'test';

interface StepDef {
  id: BuilderStep;
  icon: typeof User;
  label: string;
  disabledFor?: string[];
}

interface AuthProfileBuilderProps {
  profile?: AuthProfile;
}

export function AuthProfileBuilder({ profile }: AuthProfileBuilderProps) {
  const { t } = useTranslation('settings');
  const navigate = useNavigate();
  const isEditing = !!profile;
  const createAuthProfile = useCreateAuthProfile();
  const updateAuthProfile = useUpdateAuthProfile();
  const deleteAuthProfile = useDeleteAuthProfile();
  const testAuthProfile = useTestAuthProfile();

  const [draft, setDraft] = useState<DraftState>(() => createDraft(profile));
  const [savedSnapshot, setSavedSnapshot] = useState(() => serializeSnapshot(createDraft(profile)));
  const [activeStep, setActiveStep] = useState<BuilderStep>('profile');
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [testResult, setTestResult] = useState<{
    ok: boolean;
    attempted: boolean;
    httpStatus?: number;
    cached?: boolean;
    details?: Record<string, unknown>;
  } | null>(null);

  const isPending = createAuthProfile.isPending || updateAuthProfile.isPending;
  const isDirty = serializeSnapshot(draft) !== savedSnapshot;

  const { handleBack, UnsavedDialog, markClean } = useUnsavedChanges({
    isDirty,
    backTo: '/settings/auth-profiles',
    blockNavigation: true,
  });

  useSubmissionLoadingModal(isPending, isEditing ? 'update' : 'create');

  const derivedKey = useMemo(() => slugifyName(draft.name), [draft.name]);

  const steps: StepDef[] = useMemo(
    () => [
      { id: 'profile', icon: User, label: t('authProfiles.steps.profile') },
      { id: 'config', icon: Settings, label: t('authProfiles.steps.config') },
      {
        id: 'validation',
        icon: Check,
        label: t('authProfiles.steps.validation'),
        disabledFor: ['api_key', 'oidc_standard'],
      },
      { id: 'test', icon: Play, label: t('authProfiles.steps.test') },
    ],
    [t],
  );

  const isStepDisabled = (step: StepDef) =>
    step.disabledFor?.includes(draft.type) ?? false;

  const availableSteps = steps.filter((s) => !isStepDisabled(s));
  const currentStepIndex = availableSteps.findIndex((s) => s.id === activeStep);
  const previousStep = currentStepIndex > 0 ? availableSteps[currentStepIndex - 1] : null;
  const nextStep =
    currentStepIndex < availableSteps.length - 1 ? availableSteps[currentStepIndex + 1] : null;

  const cleanSecretPayload = (
    raw: Record<string, string | undefined> | undefined,
  ): Record<string, string> | undefined => {
    if (!raw) return undefined;
    return Object.fromEntries(
      Object.entries(raw).filter((entry): entry is [string, string] => entry[1] !== undefined),
    );
  };

  const save = () => {
    const validationError = validateDraft(draft, t);
    if (validationError) {
      toast.error(validationError);
      return;
    }

    const payload = buildPayload(draft, derivedKey);

    if (isEditing && profile) {
      updateAuthProfile.mutate(
          {
            key: profile.key,
            data: {
              key: payload.key,
              name: payload.name,
              active: payload.active,
              type: payload.type,
              config: payload.config,
              cacheEnabled: payload.cacheEnabled,
              cacheTTLSeconds: payload.cacheTTLSeconds,
              secretPayload: cleanSecretPayload(payload.secretPayload),
              replaceSecret: shouldReplaceSecrets(draft),
            },
        },
        {
          onSuccess: (saved) => {
            markClean();
            setSavedSnapshot(serializeSnapshot(createDraft(saved)));
            toast.success(t('authProfiles.updateSuccess'));
            void navigate({
              to: '/settings/auth-profiles/$key',
              params: { key: saved.key },
            });
          },
          onError: (error) =>
            toast.error(getVisibleErrorMessage(error, t('authProfiles.updateError'))),
        },
      );
      return;
    }

    createAuthProfile.mutate({ ...payload, secretPayload: cleanSecretPayload(payload.secretPayload) }, {
      onSuccess: (saved) => {
        markClean();
        setSavedSnapshot(serializeSnapshot(createDraft(saved)));
        toast.success(t('authProfiles.createSuccess'));
        void navigate({
          to: '/settings/auth-profiles/$key',
          params: { key: saved.key },
        });
      },
      onError: (error) => toast.error(getVisibleErrorMessage(error, t('authProfiles.createError'))),
    });
  };

  const runTest = () => {
    const validationError = validateDraft(draft, t);
    if (validationError) {
      toast.error(validationError);
      return;
    }

    let headers: Record<string, string>;
    let query: Record<string, string>;
    let body: Record<string, unknown>;
    try {
      if (draft.type === 'oidc_standard') {
        headers = draft.test.bearerToken.trim()
          ? { Authorization: `Bearer ${draft.test.bearerToken.trim()}` }
          : {};
        query = {};
        body = {};
      } else {
        headers = parseObjectJson<string>(draft.test.headers);
        query = parseObjectJson<string>(draft.test.query);
        body = parseObjectJson<unknown>(draft.test.body);
      }
    } catch {
      toast.error(t('authProfiles.testInvalidRequest'));
      return;
    }

    const payload = buildPayload(draft, derivedKey);

    const testPayload: TestAuthProfileRequest = {
      name: draft.name,
      active: draft.active,
      type: draft.type,
      config: payload.config,
      cacheEnabled: payload.cacheEnabled,
      cacheTTLSeconds: payload.cacheTTLSeconds,
      secretPayload: cleanSecretPayload(payload.secretPayload),
      testRequest: { headers, query, body },
    };

    testAuthProfile.mutate(testPayload, {
      onSuccess: (result) => {
        setTestResult(result);
        if (!result.attempted) {
          toast.warning(t('authProfiles.testNotAttempted'));
          return;
        }
        if (result.ok) {
          toast.success(t('authProfiles.testOk'));
          return;
        }
        toast.error(t('authProfiles.testRejected'));
      },
      onError: (error) => {
        setTestResult(null);
        toast.error(getVisibleErrorMessage(error, t('authProfiles.testError')));
      },
    });
  };

  const handleDelete = () => {
    if (!profile) return;
    deleteAuthProfile.mutate(profile.key, {
      onSuccess: () => {
        markClean();
        toast.success(t('authProfiles.deleteSuccess'));
        void navigate({ to: '/settings/auth-profiles' });
      },
      onError: () => toast.error(t('authProfiles.deleteError')),
    });
  };

  const handleTypeChange = (type: DraftState['type']) => {
    setDraft((current) => applyUseCase(current, type, profile?.hasSecret ?? false));
    if (activeStep === 'validation' && (type === 'api_key' || type === 'oidc_standard')) {
      setActiveStep('config');
    }
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-6">
      <PageHeader
        title={isEditing ? t('authProfiles.editTitle') : t('authProfiles.createTitle')}
        description={t('authProfiles.description')}
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
              {steps.map((step) => {
                const disabled = isStepDisabled(step);
                const button = (
                  <button
                    key={step.id}
                    type="button"
                    onClick={() => !disabled && setActiveStep(step.id)}
                    disabled={disabled}
                    data-active={activeStep === step.id}
                    className="fe-tab-trigger"
                  >
                    {disabled ? <Lock className="h-4 w-4" /> : <step.icon className="h-4 w-4" />}
                    {step.label}
                  </button>
                );

                if (disabled) {
                  return (
                    <TooltipProvider key={step.id} delayDuration={120}>
                      <Tooltip>
                        <TooltipTrigger asChild>{button}</TooltipTrigger>
                        <TooltipContent>
                          {t('authProfiles.steps.validationDisabledTooltip')}
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  );
                }

                return button;
              })}
            </div>

            <div className="fe-tablist flex flex-wrap">
              <button
                type="button"
                onClick={() => previousStep && setActiveStep(previousStep.id)}
                disabled={!previousStep}
                className="fe-tab-trigger"
              >
                <ArrowLeft className="h-4 w-4" />
                {t('authProfiles.steps.previous')}
              </button>

              <button
                type="button"
                onClick={() => nextStep && setActiveStep(nextStep.id)}
                disabled={!nextStep}
                className="fe-tab-trigger"
              >
                {t('authProfiles.steps.next')}
                <ArrowRight className="h-4 w-4" />
              </button>
            </div>
          </div>
        </div>

        <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
          {activeStep === 'profile' ? (
            <StepProfile
              draft={draft}
              derivedKey={derivedKey}
              isEditing={isEditing}
              hasSecret={profile?.hasSecret ?? false}
              onChange={setDraft}
              onTypeChange={handleTypeChange}
            />
          ) : null}
          {activeStep === 'config' ? (
            <StepConfig draft={draft} onChange={setDraft} />
          ) : null}
          {activeStep === 'validation' ? (
            <StepValidation draft={draft} onChange={setDraft} />
          ) : null}
          {activeStep === 'test' ? (
            <StepTest
              draft={draft}
              onChange={setDraft}
              testResult={testResult}
              onRunTest={runTest}
              isTesting={testAuthProfile.isPending}
            />
          ) : null}
        </div>

        <details className="mt-6 border-t pt-5">
          <summary className="cursor-pointer text-sm font-semibold">
            {t('authProfiles.advancedTitle')}
          </summary>
          <div className="mt-4 grid gap-4 lg:grid-cols-2">
            <div className="space-y-2">
              <p className="text-sm font-medium">{t('authProfiles.humanSummary')}</p>
              <p className="text-muted-foreground text-sm">{summarizeDraft(draft, t)}</p>
            </div>
            <div className="space-y-2">
              <p className="text-sm font-medium">{t('authProfiles.generatedPayload')}</p>
              <pre className="bg-muted/25 max-w-full min-w-0 overflow-auto whitespace-pre-wrap break-all rounded-xl border p-3 text-xs">
                {JSON.stringify(buildPayload(draft, derivedKey), null, 2)}
              </pre>
            </div>
          </div>
        </details>
      </div>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t('authProfiles.deleteTitle')}
        description={t('authProfiles.deleteDescription', { key: profile?.key ?? '' })}
        variant="destructive"
        onConfirm={handleDelete}
        loading={deleteAuthProfile.isPending}
      />

      <UnsavedDialog />
    </div>
  );
}

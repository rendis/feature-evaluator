import { useSuspenseQuery } from '@tanstack/react-query';
import { Navigate } from '@tanstack/react-router';
import { AlertTriangle, Plus, Save, Trash2 } from 'lucide-react';
import {
  useEffect,
  useMemo,
  useState,
  type Dispatch,
  type KeyboardEvent,
  type ReactNode,
  type SetStateAction,
} from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import type { SecurityPolicy, UpdateSecurityPolicyRequest } from '@/api/types';

import { normalizeSecurityPolicy } from '@/api/security-policies';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { PageHeader } from '@/components/shared/page-header';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { usePermissions } from '@/hooks/use-permissions';
import { getVisibleErrorMessage } from '@/lib/display-error';
import { useUpdateSecurityPolicy } from '@/mutations/security-policy-mutations';
import { securityPolicyQueries } from '@/queries/security-policy-queries';

interface PolicySectionState {
  values: string[];
  input: string;
}

function sanitizeValues(values: string[]) {
  return Array.from(
    new Set(values.map((value) => value.trim()).filter((value) => value.length > 0)),
  );
}

function sameValues(left: string[], right: string[]) {
  if (left.length !== right.length) return false;
  return left.every((value, index) => value === right[index]);
}

function asArray(values: string[] | null | undefined) {
  return Array.isArray(values) ? values : [];
}

export function shouldWarnForCurrentOriginRemoval(params: {
  currentOrigin: string;
  currentEffective: string[];
  inherited: string[];
  nextManaged: string[];
}) {
  const currentOrigin = params.currentOrigin.trim();
  if (!currentOrigin) return false;
  if (params.inherited.includes(currentOrigin)) return false;

  const currentEffective = sanitizeValues(params.currentEffective);
  const nextEffective = sanitizeValues([...params.inherited, ...params.nextManaged]);

  return currentEffective.includes(currentOrigin) && !nextEffective.includes(currentOrigin);
}

function SectionCard({
  children,
  description,
  title,
}: {
  children: ReactNode;
  description: string;
  title: string;
}) {
  return (
    <section className="space-y-4 rounded-xl border bg-card p-6 shadow-sm">
      <div className="space-y-1">
        <h2 className="text-lg font-semibold">{title}</h2>
        <p className="text-muted-foreground text-sm">{description}</p>
      </div>
      {children}
    </section>
  );
}

function ValuePills({
  disabled = false,
  emptyLabel,
  values,
}: {
  disabled?: boolean;
  emptyLabel: string;
  values: string[];
}) {
  if (values.length === 0) {
    return <p className="text-muted-foreground text-sm">{emptyLabel}</p>;
  }

  return (
    <div className="flex flex-wrap gap-2">
      {values.map((value) =>
        disabled ? (
          <button
            key={value}
            type="button"
            disabled
            className="cursor-not-allowed rounded-full border px-3 py-1 text-xs opacity-70"
          >
            {value}
          </button>
        ) : (
          <Badge key={value} variant="outline" className="px-3 py-1 font-mono text-xs">
            {value}
          </Badge>
        ),
      )}
    </div>
  );
}

function ManagedValuesEditor({
  addLabel,
  emptyLabel,
  inputPlaceholder,
  onAdd,
  onChangeInput,
  onKeyDown,
  onRemove,
  removeLabel,
  state,
  title,
}: {
  addLabel: string;
  emptyLabel: string;
  inputPlaceholder: string;
  onAdd: () => void;
  onChangeInput: (value: string) => void;
  onKeyDown: (event: KeyboardEvent<HTMLInputElement>) => void;
  onRemove: (value: string) => void;
  removeLabel: (value: string) => string;
  state: PolicySectionState;
  title: string;
}) {
  return (
    <div className="space-y-3">
      <div className="space-y-2">
        <h3 className="text-sm font-medium">{title}</h3>
        <div className="flex flex-col gap-2 sm:flex-row">
          <Input
            value={state.input}
            onChange={(event) => onChangeInput(event.target.value)}
            onKeyDown={onKeyDown}
            placeholder={inputPlaceholder}
            className="font-mono"
          />
          <Button type="button" variant="outline" onClick={onAdd}>
            <Plus className="mr-2 h-4 w-4" />
            {addLabel}
          </Button>
        </div>
      </div>
      {state.values.length === 0 ? (
        <p className="text-muted-foreground text-sm">{emptyLabel}</p>
      ) : (
        <div className="flex flex-wrap gap-2">
          {state.values.map((value) => (
            <div
              key={value}
              className="bg-muted inline-flex items-center gap-2 rounded-full border px-3 py-1"
            >
              <span className="font-mono text-xs">{value}</span>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={() => onRemove(value)}
                aria-label={removeLabel(value)}
              >
                <Trash2 className="h-3 w-3" />
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function SecurityPolicySection({
  addLabel,
  description,
  effective,
  effectiveTitle,
  emptyEffectiveLabel,
  emptyInheritedLabel,
  emptyManagedLabel,
  inherited,
  inheritedDescription,
  inheritedTitle,
  inputPlaceholder,
  managed,
  managedTitle,
  onAdd,
  onChangeInput,
  onKeyDown,
  onRemove,
  removeLabel,
  title,
}: {
  addLabel: string;
  description: string;
  effective: string[];
  effectiveTitle: string;
  emptyEffectiveLabel: string;
  emptyInheritedLabel: string;
  emptyManagedLabel: string;
  inherited: string[];
  inheritedDescription: string;
  inheritedTitle: string;
  inputPlaceholder: string;
  managed: PolicySectionState;
  managedTitle: string;
  onAdd: () => void;
  onChangeInput: (value: string) => void;
  onKeyDown: (event: KeyboardEvent<HTMLInputElement>) => void;
  onRemove: (value: string) => void;
  removeLabel: (value: string) => string;
  title: string;
}) {
  return (
    <SectionCard title={title} description={description}>
      <ManagedValuesEditor
        addLabel={addLabel}
        emptyLabel={emptyManagedLabel}
        inputPlaceholder={inputPlaceholder}
        onAdd={onAdd}
        onChangeInput={onChangeInput}
        onKeyDown={onKeyDown}
        onRemove={onRemove}
        removeLabel={removeLabel}
        state={managed}
        title={managedTitle}
      />

      <div className="space-y-2">
        <h3 className="text-sm font-medium">{inheritedTitle}</h3>
        <p className="text-muted-foreground text-sm">{inheritedDescription}</p>
        <ValuePills disabled values={inherited} emptyLabel={emptyInheritedLabel} />
      </div>

      <div className="space-y-2">
        <h3 className="text-sm font-medium">{effectiveTitle}</h3>
        <ValuePills values={effective} emptyLabel={emptyEffectiveLabel} />
      </div>
    </SectionCard>
  );
}

function buildInitialState(policy: SecurityPolicy) {
  const normalizedPolicy = normalizeSecurityPolicy(policy);

  return {
    corsOrigins: {
      values: [...asArray(normalizedPolicy.corsOrigins.managed)],
      input: '',
    },
    externalApiAllowHosts: {
      values: [...asArray(normalizedPolicy.externalApiAllowHosts.managed)],
      input: '',
    },
  };
}

export function SecurityPolicyRoutePage() {
  const { can } = usePermissions();

  if (!can('security.manage')) {
    return <Navigate to="/auth/access-denied" />;
  }

  return <SecurityPolicyPage />;
}

export function SecurityPolicyPage() {
  const { t } = useTranslation('security-policies');
  const { data } = useSuspenseQuery(securityPolicyQueries.detail());
  const policy = normalizeSecurityPolicy(data);
  const updateSecurityPolicy = useUpdateSecurityPolicy();
  const [corsOrigins, setCorsOrigins] = useState<PolicySectionState>(
    () => buildInitialState(policy).corsOrigins,
  );
  const [externalApiAllowHosts, setExternalApiAllowHosts] = useState<PolicySectionState>(
    () => buildInitialState(policy).externalApiAllowHosts,
  );
  const [warningOpen, setWarningOpen] = useState(false);
  const [pendingPayload, setPendingPayload] = useState<UpdateSecurityPolicyRequest | null>(null);

  useEffect(() => {
    const nextState = buildInitialState(policy);
    setCorsOrigins(nextState.corsOrigins);
    setExternalApiAllowHosts(nextState.externalApiAllowHosts);
  }, [policy]);

  const payload = useMemo<UpdateSecurityPolicyRequest>(
    () => ({
      corsOrigins: sanitizeValues(corsOrigins.values),
      externalApiAllowHosts: sanitizeValues(externalApiAllowHosts.values),
    }),
    [corsOrigins.values, externalApiAllowHosts.values],
  );

  const currentOrigin = window.location.origin;
  const warnForCurrentOrigin = shouldWarnForCurrentOriginRemoval({
    currentOrigin,
    currentEffective: asArray(policy.corsOrigins.effective),
    inherited: asArray(policy.corsOrigins.inherited),
    nextManaged: payload.corsOrigins,
  });
  const isDirty =
    !sameValues(payload.corsOrigins, policy.corsOrigins.managed) ||
    !sameValues(payload.externalApiAllowHosts, policy.externalApiAllowHosts.managed);

  const submit = (nextPayload: UpdateSecurityPolicyRequest) => {
    updateSecurityPolicy.mutate(nextPayload, {
      onSuccess: () => {
        setPendingPayload(null);
        setWarningOpen(false);
        toast.success(t('messages.saved'));
      },
      onError: (error) => {
        toast.error(getVisibleErrorMessage(error, t('messages.saveError')));
      },
    });
  };

  const handleSave = () => {
    if (!isDirty || updateSecurityPolicy.isPending) {
      return;
    }

    if (warnForCurrentOrigin) {
      setPendingPayload(payload);
      setWarningOpen(true);
      return;
    }

    submit(payload);
  };

  const handleAdd = (
    state: PolicySectionState,
    setState: Dispatch<SetStateAction<PolicySectionState>>,
  ) => {
    const nextValue = state.input.trim();
    if (!nextValue) {
      return;
    }

    setState({
      values: sanitizeValues([...state.values, nextValue]),
      input: '',
    });
  };

  const handleRemove = (value: string, setState: Dispatch<SetStateAction<PolicySectionState>>) => {
    setState((current) => ({
      ...current,
      values: current.values.filter((entry) => entry !== value),
    }));
  };

  const handleKeyDown =
    (state: PolicySectionState, setState: Dispatch<SetStateAction<PolicySectionState>>) =>
    (event: KeyboardEvent<HTMLInputElement>) => {
      if (event.key !== 'Enter') {
        return;
      }
      event.preventDefault();
      handleAdd(state, setState);
    };

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('title')}
        description={t('description')}
        actions={
          <Button onClick={handleSave} disabled={!isDirty || updateSecurityPolicy.isPending}>
            <Save className="mr-2 h-4 w-4" />
            {t('actions.save')}
          </Button>
        }
      />

      <SecurityPolicySection
        title={t('cors.title')}
        description={t('cors.description')}
        managed={corsOrigins}
        managedTitle={t('managed.title')}
        addLabel={t('cors.add')}
        inputPlaceholder={t('cors.placeholder')}
        emptyManagedLabel={t('managed.empty')}
        inheritedTitle={t('inherited.title')}
        inheritedDescription={t('inherited.description')}
        inherited={asArray(policy.corsOrigins.inherited)}
        emptyInheritedLabel={t('inherited.empty')}
        effectiveTitle={t('effective.title')}
        effective={sanitizeValues([
          ...asArray(policy.corsOrigins.inherited),
          ...payload.corsOrigins,
        ])}
        emptyEffectiveLabel={t('effective.empty')}
        onAdd={() => handleAdd(corsOrigins, setCorsOrigins)}
        onChangeInput={(value) => setCorsOrigins((current) => ({ ...current, input: value }))}
        onKeyDown={handleKeyDown(corsOrigins, setCorsOrigins)}
        onRemove={(value) => handleRemove(value, setCorsOrigins)}
        removeLabel={(value) => t('managed.removeValue', { value })}
      />

      <SecurityPolicySection
        title={t('hosts.title')}
        description={t('hosts.description')}
        managed={externalApiAllowHosts}
        managedTitle={t('managed.title')}
        addLabel={t('hosts.add')}
        inputPlaceholder={t('hosts.placeholder')}
        emptyManagedLabel={t('managed.empty')}
        inheritedTitle={t('inherited.title')}
        inheritedDescription={t('inherited.description')}
        inherited={asArray(policy.externalApiAllowHosts.inherited)}
        emptyInheritedLabel={t('inherited.empty')}
        effectiveTitle={t('effective.title')}
        effective={sanitizeValues([
          ...asArray(policy.externalApiAllowHosts.inherited),
          ...payload.externalApiAllowHosts,
        ])}
        emptyEffectiveLabel={t('effective.empty')}
        onAdd={() => handleAdd(externalApiAllowHosts, setExternalApiAllowHosts)}
        onChangeInput={(value) =>
          setExternalApiAllowHosts((current) => ({ ...current, input: value }))
        }
        onKeyDown={handleKeyDown(externalApiAllowHosts, setExternalApiAllowHosts)}
        onRemove={(value) => handleRemove(value, setExternalApiAllowHosts)}
        removeLabel={(value) => t('managed.removeValue', { value })}
      />

      <section className="rounded-xl border bg-card p-6 shadow-sm">
        <div className="flex items-start gap-3">
          <AlertTriangle className="text-warning mt-0.5 h-4 w-4 shrink-0" />
          <div className="space-y-1">
            <p className="font-medium">{t('notes.title')}</p>
            <p className="text-muted-foreground text-sm">{t('notes.description')}</p>
            {policy.updatedAt ? (
              <p className="text-muted-foreground text-xs">
                {t('notes.updatedBy', {
                  updatedAt: policy.updatedAt,
                  updatedBy: policy.updatedBy || t('notes.system'),
                })}
              </p>
            ) : null}
          </div>
        </div>
      </section>

      <ConfirmDialog
        open={warningOpen}
        onOpenChange={setWarningOpen}
        title={t('warning.title')}
        description={t('warning.description', { origin: currentOrigin })}
        confirmLabel={t('warning.confirm')}
        variant="destructive"
        onConfirm={() => {
          if (!pendingPayload) return;
          submit(pendingPayload);
        }}
        loading={updateSecurityPolicy.isPending}
      />
    </div>
  );
}

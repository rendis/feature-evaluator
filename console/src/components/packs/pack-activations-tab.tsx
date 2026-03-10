import { useSuspenseQuery } from '@tanstack/react-query';
import { Zap } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { ActivateDialog } from './activate-dialog';

import type { Pack, PackActivation, TargetType } from '@/api/types';

import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { EmptyState } from '@/components/shared/empty-state';
import { PermissionButton } from '@/components/shared/permission-button';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useLocaleFormatters } from '@/hooks/use-locale-formatters';
import { usePermissions } from '@/hooks/use-permissions';
import { useDeactivatePack } from '@/mutations/pack-mutations';
import { packQueries } from '@/queries/pack-queries';

interface PackActivationsTabProps {
  pack: Pack;
}

function targetBadgeVariant(type: TargetType) {
  switch (type) {
    case 'tenant':
      return 'default' as const;
    case 'campus':
      return 'secondary' as const;
    case 'program':
      return 'outline' as const;
  }
}

function ActivationSummary({ activations }: { activations: PackActivation[] }) {
  const { t } = useTranslation('packs');
  const counts = {
    tenant: activations.filter((activation) => activation.targetType === 'tenant').length,
    campus: activations.filter((activation) => activation.targetType === 'campus').length,
    program: activations.filter((activation) => activation.targetType === 'program').length,
  };

  if (activations.length === 0) {
    return null;
  }

  return (
    <div className="text-muted-foreground flex gap-2 text-sm">
      {counts.tenant > 0 ? <span>{counts.tenant} {t('targetTypes.tenant', { count: counts.tenant })}</span> : null}
      {counts.campus > 0 ? <span>{counts.campus} {t('targetTypes.campus', { count: counts.campus })}</span> : null}
      {counts.program > 0 ? <span>{counts.program} {t('targetTypes.program', { count: counts.program })}</span> : null}
    </div>
  );
}

function PackActivationsHeader({
  activations,
  onActivate,
}: {
  activations: PackActivation[];
  onActivate: () => void;
}) {
  const { t } = useTranslation('packs');

  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-4">
        <h2 className="text-lg font-semibold">{t('tabs.activations')}</h2>
        <ActivationSummary activations={activations} />
      </div>
      <PermissionButton permission="features.write" size="sm" onClick={onActivate}>
        <Zap className="mr-1 h-3 w-3" />
        {t('activations.activate')}
      </PermissionButton>
    </div>
  );
}

function PackActivationsEmptyState({ onActivate }: { onActivate: () => void }) {
  const { t } = useTranslation('packs');

  return (
    <EmptyState
      icon={<Zap className="h-10 w-10" />}
      title={t('activations.empty.title')}
      description={t('activations.empty.description')}
      action={
        <PermissionButton permission="features.write" size="sm" onClick={onActivate}>
          {t('activations.activate')}
        </PermissionButton>
      }
    />
  );
}

function PackActivationsTable({
  activations,
  canWrite,
  onDeactivate,
}: {
  activations: PackActivation[];
  canWrite: boolean;
  onDeactivate: (activation: PackActivation) => void;
}) {
  const { t } = useTranslation('packs');
  const { formatDate, formatDateTime, formatRelativeTime } = useLocaleFormatters();

  return (
    <div className="rounded-md border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/50">
            <th className="px-4 py-3 text-left font-medium">{t('activations.targetType')}</th>
            <th className="px-4 py-3 text-left font-medium">{t('activations.targetId')}</th>
            <th className="hidden px-4 py-3 text-left font-medium md:table-cell">{t('activations.activatedBy')}</th>
            <th className="hidden px-4 py-3 text-left font-medium sm:table-cell">{t('activations.activated')}</th>
            <th className="hidden px-4 py-3 text-left font-medium lg:table-cell">{t('activations.expires')}</th>
            {canWrite ? <th className="px-4 py-3 text-right font-medium">{t('fields.actions')}</th> : null}
          </tr>
        </thead>
        <tbody>
          {activations.map((activation) => (
            <tr key={activation.id} className="border-b last:border-0">
              <td className="px-4 py-3">
                <Badge variant={targetBadgeVariant(activation.targetType)}>
                  {t(`targetTypes.${activation.targetType}`)}
                </Badge>
              </td>
              <td className="px-4 py-3 font-mono text-xs">{activation.targetId}</td>
              <td className="text-muted-foreground hidden px-4 py-3 md:table-cell">{activation.activatedBy}</td>
              <td className="text-muted-foreground hidden px-4 py-3 sm:table-cell">
                <span title={formatDateTime(activation.activatedAt)}>
                  {formatRelativeTime(activation.activatedAt)}
                </span>
              </td>
              <td className="text-muted-foreground hidden px-4 py-3 lg:table-cell">
                {activation.expiresAt ? formatDate(activation.expiresAt) : t('activations.never')}
              </td>
              {canWrite ? (
                <td className="px-4 py-3 text-right">
                  <Button variant="ghost" size="sm" className="text-destructive" onClick={() => onDeactivate(activation)}>
                    {t('activations.deactivate')}
                  </Button>
                </td>
              ) : null}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function PackActivationDialogs({
  activateOpen,
  deactivatePack,
  deactivateTarget,
  handleDeactivate,
  onActivateOpenChange,
  onDeactivateOpenChange,
  packKey,
}: {
  activateOpen: boolean;
  deactivatePack: ReturnType<typeof useDeactivatePack>;
  deactivateTarget: PackActivation | null;
  handleDeactivate: () => void;
  onActivateOpenChange: (open: boolean) => void;
  onDeactivateOpenChange: (open: boolean) => void;
  packKey: string;
}) {
  const { t } = useTranslation('packs');

  return (
    <>
      <ActivateDialog open={activateOpen} onOpenChange={onActivateOpenChange} packKey={packKey} />
      <ConfirmDialog
        open={!!deactivateTarget}
        onOpenChange={onDeactivateOpenChange}
        title={t('activations.deactivateConfirm.title')}
        description={t('activations.deactivateConfirm.description', {
          pack: packKey,
          type: deactivateTarget?.targetType,
          target: deactivateTarget?.targetId,
        })}
        variant="destructive"
        confirmLabel={t('activations.deactivate')}
        onConfirm={handleDeactivate}
        loading={deactivatePack.isPending}
      />
    </>
  );
}

export function PackActivationsTab({ pack }: PackActivationsTabProps) {
  const { t } = useTranslation('packs');
  const { can } = usePermissions();
  const canWrite = can('features.write');
  const { data: activations } = useSuspenseQuery(packQueries.activations(pack.key));
  const [activateOpen, setActivateOpen] = useState(false);
  const [deactivateTarget, setDeactivateTarget] = useState<PackActivation | null>(null);
  const deactivatePack = useDeactivatePack();

  const handleDeactivate = () => {
    if (!deactivateTarget) return;
    deactivatePack.mutate(
      { key: pack.key, data: { targetType: deactivateTarget.targetType, targetId: deactivateTarget.targetId } },
      {
        onSuccess: () => {
          setDeactivateTarget(null);
          toast.success(t('activations.deactivateSuccess'));
        },
        onError: () => toast.error(t('activations.deactivateError')),
      },
    );
  };

  return (
    <div className="space-y-4">
      <PackActivationsHeader activations={activations} onActivate={() => setActivateOpen(true)} />
      {activations.length === 0 ? (
        <PackActivationsEmptyState onActivate={() => setActivateOpen(true)} />
      ) : (
        <PackActivationsTable
          activations={activations}
          canWrite={canWrite}
          onDeactivate={setDeactivateTarget}
        />
      )}
      <PackActivationDialogs
        activateOpen={activateOpen}
        deactivatePack={deactivatePack}
        deactivateTarget={deactivateTarget}
        handleDeactivate={handleDeactivate}
        onActivateOpenChange={setActivateOpen}
        onDeactivateOpenChange={(open) => !open && setDeactivateTarget(null)}
        packKey={pack.key}
      />
    </div>
  );
}

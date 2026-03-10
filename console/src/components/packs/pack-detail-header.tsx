import { useNavigate } from '@tanstack/react-router';
import { ArrowLeft, Pencil, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { PackToggle } from './pack-toggle';

import type { Pack } from '@/api/types';

import { TierBadge } from '@/components/shared/tier-badge';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { PermissionButton } from '@/components/shared/permission-button';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useDeletePack } from '@/mutations/pack-mutations';

interface PackDetailHeaderProps {
  pack: Pack;
  onEdit?: () => void;
}

function PackTrialBadge({ trialUntil }: { trialUntil: string }) {
  const { t } = useTranslation('packs');
  const trialDate = new Date(trialUntil);
  const isActive = trialDate > new Date();

  if (isActive) {
    return (
      <Badge variant="outline" className="border-amber-300 bg-amber-50 text-amber-700 dark:border-amber-700 dark:bg-amber-950 dark:text-amber-300">
        {t('trialActive', { date: trialDate.toLocaleDateString() })}
      </Badge>
    );
  }

  return (
    <Badge variant="outline" className="text-muted-foreground">
      {t('trialExpired')}
    </Badge>
  );
}

function PackInheritsBadges({ pack }: { pack: Pack }) {
  const { t } = useTranslation('packs');

  if (!pack.inheritsFrom || pack.inheritsFrom.length === 0) {
    return null;
  }

  return (
    <div className="mt-2 flex flex-wrap items-center gap-1.5">
      <span className="text-muted-foreground text-xs">{t('inheritsFromLabel')}:</span>
      {pack.inheritsFrom.map((parent) => (
        <Badge key={parent.key} variant="outline" className="text-xs">
          {parent.name}
        </Badge>
      ))}
    </div>
  );
}

export function PackDetailHeader({ pack, onEdit }: PackDetailHeaderProps) {
  const { t } = useTranslation('packs');
  const navigate = useNavigate();
  const deletePack = useDeletePack();
  const [deleteOpen, setDeleteOpen] = useState(false);

  const handleDelete = () => {
    deletePack.mutate(pack.key, {
      onSuccess: () => {
        toast.success(t('delete.success'));
        void navigate({ to: '/settings/packs' });
      },
      onError: () => toast.error(t('delete.error')),
    });
  };

  const showResolvedCount =
    pack.resolvedFeatureCount !== undefined &&
    pack.resolvedFeatureCount !== pack.featureKeys.length;

  return (
    <>
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" onClick={() => navigate({ to: '/settings/packs' })}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold">{pack.name}</h1>
              <PackToggle packKey={pack.key} enabled={pack.enabled} />
              {pack.tier ? <TierBadge tier={pack.tier} /> : null}
            </div>
            <p className="text-muted-foreground font-mono text-sm">{pack.key}</p>
            {pack.description ? (
              <p className="text-muted-foreground mt-1 text-sm">{pack.description}</p>
            ) : null}
            <PackInheritsBadges pack={pack} />
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Badge variant="secondary">
            {pack.featureKeys.length} {t('featureCount')}
          </Badge>
          {showResolvedCount ? (
            <Badge variant="outline" className="text-xs">
              {t('resolvedFeatures', { count: pack.resolvedFeatureCount })}
            </Badge>
          ) : null}
          {pack.trialUntil ? <PackTrialBadge trialUntil={pack.trialUntil} /> : null}
          <PermissionButton
            permission="features.write"
            variant="outline"
            size="sm"
            onClick={onEdit}
          >
            <Pencil className="mr-1 h-3 w-3" />
            {t('edit')}
          </PermissionButton>
          <PermissionButton
            permission="features.write"
            variant="outline"
            size="sm"
            className="text-destructive"
            onClick={() => setDeleteOpen(true)}
          >
            <Trash2 className="h-3 w-3" />
          </PermissionButton>
        </div>
      </div>

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t('delete.title')}
        description={t('delete.description', { key: pack.key })}
        variant="destructive"
        onConfirm={handleDelete}
        loading={deletePack.isPending}
      />
    </>
  );
}

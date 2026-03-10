import { Pencil, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import type { Tier } from '@/api/types';

import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { TierBadge } from '@/components/shared/tier-badge';
import { Button } from '@/components/ui/button';
import { useDeleteTier } from '@/mutations/tier-mutations';

interface TierListProps {
  tiers: Tier[];
  onEdit: (tier: Tier) => void;
}

export function TierList({ tiers, onEdit }: TierListProps) {
  const { t } = useTranslation('tiers');
  const deleteTier = useDeleteTier();
  const [deleteTarget, setDeleteTarget] = useState<Tier | null>(null);

  const handleDelete = () => {
    if (!deleteTarget) return;
    deleteTier.mutate(deleteTarget.key, {
      onSuccess: () => {
        toast.success(t('delete.success'));
        setDeleteTarget(null);
      },
      onError: () => {
        toast.error(t('delete.error'));
      },
    });
  };

  const sorted = [...tiers].sort((a, b) => a.level - b.level);

  return (
    <>
      <div className="fe-empty-surface overflow-hidden rounded-lg border">
        <table className="w-full">
          <thead>
            <tr className="border-b">
              <th className="text-content-muted px-4 py-3 text-left text-sm font-medium">
                {t('fields.name')}
              </th>
              <th className="text-content-muted px-4 py-3 text-left text-sm font-medium">
                {t('fields.level')}
              </th>
              <th className="text-content-muted px-4 py-3 text-right text-sm font-medium">
                {t('fields.actions')}
              </th>
            </tr>
          </thead>
          <tbody>
            {sorted.map((tier) => (
              <tr key={tier.id} className="border-b last:border-b-0">
                <td className="px-4 py-3">
                  <TierBadge tier={tier} />
                </td>
                <td className="text-content-muted px-4 py-3 text-sm">{tier.level}</td>
                <td className="px-4 py-3 text-right">
                  <div className="flex items-center justify-end gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onEdit(tier)}
                    >
                      <Pencil className="h-4 w-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => setDeleteTarget(tier)}
                    >
                      <Trash2 className="text-destructive h-4 w-4" />
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={t('delete.title')}
        description={t('delete.confirm')}
        variant="destructive"
        onConfirm={handleDelete}
        loading={deleteTier.isPending}
      />
    </>
  );
}

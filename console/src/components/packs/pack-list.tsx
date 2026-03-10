import { Link } from '@tanstack/react-router';
import { MoreHorizontal, Pencil, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { PackCard } from './pack-card';
import { PackToggle } from './pack-toggle';

import type { Pack } from '@/api/types';

import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';
import { useLocaleFormatters } from '@/hooks/use-locale-formatters';
import { useMobile } from '@/hooks/use-mobile';
import { usePermissions } from '@/hooks/use-permissions';
import { useDeletePack } from '@/mutations/pack-mutations';

interface PackListProps {
  packs: Pack[];
}

function PackActions({
  onDelete,
  pack,
}: {
  onDelete: (pack: Pack) => void;
  pack: Pack;
}) {
  const { t } = useTranslation('packs');

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon">
          <MoreHorizontal className="h-4 w-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem asChild>
          <Link to="/settings/packs/$packKey" params={{ packKey: pack.key }}>
            <Pencil className="mr-2 h-4 w-4" />
            {t('edit')}
          </Link>
        </DropdownMenuItem>
        <DropdownMenuItem className="text-destructive" onClick={() => onDelete(pack)}>
          <Trash2 className="mr-2 h-4 w-4" />
          {t('deletePack')}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function PackDesktopRow({
  canWrite,
  onDelete,
  pack,
}: {
  canWrite: boolean;
  onDelete: (pack: Pack) => void;
  pack: Pack;
}) {
  const { formatDate } = useLocaleFormatters();
  return (
    <tr className="border-b last:border-0">
      <td className="px-4 py-3">
        <Link
          to="/settings/packs/$packKey"
          params={{ packKey: pack.key }}
          className="hover:text-primary font-medium transition-colors"
        >
          {pack.name}
        </Link>
      </td>
      <td className="text-muted-foreground hidden px-4 py-3 font-mono text-xs sm:table-cell">{pack.key}</td>
      <td className="px-4 py-3">
        <Badge variant="secondary">{pack.featureKeys.length}</Badge>
      </td>
      <td className="px-4 py-3 text-center">
        <PackToggle packKey={pack.key} enabled={pack.enabled} />
      </td>
      <td className="text-muted-foreground hidden px-4 py-3 lg:table-cell">
        {formatDate(pack.createdAt)}
      </td>
      {canWrite ? (
        <td className="px-4 py-3 text-right">
          <PackActions onDelete={onDelete} pack={pack} />
        </td>
      ) : null}
    </tr>
  );
}

function PackDeleteDialog({
  deletePack,
  deleteTarget,
  onConfirm,
  onOpenChange,
}: {
  deletePack: ReturnType<typeof useDeletePack>;
  deleteTarget: Pack | null;
  onConfirm: () => void;
  onOpenChange: (open: boolean) => void;
}) {
  const { t } = useTranslation('packs');

  return (
    <ConfirmDialog
      open={!!deleteTarget}
      onOpenChange={onOpenChange}
      title={t('delete.title')}
      description={t('delete.description', { key: deleteTarget?.key })}
      variant="destructive"
      onConfirm={onConfirm}
      loading={deletePack.isPending}
    />
  );
}

function PackDesktopTable({ packs }: { packs: Pack[] }) {
  const { t } = useTranslation('packs');
  const { can } = usePermissions();
  const canWrite = can('features.write');
  const deletePack = useDeletePack();
  const [deleteTarget, setDeleteTarget] = useState<Pack | null>(null);

  const handleDelete = () => {
    if (!deleteTarget) return;
    deletePack.mutate(deleteTarget.key, {
      onSuccess: () => {
        setDeleteTarget(null);
        toast.success(t('delete.success'));
      },
      onError: () => toast.error(t('delete.error')),
    });
  };

  return (
    <>
      <div className="rounded-md border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b bg-muted/50">
              <th className="px-4 py-3 text-left font-medium">{t('fields.name')}</th>
              <th className="hidden px-4 py-3 text-left font-medium sm:table-cell">{t('fields.key')}</th>
              <th className="px-4 py-3 text-left font-medium">{t('fields.features')}</th>
              <th className="px-4 py-3 text-center font-medium">{t('fields.enabled')}</th>
              <th className="hidden px-4 py-3 text-left font-medium lg:table-cell">{t('fields.created')}</th>
              {canWrite ? <th className="px-4 py-3 text-right font-medium">{t('fields.actions')}</th> : null}
            </tr>
          </thead>
          <tbody>
            {packs.map((pack) => (
              <PackDesktopRow key={pack.key} canWrite={canWrite} onDelete={setDeleteTarget} pack={pack} />
            ))}
          </tbody>
        </table>
      </div>
      <PackDeleteDialog
        deletePack={deletePack}
        deleteTarget={deleteTarget}
        onConfirm={handleDelete}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      />
    </>
  );
}

export function PackList({ packs }: PackListProps) {
  const isMobile = useMobile();

  if (isMobile) {
    return (
      <div className="space-y-3">
        {packs.map((pack) => (
          <PackCard key={pack.key} pack={pack} />
        ))}
      </div>
    );
  }

  return <PackDesktopTable packs={packs} />;
}

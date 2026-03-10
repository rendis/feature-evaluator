import { MoreHorizontal, RefreshCw, XCircle } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { ApiKeyCreatedDialog } from './api-key-created-dialog';

import type { ApiKey } from '@/api/types';

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
import { usePermissions } from '@/hooks/use-permissions';
import { useRevokeApiKey, useRotateApiKey } from '@/mutations/api-key-mutations';

interface ApiKeyTableProps {
  apiKeys: ApiKey[];
}

type ApiKeyStatus = 'active' | 'expired' | 'revoked' | 'expiring_soon';

function getStatus(key: ApiKey): ApiKeyStatus {
  if (key.revoked) return 'revoked';
  if (key.expiresAt) {
    const expires = new Date(key.expiresAt);
    if (expires < new Date()) return 'expired';
    const thirtyDays = 30 * 24 * 60 * 60 * 1000;
    if (expires.getTime() - Date.now() < thirtyDays) return 'expiring_soon';
  }
  return 'active';
}

function statusBadgeVariant(status: ApiKeyStatus) {
  switch (status) {
    case 'active':
      return 'success' as const;
    case 'expired':
      return 'destructive' as const;
    case 'revoked':
      return 'secondary' as const;
    case 'expiring_soon':
      return 'warning' as const;
  }
}

function ApiKeyRow({
  canManage,
  onRevoke,
  onRotate,
  t,
  value,
}: {
  canManage: boolean;
  onRevoke: (key: ApiKey) => void;
  onRotate: (key: ApiKey) => void;
  t: ReturnType<typeof useTranslation<'settings'>>['t'];
  value: ApiKey;
}) {
  const { formatDate, formatRelativeTime } = useLocaleFormatters();
  const status = getStatus(value);

  return (
    <tr className="border-b last:border-0">
      <td className="px-4 py-3">
        <p className="font-medium">{value.name}</p>
        <p className="text-muted-foreground text-xs">{value.prefix}...</p>
      </td>
      <td className="px-4 py-3">
        <Badge variant="default">{t('apiKeys.typeAdmin')}</Badge>
      </td>
      <td className="hidden px-4 py-3 md:table-cell">
        <div className="flex flex-wrap gap-1">
          {value.permissions.length > 0
            ? value.permissions.map((permission) => (
                <Badge key={permission} variant="secondary" className="text-xs">
                  {t(`apiKeys.perm.${permission}`)}
                </Badge>
              ))
            : '-'}
        </div>
      </td>
      <td className="text-muted-foreground hidden px-4 py-3 lg:table-cell">
        {formatDate(value.createdAt)}
      </td>
      <td className="text-muted-foreground hidden px-4 py-3 lg:table-cell">
        {value.lastUsedAt ? formatRelativeTime(value.lastUsedAt) : '-'}
      </td>
      <td className="text-muted-foreground hidden px-4 py-3 sm:table-cell">
        {value.expiresAt ? formatDate(value.expiresAt) : '-'}
      </td>
      <td className="px-4 py-3">
        <Badge variant={statusBadgeVariant(status)}>{t(`apiKeys.status.${status}`)}</Badge>
      </td>
      {canManage ? (
        <td className="px-4 py-3 text-right">
          {!value.revoked ? (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon">
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => onRotate(value)}>
                  <RefreshCw className="mr-2 h-4 w-4" />
                  {t('apiKeys.rotate')}
                </DropdownMenuItem>
                <DropdownMenuItem className="text-destructive" onClick={() => onRevoke(value)}>
                  <XCircle className="mr-2 h-4 w-4" />
                  {t('apiKeys.revoke')}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          ) : null}
        </td>
      ) : null}
    </tr>
  );
}

function ApiKeyTableDialogs({
  isRevoking,
  isRotating,
  onRevoke,
  onRotate,
  newRawKey,
  revokeTarget,
  rotateTarget,
  setNewRawKey,
  setRevokeTarget,
  setRotateTarget,
}: {
  isRevoking: boolean;
  isRotating: boolean;
  newRawKey: string | null;
  onRevoke: () => void;
  onRotate: () => void;
  revokeTarget: ApiKey | null;
  rotateTarget: ApiKey | null;
  setNewRawKey: (value: string | null) => void;
  setRevokeTarget: (value: ApiKey | null) => void;
  setRotateTarget: (value: ApiKey | null) => void;
}) {
  const { t } = useTranslation('settings');

  return (
    <>
      <ConfirmDialog
        open={!!revokeTarget}
        onOpenChange={(open) => !open && setRevokeTarget(null)}
        title={t('apiKeys.revokeConfirm.title')}
        description={t('apiKeys.revokeConfirm.description', { name: revokeTarget?.name })}
        variant="destructive"
        onConfirm={onRevoke}
        loading={isRevoking}
      />

      <ConfirmDialog
        open={!!rotateTarget}
        onOpenChange={(open) => !open && setRotateTarget(null)}
        title={t('apiKeys.rotateConfirm.title')}
        description={t('apiKeys.rotateConfirm.description', { name: rotateTarget?.name })}
        onConfirm={onRotate}
        loading={isRotating}
      />

      {newRawKey ? (
        <ApiKeyCreatedDialog
          open={!!newRawKey}
          rawKey={newRawKey}
          onDismiss={() => setNewRawKey(null)}
        />
      ) : null}
    </>
  );
}

function useApiKeyActions({
  revokeApiKey,
  revokeTarget,
  rotateApiKey,
  rotateTarget,
  setNewRawKey,
  setRevokeTarget,
  setRotateTarget,
  t,
}: {
  revokeApiKey: ReturnType<typeof useRevokeApiKey>;
  revokeTarget: ApiKey | null;
  rotateApiKey: ReturnType<typeof useRotateApiKey>;
  rotateTarget: ApiKey | null;
  setNewRawKey: (value: string | null) => void;
  setRevokeTarget: (value: ApiKey | null) => void;
  setRotateTarget: (value: ApiKey | null) => void;
  t: ReturnType<typeof useTranslation<'settings'>>['t'];
}) {
  const handleRevoke = () => {
    if (!revokeTarget) return;
    revokeApiKey.mutate(revokeTarget.id, {
      onSuccess: () => {
        setRevokeTarget(null);
        toast.success(t('apiKeys.revokeSuccess'));
      },
      onError: () => toast.error(t('apiKeys.revokeError')),
    });
  };

  const handleRotate = () => {
    if (!rotateTarget) return;
    rotateApiKey.mutate(rotateTarget.id, {
      onSuccess: (response) => {
        setRotateTarget(null);
        setNewRawKey(response.key);
      },
      onError: () => toast.error(t('apiKeys.rotateError')),
    });
  };

  return { handleRevoke, handleRotate };
}

export function ApiKeyTable({ apiKeys }: ApiKeyTableProps) {
  const { t } = useTranslation('settings');
  const { can } = usePermissions();
  const revokeApiKey = useRevokeApiKey();
  const rotateApiKey = useRotateApiKey();
  const canManage = can('members.manage');

  const [revokeTarget, setRevokeTarget] = useState<ApiKey | null>(null);
  const [rotateTarget, setRotateTarget] = useState<ApiKey | null>(null);
  const [newRawKey, setNewRawKey] = useState<string | null>(null);
  const { handleRevoke, handleRotate } = useApiKeyActions({
    revokeApiKey,
    revokeTarget,
    rotateApiKey,
    rotateTarget,
    setNewRawKey,
    setRevokeTarget,
    setRotateTarget,
    t,
  });

  return (
    <>
      <div className="rounded-md border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b bg-muted/50">
              <th className="px-4 py-3 text-left font-medium">{t('apiKeys.name')}</th>
              <th className="px-4 py-3 text-left font-medium">{t('apiKeys.type')}</th>
              <th className="hidden px-4 py-3 text-left font-medium md:table-cell">
                {t('apiKeys.permissions')}
              </th>
              <th className="hidden px-4 py-3 text-left font-medium lg:table-cell">
                {t('apiKeys.created')}
              </th>
              <th className="hidden px-4 py-3 text-left font-medium lg:table-cell">
                {t('apiKeys.lastUsed')}
              </th>
              <th className="hidden px-4 py-3 text-left font-medium sm:table-cell">
                {t('apiKeys.expires')}
              </th>
              <th className="px-4 py-3 text-left font-medium">{t('apiKeys.statusLabel')}</th>
              {canManage ? (
                <th className="px-4 py-3 text-right font-medium">{t('apiKeys.actions')}</th>
              ) : null}
            </tr>
          </thead>
          <tbody>
            {apiKeys.map((key) => (
              <ApiKeyRow
                key={key.id}
                canManage={canManage}
                onRevoke={setRevokeTarget}
                onRotate={setRotateTarget}
                t={t}
                value={key}
              />
            ))}
          </tbody>
        </table>
      </div>

      <ApiKeyTableDialogs
        isRevoking={revokeApiKey.isPending}
        isRotating={rotateApiKey.isPending}
        newRawKey={newRawKey}
        onRevoke={handleRevoke}
        onRotate={handleRotate}
        revokeTarget={revokeTarget}
        rotateTarget={rotateTarget}
        setNewRawKey={setNewRawKey}
        setRevokeTarget={setRevokeTarget}
        setRotateTarget={setRotateTarget}
      />
    </>
  );
}

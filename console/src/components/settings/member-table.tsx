import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { MemberDesktopTable } from './member-desktop-table';
import { MemberMobileCards } from './member-mobile-cards';

import type { Member, MemberRole } from '@/api/types';

import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { useCurrentUser } from '@/hooks/use-current-user';
import { usePermissions } from '@/hooks/use-permissions';
import { useRemoveMember, useUpdateMemberRole } from '@/mutations/member-mutations';

interface MemberTableProps {
  members: Member[];
}

export function MemberTable({ members }: MemberTableProps) {
  const { t } = useTranslation('settings');
  const currentUser = useCurrentUser();
  const { can } = usePermissions();
  const updateRole = useUpdateMemberRole();
  const removeMember = useRemoveMember();
  const [deleteTarget, setDeleteTarget] = useState<Member | null>(null);

  const canManage = can('members.manage');

  const handleRoleChange = (member: Member, role: MemberRole) => {
    updateRole.mutate(
      { id: member.id, role },
      { onError: () => toast.error(t('members.addForm.error')) },
    );
  };

  const handleDelete = () => {
    if (!deleteTarget) return;
    removeMember.mutate(deleteTarget.id, {
      onSuccess: () => setDeleteTarget(null),
      onError: () => toast.error(t('members.addForm.error')),
    });
  };

  const shared = { members, currentUser, canManage, handleRoleChange, setDeleteTarget };

  return (
    <>
      <MemberDesktopTable {...shared} />
      <MemberMobileCards {...shared} />
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={t('members.deleteConfirm.title')}
        description={t('members.deleteConfirm.description', { email: deleteTarget?.email })}
        variant="destructive"
        onConfirm={handleDelete}
        loading={removeMember.isPending}
      />
    </>
  );
}

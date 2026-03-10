import { MoreHorizontal, Shield, Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { RoleSelect } from './role-select';

import type { Member, MemberRole } from '@/api/types';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

interface MemberDesktopTableProps {
  members: Member[];
  currentUser: Member;
  canManage: boolean;
  handleRoleChange: (member: Member, role: MemberRole) => void;
  setDeleteTarget: (member: Member | null) => void;
}

function roleBadgeVariant(role: MemberRole) {
  if (role === 'owner') return 'default' as const;
  if (role === 'admin') return 'secondary' as const;
  return 'outline' as const;
}

export function MemberDesktopTable(props: MemberDesktopTableProps) {
  const { members, currentUser, canManage, handleRoleChange, setDeleteTarget } = props;
  const { t } = useTranslation('settings');

  return (
    <div className="hidden sm:block">
      <div className="rounded-md border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b bg-muted/50">
              <th className="px-4 py-3 text-left font-medium">{t('members.email')}</th>
              <th className="px-4 py-3 text-left font-medium">{t('members.role')}</th>
              <th className="px-4 py-3 text-left font-medium">{t('members.addedBy')}</th>
              {canManage ? (
                <th className="px-4 py-3 text-right font-medium">{t('members.actions')}</th>
              ) : null}
            </tr>
          </thead>
          <tbody>
            {members.map((m) => (
              <MemberRow
                key={m.id}
                member={m}
                isSelf={m.id === currentUser.id}
                canManage={canManage}
                onRoleChange={handleRoleChange}
                onDelete={setDeleteTarget}
              />
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

interface MemberRowProps {
  member: Member;
  isSelf: boolean;
  canManage: boolean;
  onRoleChange: (member: Member, role: MemberRole) => void;
  onDelete: (member: Member | null) => void;
}

function MemberRow({ member, isSelf, canManage, onRoleChange, onDelete }: MemberRowProps) {
  const { t } = useTranslation('settings');
  const isOwner = member.role === 'owner';

  return (
    <tr className="border-b last:border-0">
      <td className="px-4 py-3">
        <p className="font-medium">{member.displayName || member.email}</p>
        <p className="text-muted-foreground text-xs">{member.email}</p>
      </td>
      <td className="px-4 py-3">
        {canManage && !isOwner && !isSelf ? (
          <RoleSelect
            value={member.role}
            onValueChange={(role) => onRoleChange(member, role)}
            excludeOwner
          />
        ) : (
          <Badge variant={roleBadgeVariant(member.role)}>{t(`members.roles.${member.role}`)}</Badge>
        )}
      </td>
      <td className="text-muted-foreground px-4 py-3">{member.addedBy}</td>
      {canManage ? (
        <td className="px-4 py-3 text-right">
          {!isOwner && !isSelf ? (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon">
                  <MoreHorizontal className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem className="text-destructive" onClick={() => onDelete(member)}>
                  <Trash2 className="mr-2 h-4 w-4" />
                  {t('members.removeMember')}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          ) : isOwner ? (
            <Badge variant="default">
              <Shield className="mr-1 h-3 w-3" />
              {t('members.roles.owner')}
            </Badge>
          ) : null}
        </td>
      ) : null}
    </tr>
  );
}

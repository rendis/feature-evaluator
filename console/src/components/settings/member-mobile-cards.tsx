import { Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { RoleSelect } from './role-select';

import type { Member, MemberRole } from '@/api/types';

import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';

interface MemberMobileCardsProps {
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

export function MemberMobileCards(props: MemberMobileCardsProps) {
  const { members, currentUser, canManage, handleRoleChange, setDeleteTarget } = props;
  const { t } = useTranslation('settings');

  return (
    <div className="space-y-3 sm:hidden">
      {members.map((member) => {
        const isSelf = member.id === currentUser.id;
        const isOwner = member.role === 'owner';
        return (
          <div key={member.id} className="rounded-lg border p-4">
            <div className="flex items-start justify-between">
              <div>
                <p className="font-medium">{member.displayName || member.email}</p>
                <p className="text-muted-foreground text-xs">{member.email}</p>
              </div>
              <Badge variant={roleBadgeVariant(member.role)}>
                {t(`members.roles.${member.role}`)}
              </Badge>
            </div>
            {canManage && !isOwner && !isSelf ? (
              <div className="mt-3 flex gap-2">
                <RoleSelect
                  value={member.role}
                  onValueChange={(role) => handleRoleChange(member, role)}
                  excludeOwner
                />
                <Button
                  variant="outline"
                  size="sm"
                  className="text-destructive"
                  onClick={() => setDeleteTarget(member)}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}

import { useTranslation } from 'react-i18next';

import type { MemberRole } from '@/api/types';

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

interface RoleSelectProps {
  value: MemberRole;
  onValueChange: (value: MemberRole) => void;
  disabled?: boolean;
  excludeOwner?: boolean;
}

const ROLES: MemberRole[] = ['owner', 'admin', 'editor', 'viewer'];

export function RoleSelect({ value, onValueChange, disabled, excludeOwner }: RoleSelectProps) {
  const { t } = useTranslation('settings');
  const roles = excludeOwner ? ROLES.filter((r) => r !== 'owner') : ROLES;

  return (
    <Select value={value} onValueChange={(v) => onValueChange(v as MemberRole)} disabled={disabled}>
      <SelectTrigger className="w-[140px]">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {roles.map((role) => (
          <SelectItem key={role} value={role}>
            {t(`members.roles.${role}`)}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}

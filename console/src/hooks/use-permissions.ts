import { useCallback } from 'react';

import { useAuth } from './use-auth';

import type { Permission } from '@/auth/roles';

import { hasPermission } from '@/auth/roles';

/** Check permissions for the current user. */
export function usePermissions() {
  const { member } = useAuth();

  const can = useCallback(
    (permission: Permission) => {
      if (!member) return false;
      return hasPermission(member.role, permission);
    },
    [member],
  );

  return { can, role: member?.role ?? null };
}

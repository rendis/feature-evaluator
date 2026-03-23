import type { MemberRole } from '@/api/types';

type Permission =
  | 'features.read'
  | 'features.write'
  | 'segments.read'
  | 'segments.write'
  | 'members.read'
  | 'members.manage'
  | 'settings.manage'
  | 'security.manage'
  | 'audit.read'
  | 'experiments.read'
  | 'experiments.write'
  | 'workspace.delete'
  | 'ownership.transfer';

const rolePermissions: Record<MemberRole, Permission[]> = {
  owner: [
    'features.read',
    'features.write',
    'segments.read',
    'segments.write',
    'members.read',
    'members.manage',
    'settings.manage',
    'security.manage',
    'audit.read',
    'experiments.read',
    'experiments.write',
    'workspace.delete',
    'ownership.transfer',
  ],
  admin: [
    'features.read',
    'features.write',
    'segments.read',
    'segments.write',
    'members.read',
    'members.manage',
    'settings.manage',
    'audit.read',
    'experiments.read',
    'experiments.write',
  ],
  editor: [
    'features.read',
    'features.write',
    'segments.read',
    'segments.write',
    'members.read',
    'audit.read',
    'experiments.read',
    'experiments.write',
  ],
  viewer: ['features.read', 'segments.read', 'members.read', 'audit.read', 'experiments.read'],
};

export function hasPermission(role: MemberRole, permission: Permission): boolean {
  return rolePermissions[role].includes(permission);
}

export type { Permission };

import { useMutation, useQueryClient } from '@tanstack/react-query';

import type { CreateMemberRequest, MemberRole } from '@/api/types';

import { membersApi } from '@/api/members';
import { memberQueries } from '@/queries/member-queries';

export function useCreateMember() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateMemberRequest) => membersApi.create(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: memberQueries.all() }),
  });
}

export function useUpdateMemberRole() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, role }: { id: string; role: MemberRole }) =>
      membersApi.updateRole(id, { role }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: memberQueries.all() }),
  });
}

export function useRemoveMember() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => membersApi.remove(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: memberQueries.all() }),
  });
}

export function useTransferOwnership() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => membersApi.transferOwnership(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: memberQueries.all() }),
  });
}

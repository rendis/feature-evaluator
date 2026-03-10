import { api } from './client';

import type { CreateMemberRequest, Member, UpdateMemberRoleRequest } from './types';

const BASE = '/admin/members';

export const membersApi = {
  list: () => api.get<{ data: Member[] }>(BASE).then((r) => r.data),
  create: (data: CreateMemberRequest) => api.post<Member>(BASE, data),
  updateRole: (id: string, data: UpdateMemberRoleRequest) =>
    api.put<Member>(`${BASE}/${id}/role`, data),
  remove: (id: string) => api.delete<undefined>(`${BASE}/${id}`),
  transferOwnership: (id: string) => api.post<undefined>(`${BASE}/${id}/transfer-ownership`),
  me: () => api.get<Member>(`${BASE}/me`),
};

import { useMutation, useQueryClient } from '@tanstack/react-query';

import type {
  CreateAuthProfileRequest,
  TestAuthProfileRequest,
  UpdateAuthProfileRequest,
} from '@/api/authprofiles';

import { authProfilesApi } from '@/api/authprofiles';
import { authProfileQueries } from '@/queries/auth-profile-queries';

export function useCreateAuthProfile() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateAuthProfileRequest) => authProfilesApi.create(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: authProfileQueries.all() }),
  });
}

export function useUpdateAuthProfile() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ key, data }: { key: string; data: UpdateAuthProfileRequest }) =>
      authProfilesApi.update(key, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: authProfileQueries.all() }),
  });
}

export function useDeleteAuthProfile() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (key: string) => authProfilesApi.remove(key),
    onSuccess: () => qc.invalidateQueries({ queryKey: authProfileQueries.all() }),
  });
}

export function useTestAuthProfile() {
  return useMutation({
    mutationFn: (data: TestAuthProfileRequest) => authProfilesApi.test(data),
  });
}

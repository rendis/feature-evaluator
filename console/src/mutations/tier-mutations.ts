import { useMutation, useQueryClient } from '@tanstack/react-query';

import type { CreateTierRequest, UpdateTierRequest } from '@/api/tiers';

import { tiersApi } from '@/api/tiers';
import { tierQueries } from '@/queries/tier-queries';

export function useCreateTier() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateTierRequest) => tiersApi.create(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: tierQueries.all() }),
  });
}

export function useUpdateTier() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ key, data }: { key: string; data: UpdateTierRequest }) =>
      tiersApi.update(key, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: tierQueries.all() }),
  });
}

export function useDeleteTier() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (key: string) => tiersApi.remove(key),
    onSuccess: () => qc.invalidateQueries({ queryKey: tierQueries.all() }),
  });
}

export function useUploadTierIcon() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ name, file }: { name: string; file: File }) =>
      tiersApi.uploadIcon(name, file),
    onSuccess: () => qc.invalidateQueries({ queryKey: tierQueries.all() }),
  });
}

export function useDeleteTierIcon() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => tiersApi.deleteIcon(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: tierQueries.all() }),
  });
}

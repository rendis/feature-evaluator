import { useMutation, useQueryClient } from '@tanstack/react-query';

import type {
  ActivatePackRequest,
  CreatePackRequest,
  DeactivatePackRequest,
  Pack,
  UpdatePackRequest,
} from '@/api/types';

import { packsApi } from '@/api/packs';
import { packQueries } from '@/queries/pack-queries';

export function useCreatePack() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreatePackRequest) => packsApi.create(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: packQueries.all() }),
  });
}

export function useUpdatePack() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ key, data }: { key: string; data: UpdatePackRequest }) =>
      packsApi.update(key, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: packQueries.all() }),
  });
}

export function useDeletePack() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (key: string) => packsApi.remove(key),
    onSuccess: () => qc.invalidateQueries({ queryKey: packQueries.all() }),
  });
}

export function useTogglePack() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ key, enabled }: { key: string; enabled: boolean }) =>
      packsApi.toggle(key, enabled),
    onMutate: async ({ key, enabled }) => {
      await qc.cancelQueries({ queryKey: packQueries.all() });
      const prev = qc.getQueryData<Pack>(packQueries.detail(key).queryKey);
      if (prev) {
        qc.setQueryData(packQueries.detail(key).queryKey, { ...prev, enabled });
      }
      return { prev, key };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) {
        qc.setQueryData(packQueries.detail(ctx.key).queryKey, ctx.prev);
      }
    },
    onSettled: () => qc.invalidateQueries({ queryKey: packQueries.all() }),
  });
}

export function useActivatePack() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ key, data }: { key: string; data: ActivatePackRequest }) =>
      packsApi.activate(key, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: packQueries.all() }),
  });
}

export function useDeactivatePack() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ key, data }: { key: string; data: DeactivatePackRequest }) =>
      packsApi.deactivate(key, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: packQueries.all() }),
  });
}

import { useMutation, useQueryClient } from '@tanstack/react-query';

import type { CreateTagRequest, UpdateTagRequest } from '@/api/tags';

import { tagsApi } from '@/api/tags';
import { tagQueries } from '@/queries/tag-queries';

export function useCreateTag() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateTagRequest) => tagsApi.create(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: tagQueries.all() }),
  });
}

export function useUpdateTag() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ key, data }: { key: string; data: UpdateTagRequest }) =>
      tagsApi.update(key, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: tagQueries.all() }),
  });
}

export function useDeleteTag() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (key: string) => tagsApi.remove(key),
    onSuccess: () => qc.invalidateQueries({ queryKey: tagQueries.all() }),
  });
}

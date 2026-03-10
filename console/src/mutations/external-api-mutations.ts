import { useMutation, useQueryClient } from '@tanstack/react-query';

import type { ExternalApiUpsertRequest, TestExternalApiRequest } from '@/api/external-apis';

import { externalApisApi } from '@/api/external-apis';
import { externalApiQueries } from '@/queries/external-api-queries';

export function useCreateExternalApi() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: ExternalApiUpsertRequest) => externalApisApi.create(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: externalApiQueries.all() }),
  });
}

export function useUpdateExternalApi() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ key, data }: { key: string; data: ExternalApiUpsertRequest }) =>
      externalApisApi.update(key, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: externalApiQueries.all() }),
  });
}

export function useDeleteExternalApi() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (key: string) => externalApisApi.remove(key),
    onSuccess: () => qc.invalidateQueries({ queryKey: externalApiQueries.all() }),
  });
}

export function useTestExternalApi() {
  return useMutation({
    mutationFn: (data: TestExternalApiRequest) => externalApisApi.test(data),
  });
}

import { useMutation, useQueryClient } from '@tanstack/react-query';

import type { CreateApiKeyRequest } from '@/api/types';

import { apiKeysApi } from '@/api/apikeys';
import { apiKeyQueries } from '@/queries/api-key-queries';

export function useCreateApiKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateApiKeyRequest) => apiKeysApi.create(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: apiKeyQueries.all() }),
  });
}

export function useRevokeApiKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiKeysApi.revoke(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: apiKeyQueries.all() }),
  });
}

export function useRotateApiKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiKeysApi.rotate(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: apiKeyQueries.all() }),
  });
}

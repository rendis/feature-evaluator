import { useMutation, useQueryClient } from '@tanstack/react-query';

import type { CreateExperimentRequest, UpdateExperimentRequest } from '@/api/types';

import { experimentsApi } from '@/api/experiments';
import { experimentQueries } from '@/queries/experiment-queries';

export function useCreateExperiment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateExperimentRequest) => experimentsApi.create(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: experimentQueries.all() }),
  });
}

export function useUpdateExperiment(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateExperimentRequest) => experimentsApi.update(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: experimentQueries.all() }),
  });
}

export function useStartExperiment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => experimentsApi.start(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: experimentQueries.all() }),
  });
}

export function usePauseExperiment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => experimentsApi.pause(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: experimentQueries.all() }),
  });
}

export function useCompleteExperiment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => experimentsApi.complete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: experimentQueries.all() }),
  });
}

export function useDeclareWinner() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, variantKey }: { id: string; variantKey: string }) =>
      experimentsApi.declareWinner(id, variantKey),
    onSuccess: () => qc.invalidateQueries({ queryKey: experimentQueries.all() }),
  });
}

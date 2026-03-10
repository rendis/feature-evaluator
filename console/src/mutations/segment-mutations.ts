import { useMutation, useQueryClient } from '@tanstack/react-query';

import type { CreateSegmentRequest, ImportSegmentDataRequest, UpdateSegmentRequest } from '@/api/segments';

import { segmentsApi } from '@/api/segments';
import { segmentQueries } from '@/queries/segment-queries';


export function useCreateSegment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateSegmentRequest) => segmentsApi.create(data),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: segmentQueries.all() });
    },
  });
}

export function useUpdateSegment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ key, data }: { key: string; data: UpdateSegmentRequest }) =>
      segmentsApi.update(key, data),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: segmentQueries.all() });
    },
  });
}

export function useDeleteSegment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (key: string) => segmentsApi.remove(key),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: segmentQueries.all() });
    },
  });
}

export function useImportSegmentData(segmentKey: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: ImportSegmentDataRequest) => segmentsApi.data.import(segmentKey, data),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: segmentQueries.all() });
    },
  });
}

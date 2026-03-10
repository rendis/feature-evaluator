import { useMutation, useQueryClient } from '@tanstack/react-query';

import type { CreateScheduleRequest } from '@/api/schedules';

import { schedulesApi } from '@/api/schedules';
import { scheduleQueries } from '@/queries/schedule-queries';

export function useCreateSchedule(featureKey: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateScheduleRequest) => schedulesApi.create(featureKey, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: scheduleQueries.all() }),
  });
}

export function useCancelSchedule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => schedulesApi.cancel(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: scheduleQueries.all() }),
  });
}

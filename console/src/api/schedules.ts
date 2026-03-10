import { api } from './client';

export type ScheduleStatus = 'pending' | 'executing' | 'completed' | 'failed' | 'cancelled';
export type ChangeType = 'toggle' | 'update' | 'default_value' | 'environment';

export interface ScheduledChange {
  id: string;
  workspaceKey: string;
  featureKey: string;
  changeType: ChangeType;
  payload: Record<string, unknown>;
  scheduledAt: string;
  status: ScheduleStatus;
  error?: string;
  executedAt?: string | null;
  createdBy: string;
  createdAt: string;
}

export interface CreateScheduleRequest {
  changeType: ChangeType;
  payload: Record<string, unknown>;
  scheduledAt: string;
}

const BASE = '/admin/features';

export const schedulesApi = {
  list: (featureKey: string) =>
    api
      .get<{ data: ScheduledChange[] }>(`${BASE}/${featureKey}/schedules`)
      .then((r) => r.data),
  create: (featureKey: string, data: CreateScheduleRequest) =>
    api.post<ScheduledChange>(`${BASE}/${featureKey}/schedules`, data),
  cancel: (id: string) => api.delete<undefined>(`/admin/schedules/${id}`),
};

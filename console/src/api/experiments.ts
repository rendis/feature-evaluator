import { api } from './client';

import type {
  CreateExperimentRequest,
  Experiment,
  ExperimentResults,
  UpdateExperimentRequest,
} from './types';

const BASE = '/admin/experiments';

export const experimentsApi = {
  list: () =>
    api.get<{ data: Experiment[] }>(BASE).then((r) => r.data),
  get: (id: string) => api.get<Experiment>(`${BASE}/${id}`),
  create: (data: CreateExperimentRequest) => api.post<Experiment>(BASE, data),
  update: (id: string, data: UpdateExperimentRequest) =>
    api.put<Experiment>(`${BASE}/${id}`, data),
  start: (id: string) => api.post<undefined>(`${BASE}/${id}/start`),
  pause: (id: string) => api.post<undefined>(`${BASE}/${id}/pause`),
  complete: (id: string) => api.post<undefined>(`${BASE}/${id}/complete`),
  declareWinner: (id: string, variantKey: string) =>
    api.post<undefined>(`${BASE}/${id}/declare-winner`, { variantKey }),
  results: (id: string) => api.get<ExperimentResults>(`${BASE}/${id}/results`),
};

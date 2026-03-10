import { api } from './client';

import type { ExternalApiBinding, Rule, SourceBindings } from './types';

export interface CreateRuleRequest {
  name: string;
  priority: number;
  enabled: boolean;
  expression: string;
  value: unknown;
  sourceBindings?: SourceBindings;
  externalApiBindings?: ExternalApiBinding[];
  metadata?: Record<string, unknown>;
  rolloutPercentage?: number | null;
}

export type UpdateRuleRequest = CreateRuleRequest;

const rulesBase = (featureKey: string) => `/admin/features/${featureKey}/rules`;

export const rulesApi = {
  list: (featureKey: string) =>
    api.get<{ data: Rule[] }>(rulesBase(featureKey)).then((r) => r.data),
  create: (featureKey: string, data: CreateRuleRequest) =>
    api.post<Rule>(rulesBase(featureKey), data),
  update: (featureKey: string, ruleId: string, data: UpdateRuleRequest) =>
    api.put<Rule>(`${rulesBase(featureKey)}/${ruleId}`, data),
  remove: (featureKey: string, ruleId: string) =>
    api.delete<undefined>(`${rulesBase(featureKey)}/${ruleId}`),
  reorder: (featureKey: string, ruleIds: string[]) =>
    api.put<undefined>(`${rulesBase(featureKey)}/reorder`, { ruleIds }),
};

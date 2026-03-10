import { useMutation, useQueryClient } from '@tanstack/react-query';

import type {
  CreateRuleRequest,
  UpdateRuleRequest,
} from '@/api/rules';
import type { Rule } from '@/api/types';

import { rulesApi } from '@/api/rules';
import { featureQueries } from '@/queries/feature-queries';
import { ruleQueries } from '@/queries/rule-queries';

export function useCreateRule(featureKey: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateRuleRequest) => rulesApi.create(featureKey, data),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ruleQueries.all(featureKey) });
      void qc.invalidateQueries({ queryKey: featureQueries.detail(featureKey).queryKey });
    },
  });
}

export function useUpdateRule(featureKey: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ ruleId, data }: { ruleId: string; data: UpdateRuleRequest }) =>
      rulesApi.update(featureKey, ruleId, data),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ruleQueries.all(featureKey) });
      void qc.invalidateQueries({ queryKey: featureQueries.detail(featureKey).queryKey });
    },
  });
}

export function useDeleteRule(featureKey: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (ruleId: string) => rulesApi.remove(featureKey, ruleId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ruleQueries.all(featureKey) });
      void qc.invalidateQueries({ queryKey: featureQueries.detail(featureKey).queryKey });
    },
  });
}

export function useReorderRules(featureKey: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (ruleIds: string[]) => rulesApi.reorder(featureKey, ruleIds),
    onMutate: async (ruleIds) => {
      await qc.cancelQueries({ queryKey: ruleQueries.all(featureKey) });
      const prev = qc.getQueryData<Rule[]>(ruleQueries.list(featureKey).queryKey);
      if (prev) {
        const sorted = ruleIds
          .map((id, i) => {
            const rule = prev.find((r) => r.id === id);
            return rule ? { ...rule, priority: i + 1 } : null;
          })
          .filter(Boolean) as Rule[];
        qc.setQueryData(ruleQueries.list(featureKey).queryKey, sorted);
      }
      return { prev };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) {
        qc.setQueryData(ruleQueries.list(featureKey).queryKey, ctx.prev);
      }
    },
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: ruleQueries.all(featureKey) });
    },
  });
}


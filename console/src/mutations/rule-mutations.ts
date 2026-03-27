import { useMutation, useQueryClient } from '@tanstack/react-query';

import type {
  CreateRuleRequest,
  UpdateRuleRequest,
} from '@/api/rules';
import type { Feature, Rule } from '@/api/types';

import { rulesApi } from '@/api/rules';
import { featureQueries } from '@/queries/feature-queries';
import { ruleQueries } from '@/queries/rule-queries';

function toUpdateRuleRequest(rule: Rule, enabled: boolean): UpdateRuleRequest {
  return {
    name: rule.name,
    priority: rule.priority,
    enabled,
    expression: rule.expression,
    value: rule.value,
    sourceBindings: rule.sourceBindings,
    externalApiBindings: rule.externalApiBindings,
    metadata: rule.metadata,
    rolloutPercentage: rule.rolloutPercentage ?? null,
  };
}

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

export function useToggleRule(featureKey: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ rule }: { rule: Rule }) =>
      rulesApi.update(featureKey, rule.id, toUpdateRuleRequest(rule, !rule.enabled)),
    onMutate: async ({ rule }) => {
      await Promise.all([
        qc.cancelQueries({ queryKey: ruleQueries.all(featureKey) }),
        qc.cancelQueries({ queryKey: featureQueries.detail(featureKey).queryKey }),
      ]);

      const previousRules = qc.getQueryData<Rule[]>(ruleQueries.list(featureKey).queryKey);
      const previousFeature = qc.getQueryData<Feature>(featureQueries.detail(featureKey).queryKey);

      if (previousRules) {
        qc.setQueryData<Rule[]>(
          ruleQueries.list(featureKey).queryKey,
          previousRules.map((current) =>
            current.id === rule.id ? { ...current, enabled: !current.enabled } : current,
          ),
        );
      }

      if (previousFeature?.rules) {
        qc.setQueryData<Feature>(featureQueries.detail(featureKey).queryKey, {
          ...previousFeature,
          rules: previousFeature.rules.map((current) =>
            current.id === rule.id ? { ...current, enabled: !current.enabled } : current,
          ),
        });
      }

      return { previousRules, previousFeature };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.previousRules) {
        qc.setQueryData(ruleQueries.list(featureKey).queryKey, ctx.previousRules);
      }
      if (ctx?.previousFeature) {
        qc.setQueryData(featureQueries.detail(featureKey).queryKey, ctx.previousFeature);
      }
    },
    onSettled: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: ruleQueries.all(featureKey) }),
        qc.invalidateQueries({
          queryKey: featureQueries.detail(featureKey).queryKey,
          exact: true,
        }),
      ]);
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

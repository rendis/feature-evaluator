import { useMutation, useQueryClient } from '@tanstack/react-query';

import type { CreateFeatureRequest, UpdateFeatureRequest } from '@/api/features';
import type { Feature, FeatureListItem, PaginatedResponse } from '@/api/types';

import { featuresApi } from '@/api/features';
import { featureQueries } from '@/queries/feature-queries';

function patchFeatureListCaches(
  qc: ReturnType<typeof useQueryClient>,
  updater: (item: FeatureListItem) => FeatureListItem | null,
) {
  qc.setQueriesData<PaginatedResponse<FeatureListItem>>(
    { queryKey: featureQueries.lists() },
    (current) => {
      if (!current) {
        return current;
      }

      return {
        ...current,
        data: current.data
          .map((item) => updater(item))
          .filter((item): item is FeatureListItem => item !== null),
      };
    },
  );
}

function mergeFeatureIntoListItem(item: FeatureListItem, feature: Feature): FeatureListItem {
  const shared = {
    ...item,
    id: feature.id,
    key: feature.key,
    name: feature.name,
    description: feature.description,
    enabled: feature.enabled,
    valueType: feature.valueType,
    tags: feature.tags,
    environments: feature.environments,
    accessPolicy: feature.accessPolicy,
    authProfileKey: feature.authProfileKey,
    ruleCount: feature.ruleCount ?? feature.rules?.length ?? 0,
    createdAt: feature.createdAt,
    updatedAt: feature.updatedAt,
    createdBy: feature.createdBy,
    updatedBy: feature.updatedBy,
  };

  if ('packCount' in item) {
    return shared;
  }

  return {
    ...shared,
    defaultValue: feature.defaultValue,
    metadata: feature.metadata,
    activeFrom: feature.activeFrom,
    activeUntil: feature.activeUntil,
    inputContract: feature.inputContract,
    packs: feature.packs,
    rules: feature.rules,
  } satisfies Feature;
}

export function useCreateFeature() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateFeatureRequest) => featuresApi.create(data),
    onSuccess: (feature) => {
      qc.setQueryData(featureQueries.detail(feature.key).queryKey, feature);
      return qc.invalidateQueries({ queryKey: featureQueries.lists() });
    },
  });
}

export function useUpdateFeature() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ key, data }: { key: string; data: UpdateFeatureRequest }) =>
      featuresApi.update(key, data),
    onSuccess: (feature) => {
      qc.setQueryData(featureQueries.detail(feature.key).queryKey, feature);
      patchFeatureListCaches(qc, (item) =>
        item.key === feature.key ? mergeFeatureIntoListItem(item, feature) : item,
      );
      return qc.invalidateQueries({ queryKey: featureQueries.lists() });
    },
  });
}

export function useDeleteFeature() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (key: string) => featuresApi.remove(key),
    onSuccess: (_data, key) => {
      qc.removeQueries({ queryKey: featureQueries.detail(key).queryKey, exact: true });
      patchFeatureListCaches(qc, (item) => (item.key === key ? null : item));
      return qc.invalidateQueries({ queryKey: featureQueries.lists() });
    },
  });
}

export function useToggleFeature() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ key, enabled }: { key: string; enabled: boolean }) =>
      featuresApi.toggle(key, enabled),
    onMutate: async ({ key, enabled }) => {
      await Promise.all([
        qc.cancelQueries({ queryKey: featureQueries.lists() }),
        qc.cancelQueries({ queryKey: featureQueries.detail(key).queryKey }),
      ]);

      const previousLists = qc.getQueriesData<PaginatedResponse<FeatureListItem>>({
        queryKey: featureQueries.lists(),
      });
      const previousDetail = qc.getQueryData<Feature>(featureQueries.detail(key).queryKey);

      if (previousDetail) {
        qc.setQueryData(featureQueries.detail(key).queryKey, { ...previousDetail, enabled });
      }
      patchFeatureListCaches(qc, (item) => (item.key === key ? { ...item, enabled } : item));

      return { key, previousDetail, previousLists };
    },
    onError: (_err, _vars, context) => {
      if (context?.previousDetail) {
        qc.setQueryData(featureQueries.detail(context.key).queryKey, context.previousDetail);
      }
      for (const [queryKey, data] of context?.previousLists ?? []) {
        qc.setQueryData(queryKey, data);
      }
    },
    onSettled: async (_data, _error, variables) => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: featureQueries.lists() }),
        qc.invalidateQueries({
          queryKey: featureQueries.detail(variables.key).queryKey,
          exact: true,
        }),
      ]);
    },
  });
}

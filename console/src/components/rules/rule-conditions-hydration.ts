import { isBuilderGroup } from './conditions-builder-types';

import type {
  BuilderCondition,
  BuilderExternalApiCondition,
  BuilderFieldCategory,
  BuilderFieldType,
  BuilderGroup,
  BuilderInputRef,
} from './conditions-builder-types';
import type { ExternalApi, ExternalApiBinding, ParamMapping } from '@/api/types';

export interface HydrationInputField {
  category: BuilderFieldCategory;
  label: string;
  path: string;
  type: BuilderFieldType;
}

interface HydrationContext {
  apisByKey: Map<string, ExternalApi>;
  bindingsByKey: Map<string, ExternalApiBinding>;
  inputFieldsByPath: Map<string, HydrationInputField>;
}

export function hydrateExternalApiConditions(
  root: BuilderGroup,
  {
    externalApiBindings,
    externalApis,
    inputFields,
  }: {
    externalApiBindings: ExternalApiBinding[];
    externalApis: ExternalApi[];
    inputFields: HydrationInputField[];
  },
): BuilderGroup {
  const context: HydrationContext = {
    apisByKey: new Map(externalApis.map((api) => [api.key, api])),
    bindingsByKey: new Map(externalApiBindings.map((binding) => [binding.externalApiKey, binding])),
    inputFieldsByPath: new Map(inputFields.map((field) => [field.path, field])),
  };

  const [nextRoot, changed] = hydrateGroup(root, context);
  return changed ? nextRoot : root;
}

function hydrateGroup(group: BuilderGroup, context: HydrationContext): [BuilderGroup, boolean] {
  let changed = false;

  const items = group.items.map((item) => {
    if (isBuilderGroup(item)) {
      const [nextGroup, groupChanged] = hydrateGroup(item, context);
      changed = changed || groupChanged;
      return nextGroup;
    }

    const nextCondition = hydrateCondition(item, context);
    changed = changed || nextCondition !== item;
    return nextCondition;
  });

  if (!changed) {
    return [group, false];
  }

  return [{ ...group, items }, true];
}

function hydrateCondition(
  condition: BuilderCondition,
  context: HydrationContext,
): BuilderCondition {
  if (condition.conditionKind !== 'externalApi') {
    return condition;
  }

  return hydrateExternalApiCondition(condition, context);
}

function hydrateExternalApiCondition(
  condition: BuilderExternalApiCondition,
  context: HydrationContext,
): BuilderExternalApiCondition {
  if (!condition.externalApiKey) {
    return condition;
  }

  const binding = context.bindingsByKey.get(condition.externalApiKey);
  const api = context.apisByKey.get(condition.externalApiKey);
  const nextName = api?.name || condition.externalApiName || condition.externalApiKey;
  const orderedParamNames = collectParamNames(condition, api, binding);

  if (orderedParamNames.length === 0) {
    return nextName === condition.externalApiName
      ? condition
      : { ...condition, externalApiName: nextName };
  }

  const existingByName = new Map(
    condition.paramMappings.map((mapping) => [mapping.paramName, mapping]),
  );
  const bindingByName = new Map(
    (binding?.paramMappings ?? []).map((mapping) => [mapping.paramName, mapping]),
  );
  const apiParamByName = new Map((api?.params ?? []).map((param) => [param.name, param]));

  const paramMappings = orderedParamNames.map((paramName) => {
    const existingMapping = existingByName.get(paramName);
    const bindingMapping = bindingByName.get(paramName);
    const apiParam = apiParamByName.get(paramName);
    const mode = existingMapping?.mode ?? bindingMapping?.mode ?? 'input';

    return {
      paramName,
      paramType: apiParam?.type ?? existingMapping?.paramType ?? 'any',
      required: apiParam?.required ?? existingMapping?.required ?? false,
      mode,
      inputRef: resolveInputRef(
        existingMapping?.inputRef ?? null,
        bindingMapping?.inputPath,
        context.inputFieldsByPath,
      ),
      literalValue: existingMapping?.literalValue ?? bindingMapping?.literalValue ?? '',
    };
  });

  if (
    nextName === condition.externalApiName &&
    JSON.stringify(paramMappings) === JSON.stringify(condition.paramMappings)
  ) {
    return condition;
  }

  return {
    ...condition,
    externalApiName: nextName,
    cacheEnabled: binding?.cacheEnabled ?? condition.cacheEnabled,
    cacheTTL: binding?.cacheTTL ?? condition.cacheTTL,
    paramMappings,
  };
}

function collectParamNames(
  condition: BuilderExternalApiCondition,
  api: ExternalApi | undefined,
  binding: ExternalApiBinding | undefined,
): string[] {
  const names: string[] = [];
  const seen = new Set<string>();

  for (const name of (api?.params ?? []).map((param) => param.name)) {
    if (!seen.has(name)) {
      seen.add(name);
      names.push(name);
    }
  }

  for (const name of condition.paramMappings.map((mapping) => mapping.paramName)) {
    if (!seen.has(name)) {
      seen.add(name);
      names.push(name);
    }
  }

  for (const name of (binding?.paramMappings ?? []).map((mapping) => mapping.paramName)) {
    if (!seen.has(name)) {
      seen.add(name);
      names.push(name);
    }
  }

  return names;
}

function resolveInputRef(
  existingInputRef: BuilderInputRef | null,
  inputPath: ParamMapping['inputPath'],
  inputFieldsByPath: Map<string, HydrationInputField>,
): BuilderInputRef | null {
  const path = existingInputRef?.path || inputPath;
  if (!path) {
    return existingInputRef;
  }

  const exactField = inputFieldsByPath.get(path);
  if (exactField) {
    return {
      refKind: 'input',
      category: exactField.category,
      path: exactField.path,
      label: exactField.label,
      type: exactField.type,
    };
  }

  if (existingInputRef?.type && existingInputRef.type !== 'unknown') {
    return existingInputRef;
  }

  return {
    refKind: 'input',
    category: inferCategory(path),
    path,
    label: path,
    type: 'unknown',
  };
}

function inferCategory(path: string): BuilderFieldCategory {
  if (path.startsWith('headers.')) {
    return 'headers';
  }
  if (path.startsWith('requestBody.')) {
    return 'requestBody';
  }
  return 'derived';
}

import { api } from './client';

import type {
  ExternalApiBinding,
  FeatureExpressionSchema,
  FeatureExpressionScenario,
  FeatureExpressionTestResponse,
  ExpressionSchema,
  ExpressionTestResponse,
  ExpressionValidateResponse,
  SourceBindings,
} from './types';

const BASE = '/admin/expression';

export const expressionApi = {
  validate: (expression: string) =>
    api.post<ExpressionValidateResponse>(`${BASE}/validate`, { expression }),
  test: (expression: string, context: Record<string, unknown>) =>
    api.post<ExpressionTestResponse>(`${BASE}/test`, { expression, context }),
  schema: () => api.get<ExpressionSchema>(`${BASE}/schema`),
  featureSchema: (featureKey: string) =>
    api.get<FeatureExpressionSchema>(`/admin/features/${featureKey}/expression-schema`),
  featureTest: (
    featureKey: string,
    payload: {
      expression: string;
      sourceBindings?: SourceBindings;
      externalApiBindings?: ExternalApiBinding[];
      scenario: FeatureExpressionScenario;
    },
  ) =>
    api.post<FeatureExpressionTestResponse>(
      `/admin/features/${featureKey}/expression/test`,
      payload,
    ),
};

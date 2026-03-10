import { api } from './client';

import type {
  ExternalApi,
  ExternalApiExpressionProfile,
  ExternalApiExpressionVariable,
  ExternalApiParam,
  ExternalApiRequestConfig,
  ExternalApiResponseValidation,
  ExternalApiTestResponse,
  ExpressionValidateResponse,
} from './types';

const BASE = '/admin/external-apis';

export interface ExternalApiUpsertRequest {
  key: string;
  name: string;
  active: boolean;
  request: ExternalApiRequestConfig;
  params: ExternalApiParam[];
  expressionVariables: ExternalApiExpressionVariable[];
  responseValidation: ExternalApiResponseValidation;
  secretPayload?: Record<string, string>;
  replaceSecret?: boolean;
}

export interface TestExternalApiRequest extends ExternalApiUpsertRequest {
  currentKey?: string;
  paramValues?: Record<string, unknown>;
}

export const externalApisApi = {
  list: () => api.get<{ data: ExternalApi[] }>(BASE).then((response) => response.data),
  expressionProfile: () =>
    api.get<ExternalApiExpressionProfile>(`${BASE}/expression-profile`),
  validateExpression: (expression: string) =>
    api.post<ExpressionValidateResponse>(`${BASE}/expression/validate`, { expression }),
  get: (key: string) => api.get<ExternalApi>(`${BASE}/${key}`),
  create: (data: ExternalApiUpsertRequest) => api.post<ExternalApi>(BASE, data),
  update: (key: string, data: ExternalApiUpsertRequest) =>
    api.put<ExternalApi>(`${BASE}/${key}`, data),
  remove: (key: string) => api.delete<undefined>(`${BASE}/${key}`),
  test: (data: TestExternalApiRequest) => api.post<ExternalApiTestResponse>(`${BASE}/test`, data),
};

import { api } from './client';

import type { SecurityPolicy, UpdateSecurityPolicyRequest } from './types';

const BASE = '/admin/system/security-policy';

export const securityPoliciesApi = {
  get: () => api.get<SecurityPolicy>(BASE),
  update: (data: UpdateSecurityPolicyRequest) => api.put<SecurityPolicy>(BASE, data),
};

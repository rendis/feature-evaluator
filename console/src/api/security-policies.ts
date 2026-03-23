import { api } from './client';

import type { SecurityPolicy, SecurityPolicyList, UpdateSecurityPolicyRequest } from './types';

const BASE = '/admin/system/security-policy';

function normalizeList(list: Partial<SecurityPolicyList> | null | undefined): SecurityPolicyList {
  return {
    managed: Array.isArray(list?.managed) ? list.managed : [],
    inherited: Array.isArray(list?.inherited) ? list.inherited : [],
    effective: Array.isArray(list?.effective) ? list.effective : [],
  };
}

export function normalizeSecurityPolicy(
  policy: Partial<SecurityPolicy> | null | undefined,
): SecurityPolicy {
  return {
    corsOrigins: normalizeList(policy?.corsOrigins),
    updatedAt: policy?.updatedAt,
    updatedBy: typeof policy?.updatedBy === 'string' ? policy.updatedBy : '',
  };
}

export const securityPoliciesApi = {
  get: () => api.get<SecurityPolicy>(BASE).then(normalizeSecurityPolicy),
  update: (data: UpdateSecurityPolicyRequest) =>
    api.put<SecurityPolicy>(BASE, data).then(normalizeSecurityPolicy),
};

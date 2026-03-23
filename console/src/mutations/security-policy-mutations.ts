import { useMutation, useQueryClient } from '@tanstack/react-query';

import type { UpdateSecurityPolicyRequest } from '@/api/types';

import { securityPoliciesApi } from '@/api/security-policies';
import { securityPolicyQueries } from '@/queries/security-policy-queries';

export function useUpdateSecurityPolicy() {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (data: UpdateSecurityPolicyRequest) => securityPoliciesApi.update(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: securityPolicyQueries.all() }),
  });
}

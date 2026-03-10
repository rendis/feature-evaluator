import { queryOptions } from '@tanstack/react-query';

import type { ListAuditErrorsParams } from '@/api/audit';

import { auditApi } from '@/api/audit';


export const auditQueries = {
  all: () => ['audit'] as const,
  errors: (params: ListAuditErrorsParams = {}) =>
    queryOptions({
      queryKey: [...auditQueries.all(), 'errors', params],
      queryFn: () => auditApi.errors(params),
    }),
};

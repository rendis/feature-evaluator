import { queryOptions } from '@tanstack/react-query';

import type { ListSegmentRecordsParams, ListSegmentsParams } from '@/api/segments';

import { segmentsApi } from '@/api/segments';


export const segmentQueries = {
  all: () => ['segments'] as const,
  list: (params: ListSegmentsParams = {}) =>
    queryOptions({
      queryKey: [...segmentQueries.all(), 'list', params],
      queryFn: () => segmentsApi.list(params),
    }),
  detail: (key: string) =>
    queryOptions({
      queryKey: [...segmentQueries.all(), 'detail', key],
      queryFn: () => segmentsApi.get(key),
      enabled: !!key,
    }),
  schema: (segmentKey: string) =>
    queryOptions({
      queryKey: [...segmentQueries.all(), 'detail', segmentKey, 'schema'],
      queryFn: () => segmentsApi.getSchema(segmentKey),
      enabled: !!segmentKey,
    }),
  records: (segmentKey: string, params: ListSegmentRecordsParams = {}) =>
    queryOptions({
      queryKey: [...segmentQueries.all(), 'detail', segmentKey, 'records', params],
      queryFn: () => segmentsApi.records.list(segmentKey, params),
      enabled: !!segmentKey,
    }),
};

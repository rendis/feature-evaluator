import { ChevronLeft, ChevronRight, Loader2 } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { PaginatedResponse, SegmentRecord } from '@/api/types';

import { formatPreviewValue, getValueAtPath } from './segment-data-utils';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Skeleton } from '@/components/ui/skeleton';
import { useLocaleFormatters } from '@/hooks/use-locale-formatters';

interface SegmentRecordTableProps {
  data?: PaginatedResponse<SegmentRecord>;
  isLoading: boolean;
  isFetching: boolean;
  previewFields: string[];
  query: string;
  onQueryChange: (query: string) => void;
  onPageChange: (page: number) => void;
}

export function SegmentRecordTable({
  data,
  isLoading,
  isFetching,
  previewFields,
  query,
  onQueryChange,
  onPageChange,
}: SegmentRecordTableProps) {
  const { t } = useTranslation('segments');
  const { formatDateTime } = useLocaleFormatters();
  const [selected, setSelected] = useState<SegmentRecord | null>(null);
  const columns = [t('data.recordKey'), ...previewFields, t('data.createdAt')];
  const rows = data?.data ?? [];
  const pagination = data?.pagination;

  return (
    <div className="min-w-0 space-y-4">
      <Input
        placeholder={t('data.filterKey')}
        value={query}
        onChange={(event) => onQueryChange(event.target.value)}
        className="max-w-sm"
      />

      <div className="relative">
        {isLoading && !data ? (
          <SegmentRecordTableSkeleton columnCount={columns.length} />
        ) : (
          <div className="max-w-full overflow-x-auto rounded-md border" aria-busy={isFetching}>
            <table className="w-max min-w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/50">
                  <th className="px-4 py-3 text-left font-medium whitespace-nowrap">{t('data.recordKey')}</th>
                  {previewFields.map((field) => (
                    <th key={field} className="px-4 py-3 text-left font-medium whitespace-nowrap">{field}</th>
                  ))}
                  <th className="px-4 py-3 text-left font-medium whitespace-nowrap">{t('data.createdAt')}</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((record) => (
                  <tr
                    key={record.id}
                    className="cursor-pointer border-b last:border-0 hover:bg-muted/20"
                    onClick={() => setSelected(record)}
                  >
                    <td className="px-4 py-3 font-mono text-xs whitespace-nowrap">{record.recordKey}</td>
                    {previewFields.map((field) => (
                      <td key={field} className="max-w-[220px] truncate px-4 py-3 text-xs">
                        {formatPreviewValue(getValueAtPath(record.attributes, field))}
                      </td>
                    ))}
                    <td className="text-muted-foreground px-4 py-3 text-xs whitespace-nowrap">
                      {formatDateTime(record.createdAt)}
                    </td>
                  </tr>
                ))}
                {rows.length === 0 ? (
                  <tr>
                    <td colSpan={columns.length} className="text-muted-foreground px-4 py-10 text-center text-sm">
                      {t('empty.description', { ns: 'common' })}
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        )}

        {isFetching && data ? (
          <SegmentRecordTableOverlay
            columnCount={columns.length}
            title={t('actions.loading', { ns: 'common' })}
          />
        ) : null}
      </div>

      {pagination && pagination.totalPages > 1 ? (
        <SegmentRecordPagination
          page={pagination.page}
          totalPages={pagination.totalPages}
          onPageChange={onPageChange}
        />
      ) : null}

      <Dialog open={!!selected} onOpenChange={(open) => !open && setSelected(null)}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>{selected?.recordKey}</DialogTitle>
          </DialogHeader>
          <pre className="max-h-[70vh] overflow-auto rounded-md border bg-muted/30 p-4 text-xs">
            {selected ? JSON.stringify(selected.attributes, null, 2) : ''}
          </pre>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function SegmentRecordPagination({
  page,
  totalPages,
  onPageChange,
}: {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center justify-center gap-2">
      <Button
        variant="outline"
        size="icon"
        className="h-9 w-9"
        disabled={page <= 1}
        onClick={() => onPageChange(page - 1)}
        aria-label={t('pagination.previous')}
        title={t('pagination.previous')}
      >
        <ChevronLeft className="h-4 w-4" />
      </Button>
      <div className="text-muted-foreground min-w-24 text-center text-sm tabular-nums">
        {page} / {totalPages}
      </div>
      <Button
        variant="outline"
        size="icon"
        className="h-9 w-9"
        disabled={page >= totalPages}
        onClick={() => onPageChange(page + 1)}
        aria-label={t('pagination.next')}
        title={t('pagination.next')}
      >
        <ChevronRight className="h-4 w-4" />
      </Button>
    </div>
  );
}

function SegmentRecordTableSkeleton({ columnCount }: { columnCount: number }) {
  return (
    <div className="overflow-hidden rounded-md border">
      <div className="border-b bg-muted/50 px-4 py-3">
        <Skeleton className="h-4 w-40" />
      </div>
      <div className="space-y-0">
        {Array.from({ length: 5 }).map((_, rowIndex) => (
          <div key={rowIndex} className="grid border-b last:border-b-0" style={{ gridTemplateColumns: `repeat(${columnCount}, minmax(120px, 1fr))` }}>
            {Array.from({ length: columnCount }).map((__, columnIndex) => (
              <div key={columnIndex} className="px-4 py-3">
                <Skeleton className="h-4 w-full max-w-[180px]" />
              </div>
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}

function SegmentRecordTableOverlay({
  columnCount,
  title,
}: {
  columnCount: number;
  title: string;
}) {
  return (
    <div className="absolute inset-0 z-10 overflow-hidden rounded-md border bg-background">
      <div className="flex items-center justify-end gap-2 border-b px-4 py-3 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span>{title}</span>
      </div>
      <div className="space-y-0">
        {Array.from({ length: 5 }).map((_, rowIndex) => (
          <div key={rowIndex} className="grid border-b last:border-b-0" style={{ gridTemplateColumns: `repeat(${columnCount}, minmax(120px, 1fr))` }}>
            {Array.from({ length: columnCount }).map((__, columnIndex) => (
              <div key={columnIndex} className="px-4 py-3">
                <Skeleton className="h-4 w-full max-w-[180px]" />
              </div>
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}

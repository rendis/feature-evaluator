import { keepPreviousData, useQuery, useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { ArrowLeft, Database, FileJson, Pencil, Trash2, Upload } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import type { ListSegmentRecordsParams } from '@/api/segments';

import { ImportWizard } from '@/components/segments/import-wizard/import-wizard';
import { SegmentForm } from '@/components/segments/segment-form';
import { SegmentRecordTable } from '@/components/segments/segment-record-table';
import { SegmentSchemaPanel } from '@/components/segments/segment-schema-panel';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { EmptyState } from '@/components/shared/empty-state';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { PermissionButton } from '@/components/shared/permission-button';
import { Button } from '@/components/ui/button';
import { useDeleteSegment } from '@/mutations/segment-mutations';
import { segmentQueries } from '@/queries/segment-queries';

export const Route = createFileRoute('/_authenticated/segments/$segmentKey/')({
  component: SegmentDetailPage,
  pendingComponent: () => <LoadingSkeleton rows={8} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

function SegmentImportView({
  onClose,
  segmentKey,
}: {
  onClose: () => void;
  segmentKey: string;
}) {
  const { t } = useTranslation('segments');

  return (
    <div className="min-w-0 space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={onClose}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h1 className="text-2xl font-bold">{t('import.title')}</h1>
      </div>
      <div className="mx-auto max-w-4xl">
        <ImportWizard segmentKey={segmentKey} onClose={onClose} />
      </div>
    </div>
  );
}

function SegmentDetailPage() {
  const { segmentKey } = Route.useParams();
  const { t } = useTranslation('segments');
  const navigate = useNavigate();
  const deleteSegment = useDeleteSegment();

  const [deleteOpen, setDeleteOpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<'schema' | 'data'>('schema');
  const [recordParams, setRecordParams] = useState<ListSegmentRecordsParams>({ page: 1, pageSize: 20 });

  const { data: segment } = useSuspenseQuery(segmentQueries.detail(segmentKey));
  const { data: schemaData } = useSuspenseQuery(segmentQueries.schema(segmentKey));
  const {
    data: recordsData,
    isLoading: isRecordsLoading,
    isFetching: isRecordsFetching,
  } = useQuery({
    ...segmentQueries.records(segmentKey, recordParams),
    enabled: segment.recordCount > 0 && activeTab === 'data',
    placeholderData: keepPreviousData,
  });

  const handleDelete = () => {
    deleteSegment.mutate(segment.key, {
      onSuccess: () => {
        toast.success(t('delete.success'));
        void navigate({ to: '/segments' });
      },
      onError: () => toast.error(t('delete.error')),
    });
  };

  if (importOpen) {
    return <SegmentImportView segmentKey={segmentKey} onClose={() => setImportOpen(false)} />;
  }

  return (
    <div className="min-w-0 space-y-8">
      <SegmentDetailHeader
        segment={segment}
        onEdit={() => setEditOpen(true)}
        onDelete={() => setDeleteOpen(true)}
        onImport={() => setImportOpen(true)}
      />

      <div className="flex gap-2 border-b">
        <TabButton
          active={activeTab === 'schema'}
          icon={<FileJson className="h-4 w-4" />}
          label={t('schema.title')}
          onClick={() => setActiveTab('schema')}
        />
        <TabButton
          active={activeTab === 'data'}
          icon={<Database className="h-4 w-4" />}
          label={t('data.title')}
          onClick={() => setActiveTab('data')}
        />
      </div>

      {segment.recordCount === 0 ? (
        <EmptyState title={t('data.empty')} description={t('empty.description')} />
      ) : null}

      {segment.recordCount > 0 && activeTab === 'schema' ? <SegmentSchemaPanel schemaData={schemaData} /> : null}

      {segment.recordCount > 0 && activeTab === 'data' ? (
        <SegmentRecordTable
          data={recordsData}
          isLoading={isRecordsLoading}
          isFetching={isRecordsFetching}
          previewFields={schemaData.previewFields}
          query={recordParams.q ?? ''}
          onQueryChange={(q) => setRecordParams((current) => ({ ...current, q, page: 1 }))}
          onPageChange={(page) => setRecordParams((current) => ({ ...current, page }))}
        />
      ) : null}

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t('delete.title')}
        description={t('delete.description', { key: segment.key })}
        variant="destructive"
        onConfirm={handleDelete}
        loading={deleteSegment.isPending}
      />

      <SegmentForm segment={segment} open={editOpen} onOpenChange={setEditOpen} />
    </div>
  );
}

function SegmentDetailHeader({
  segment,
  onEdit,
  onDelete,
  onImport,
}: {
  segment: { name: string; key: string; description: string; recordCount: number };
  onEdit: () => void;
  onDelete: () => void;
  onImport: () => void;
}) {
  const { t } = useTranslation('segments');
  const navigate = useNavigate();

  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={() => navigate({ to: '/segments' })}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-2xl font-bold">{segment.name}</h1>
          <p className="text-muted-foreground font-mono text-sm">{segment.key}</p>
          <p className="text-muted-foreground text-sm">{t('data.recordCountLabel', { count: segment.recordCount })}</p>
        </div>
      </div>
      <div className="flex items-center gap-2">
        <PermissionButton permission="segments.write" variant="outline" size="sm" onClick={onImport}>
          <Upload className="mr-1 h-3 w-3" />
          {t('import.title')}
        </PermissionButton>
        <PermissionButton permission="segments.write" variant="outline" size="sm" onClick={onEdit}>
          <Pencil className="mr-1 h-3 w-3" />
          {t('editSegment')}
        </PermissionButton>
        <PermissionButton
          permission="segments.write"
          variant="outline"
          size="sm"
          className="text-destructive"
          onClick={onDelete}
        >
          <Trash2 className="h-3 w-3" />
        </PermissionButton>
      </div>
    </div>
  );
}

function TabButton({
  active,
  icon,
  label,
  onClick,
}: {
  active: boolean;
  icon: React.ReactNode;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`inline-flex items-center gap-2 border-b-2 px-4 py-3 text-sm font-medium ${
        active ? 'border-primary text-foreground' : 'border-transparent text-muted-foreground'
      }`}
    >
      {icon}
      {label}
    </button>
  );
}

import { zodResolver } from '@hookform/resolvers/zod';
import { useMemo, useState } from 'react';
import { useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { z } from 'zod';

import type { Segment } from '@/api/types';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { useSubmissionLoadingModal } from '@/hooks/use-global-loading';
import { getVisibleErrorMessage } from '@/lib/display-error';
import { slugifyResourceKey } from '@/lib/resource-key';
import { useCreateSegment, useUpdateSegment } from '@/mutations/segment-mutations';

const segmentSchema = z.object({
  name: z.string().min(1).max(256),
  description: z.string().max(1024),
});

type FormValues = z.infer<typeof segmentSchema>;

interface SegmentFormProps {
  segment?: Segment;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: (key: string) => void;
}

function SegmentDerivedKey({ derivedKey }: { derivedKey: string }) {
  const { t } = useTranslation('segments');

  return (
    <div className="space-y-2">
      <Label>{t('fields.key')}</Label>
      <div className="text-muted-foreground border-border/70 bg-muted/35 flex h-10 items-center rounded-md border px-3 font-mono text-sm">
        {derivedKey || t('form.generatedKey')}
      </div>
      <p className="text-muted-foreground min-h-4 text-xs">{t('form.generatedKeyHelp')}</p>
    </div>
  );
}

export function SegmentForm({ segment, open, onOpenChange, onSuccess }: SegmentFormProps) {
  const { t } = useTranslation('segments');
  const createSegment = useCreateSegment();
  const updateSegment = useUpdateSegment();
  const isEditing = !!segment;
  const [membershipCacheEnabled, setMembershipCacheEnabled] = useState(
    segment?.membershipCacheEnabled ?? (segment?.membershipCacheTTLSeconds ?? 0) > 0,
  );
  const [membershipCacheTTLSeconds, setMembershipCacheTTLSeconds] = useState(
    segment?.membershipCacheTTLSeconds ? String(segment.membershipCacheTTLSeconds) : '300',
  );
  const [recordCacheEnabled, setRecordCacheEnabled] = useState(
    segment?.recordCacheEnabled ?? (segment?.recordCacheTTLSeconds ?? 0) > 0,
  );
  const [recordCacheTTLSeconds, setRecordCacheTTLSeconds] = useState(
    segment?.recordCacheTTLSeconds ? String(segment.recordCacheTTLSeconds) : '300',
  );

  const {
    register,
    handleSubmit,
    reset,
    watch,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(segmentSchema),
    defaultValues: {
      name: segment?.name ?? '',
      description: segment?.description ?? '',
    },
  });

  const name = watch('name');
  const derivedKey = useMemo(() => slugifyResourceKey(name), [name]);

  const onSubmit = (data: FormValues) => {
    const callbacks = {
      onSuccess: (key: string) => {
        toast.success(t('form.success'));
        reset();
        setMembershipCacheEnabled(segment?.membershipCacheEnabled ?? false);
        setMembershipCacheTTLSeconds(
          segment?.membershipCacheTTLSeconds ? String(segment.membershipCacheTTLSeconds) : '300',
        );
        setRecordCacheEnabled(segment?.recordCacheEnabled ?? false);
        setRecordCacheTTLSeconds(
          segment?.recordCacheTTLSeconds ? String(segment.recordCacheTTLSeconds) : '300',
        );
        onOpenChange(false);
        onSuccess?.(key);
      },
      onError: (error: unknown) => toast.error(getVisibleErrorMessage(error, t('form.error'))),
    };

    if (isEditing) {
      updateSegment.mutate(
        {
          key: segment.key,
          data: {
            name: data.name,
            description: data.description,
            membershipCacheEnabled,
            membershipCacheTTLSeconds: membershipCacheEnabled
              ? normalizePositiveInt(membershipCacheTTLSeconds, 300)
              : 0,
            recordCacheEnabled,
            recordCacheTTLSeconds: recordCacheEnabled
              ? normalizePositiveInt(recordCacheTTLSeconds, 300)
              : 0,
          },
        },
        { onSuccess: () => callbacks.onSuccess(segment.key), onError: callbacks.onError },
      );
    } else {
      createSegment.mutate(
        {
          key: derivedKey,
          name: data.name,
          description: data.description,
          membershipCacheEnabled,
          membershipCacheTTLSeconds: membershipCacheEnabled
            ? normalizePositiveInt(membershipCacheTTLSeconds, 300)
            : 0,
          recordCacheEnabled,
          recordCacheTTLSeconds: recordCacheEnabled
            ? normalizePositiveInt(recordCacheTTLSeconds, 300)
            : 0,
        },
        {
          onSuccess: (s) => callbacks.onSuccess(s.key),
          onError: callbacks.onError,
        },
      );
    }
  };

  const isPending = createSegment.isPending || updateSegment.isPending;
  useSubmissionLoadingModal(isPending, isEditing ? 'update' : 'create');

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isEditing ? t('form.editTitle') : t('form.createTitle')}</DialogTitle>
          <DialogDescription>
            {t('form.dialogDescription', { defaultValue: 'Define a segment to group users for targeted feature delivery.' })}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">{t('fields.name')}</Label>
            <Input id="name" {...register('name')} />
            {errors.name ? <p className="text-destructive text-sm">{t('form.nameRequired')}</p> : null}
          </div>
          {!isEditing ? <SegmentDerivedKey derivedKey={derivedKey} /> : null}
          <div className="space-y-2">
            <Label htmlFor="description">{t('fields.description')}</Label>
            <Input id="description" {...register('description')} />
          </div>
          <CacheSection
            membershipCacheEnabled={membershipCacheEnabled}
            membershipCacheTTLSeconds={membershipCacheTTLSeconds}
            onMembershipCacheEnabledChange={setMembershipCacheEnabled}
            onMembershipCacheTTLSecondsChange={setMembershipCacheTTLSeconds}
            recordCacheEnabled={recordCacheEnabled}
            recordCacheTTLSeconds={recordCacheTTLSeconds}
            onRecordCacheEnabledChange={setRecordCacheEnabled}
            onRecordCacheTTLSecondsChange={setRecordCacheTTLSeconds}
          />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t('actions.cancel', { ns: 'common' })}
            </Button>
            <Button type="submit" disabled={isPending}>{t('actions.save', { ns: 'common' })}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function CacheSection({
  membershipCacheEnabled,
  membershipCacheTTLSeconds,
  onMembershipCacheEnabledChange,
  onMembershipCacheTTLSecondsChange,
  recordCacheEnabled,
  recordCacheTTLSeconds,
  onRecordCacheEnabledChange,
  onRecordCacheTTLSecondsChange,
}: {
  membershipCacheEnabled: boolean;
  membershipCacheTTLSeconds: string;
  onMembershipCacheEnabledChange: (value: boolean) => void;
  onMembershipCacheTTLSecondsChange: (value: string) => void;
  recordCacheEnabled: boolean;
  recordCacheTTLSeconds: string;
  onRecordCacheEnabledChange: (value: boolean) => void;
  onRecordCacheTTLSecondsChange: (value: string) => void;
}) {
  const { t } = useTranslation('segments');

  return (
    <div className="space-y-3 rounded-2xl border border-border/70 bg-muted/10 p-4">
      <div className="space-y-1">
        <p className="text-sm font-semibold">{t('cache.title')}</p>
        <p className="text-muted-foreground text-xs">{t('cache.help')}</p>
      </div>
      <CacheRow
        enabled={membershipCacheEnabled}
        ttlSeconds={membershipCacheTTLSeconds}
        label={t('cache.membership')}
        help={t('cache.membershipHelp')}
        inputId="segment-membership-cache-ttl"
        onEnabledChange={onMembershipCacheEnabledChange}
        onTTLChange={onMembershipCacheTTLSecondsChange}
      />
      <CacheRow
        enabled={recordCacheEnabled}
        ttlSeconds={recordCacheTTLSeconds}
        label={t('cache.record')}
        help={t('cache.recordHelp')}
        inputId="segment-record-cache-ttl"
        onEnabledChange={onRecordCacheEnabledChange}
        onTTLChange={onRecordCacheTTLSecondsChange}
      />
    </div>
  );
}

function CacheRow({
  enabled,
  ttlSeconds,
  label,
  help,
  inputId,
  onEnabledChange,
  onTTLChange,
}: {
  enabled: boolean;
  ttlSeconds: string;
  label: string;
  help: string;
  inputId: string;
  onEnabledChange: (value: boolean) => void;
  onTTLChange: (value: string) => void;
}) {
  return (
    <div className="flex flex-col gap-3 rounded-xl border border-border/60 bg-background/80 p-3 md:flex-row md:items-center md:justify-between">
      <div className="flex items-center gap-3">
        <Switch checked={enabled} aria-label={label} onCheckedChange={onEnabledChange} />
        <div>
          <p className="text-sm font-medium">{label}</p>
          <p className="text-muted-foreground text-xs">{help}</p>
        </div>
      </div>
      <div className="flex w-full items-center gap-2 md:max-w-64">
        <Label htmlFor={inputId} className="shrink-0 text-sm">
          TTL
        </Label>
        <Input
          id={inputId}
          value={ttlSeconds}
          disabled={!enabled}
          onChange={(event) => onTTLChange(event.target.value)}
          placeholder="300"
        />
      </div>
    </div>
  );
}

function normalizePositiveInt(value: string, fallback: number) {
  const parsed = Number.parseInt(value, 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

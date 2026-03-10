import { zodResolver } from '@hookform/resolvers/zod';
import { useMemo } from 'react';
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
        onOpenChange(false);
        onSuccess?.(key);
      },
      onError: (error: unknown) => toast.error(getVisibleErrorMessage(error, t('form.error'))),
    };

    if (isEditing) {
      updateSegment.mutate(
        { key: segment.key, data: { name: data.name, description: data.description } },
        { onSuccess: () => callbacks.onSuccess(segment.key), onError: callbacks.onError },
      );
    } else {
      createSegment.mutate(
        { key: derivedKey, name: data.name, description: data.description },
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

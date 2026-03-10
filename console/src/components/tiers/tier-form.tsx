import { zodResolver } from '@hookform/resolvers/zod';
import { useEffect } from 'react';
import { useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { z } from 'zod';

import { IconPicker } from './icon-picker';

import type { Tier } from '@/api/types';

import { ColorPicker } from '@/components/shared/color-picker';
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
import { useCreateTier, useUpdateTier } from '@/mutations/tier-mutations';

interface TierFormProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  tier?: Tier | null;
}

export function TierForm({ open, onOpenChange, tier }: TierFormProps) {
  const { t } = useTranslation('tiers');
  const createTier = useCreateTier();
  const updateTier = useUpdateTier();

  const isEdit = !!tier;
  const isPending = createTier.isPending || updateTier.isPending;

  const schema = z.object({
    name: z.string().min(1, t('form.nameRequired')),
    level: z.coerce.number().min(1, t('form.levelRequired')),
    color: z.string().min(1, t('form.colorRequired')),
    icon: z.string().min(1, t('form.iconRequired')),
  });

  type FormValues = z.infer<typeof schema>;

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: '',
      level: 1,
      color: '#3B82F6',
      icon: 'builtin:star',
    },
  });

  const colorValue = watch('color');
  const iconValue = watch('icon');

  useEffect(() => {
    if (open) {
      if (tier) {
        reset({
          name: tier.name,
          level: tier.level,
          color: tier.color,
          icon: tier.icon,
        });
      } else {
        reset({
          name: '',
          level: 1,
          color: '#3B82F6',
          icon: 'builtin:star',
        });
      }
    }
  }, [open, tier, reset]);

  useSubmissionLoadingModal(isPending, isEdit ? 'update' : 'create');

  const onSubmit = (data: FormValues) => {
    if (isEdit) {
      updateTier.mutate(
        { key: tier.key, data },
        {
          onSuccess: () => {
            toast.success(t('form.success'));
            onOpenChange(false);
          },
          onError: () => {
            toast.error(t('form.error'));
          },
        },
      );
    } else {
      createTier.mutate(data, {
        onSuccess: () => {
          toast.success(t('form.success'));
          reset();
          onOpenChange(false);
        },
        onError: () => {
          toast.error(t('form.error'));
        },
      });
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isEdit ? t('form.editTitle') : t('form.createTitle')}</DialogTitle>
          <DialogDescription>
            {isEdit ? t('form.editTitle') : t('form.createTitle')}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">{t('fields.name')}</Label>
            <Input id="name" {...register('name')} />
            {errors.name ? (
              <p className="text-destructive text-sm">{errors.name.message}</p>
            ) : null}
          </div>

          <div className="space-y-2">
            <Label htmlFor="level">{t('fields.level')}</Label>
            <Input id="level" type="number" min={1} {...register('level')} />
            {errors.level ? (
              <p className="text-destructive text-sm">{errors.level.message}</p>
            ) : null}
          </div>

          <div className="space-y-2">
            <Label>{t('fields.color')}</Label>
            <ColorPicker value={colorValue} onChange={(c) => setValue('color', c)} />
            {errors.color ? (
              <p className="text-destructive text-sm">{errors.color.message}</p>
            ) : null}
          </div>

          <div className="space-y-2">
            <Label>{t('fields.icon')}</Label>
            <IconPicker
              value={iconValue}
              onChange={(i) => setValue('icon', i)}
              color={colorValue}
            />
            {errors.icon ? (
              <p className="text-destructive text-sm">{errors.icon.message}</p>
            ) : null}
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t('actions.cancel', { ns: 'common' })}
            </Button>
            <Button type="submit" disabled={isPending}>
              {isEdit ? t('actions.save', { ns: 'common' }) : t('createTier')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

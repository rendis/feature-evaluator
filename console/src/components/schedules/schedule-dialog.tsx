import { zodResolver } from '@hookform/resolvers/zod';
import { useForm, useWatch } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { z } from 'zod';

import type { Feature } from '@/api/types';

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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useSubmissionLoadingModal } from '@/hooks/use-global-loading';
import { useCreateSchedule } from '@/mutations/schedule-mutations';

const scheduleSchema = z.object({
  changeType: z.enum(['toggle', 'default_value', 'environment']),
  scheduledAt: z.string().min(1),
  toggleValue: z.boolean().optional(),
  defaultValue: z.string().optional(),
});

type FormValues = z.infer<typeof scheduleSchema>;

interface ScheduleDialogProps {
  feature: Feature;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function ScheduleChangeTypeField({
  changeType,
  setValue,
}: {
  changeType: FormValues['changeType'];
  setValue: ReturnType<typeof useForm<FormValues>>['setValue'];
}) {
  const { t } = useTranslation('schedules');

  return (
    <div className="space-y-2">
      <Label>{t('form.changeType')}</Label>
      <Select value={changeType} onValueChange={(value) => setValue('changeType', value as FormValues['changeType'])}>
        <SelectTrigger>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="toggle">{t('changeTypes.toggle')}</SelectItem>
          <SelectItem value="default_value">{t('changeTypes.default_value')}</SelectItem>
          <SelectItem value="environment">{t('changeTypes.environment')}</SelectItem>
        </SelectContent>
      </Select>
    </div>
  );
}

function SchedulePayloadField({
  changeType,
  register,
  setValue,
  toggleValue,
}: {
  changeType: FormValues['changeType'];
  register: ReturnType<typeof useForm<FormValues>>['register'];
  setValue: ReturnType<typeof useForm<FormValues>>['setValue'];
  toggleValue?: boolean;
}) {
  const { t } = useTranslation('schedules');

  if (changeType === 'toggle') {
    return (
      <div className="space-y-2">
        <Label>{t('form.payload')}</Label>
        <Select value={String(toggleValue)} onValueChange={(value) => setValue('toggleValue', value === 'true')}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="true">{t('form.toggleEnable')}</SelectItem>
            <SelectItem value="false">{t('form.toggleDisable')}</SelectItem>
          </SelectContent>
        </Select>
      </div>
    );
  }

  if (changeType === 'default_value') {
    return (
      <div className="space-y-2">
        <Label htmlFor="sched-default-value">{t('form.newDefaultValue')}</Label>
        <Input id="sched-default-value" {...register('defaultValue')} className="font-mono" />
      </div>
    );
  }

  return null;
}

function ScheduleDateField({
  errors,
  register,
}: {
  errors: ReturnType<typeof useForm<FormValues>>['formState']['errors'];
  register: ReturnType<typeof useForm<FormValues>>['register'];
}) {
  const { t } = useTranslation('schedules');

  return (
    <div className="space-y-2">
      <Label htmlFor="sched-date">{t('form.scheduledAt')}</Label>
      <Input id="sched-date" type="datetime-local" {...register('scheduledAt')} />
      {errors.scheduledAt && <p className="text-destructive text-sm">{t('form.dateRequired')}</p>}
      <p className="text-muted-foreground text-xs">{t('form.scheduledAtHelper')}</p>
    </div>
  );
}

export function ScheduleDialog({ feature, open, onOpenChange }: ScheduleDialogProps) {
  const { t } = useTranslation('schedules');
  const createSchedule = useCreateSchedule(feature.key);

  const {
    control,
    register,
    handleSubmit,
    setValue,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(scheduleSchema),
    defaultValues: {
      changeType: 'toggle',
      scheduledAt: '',
      toggleValue: !feature.enabled,
      defaultValue: String(feature.defaultValue),
    },
  });

  const changeType = useWatch({ control, name: 'changeType' });
  const toggleValue = useWatch({ control, name: 'toggleValue' });
  useSubmissionLoadingModal(createSchedule.isPending, 'create');

  const onSubmit = (data: FormValues) => {
    const scheduledAt = new Date(data.scheduledAt);
    if (scheduledAt <= new Date()) {
      toast.error(t('form.dateFuture'));
      return;
    }

    let payload: Record<string, unknown> = {};
    if (data.changeType === 'toggle') {
      payload = { enabled: data.toggleValue };
    } else if (data.changeType === 'default_value') {
      payload = { defaultValue: data.defaultValue };
    }

    createSchedule.mutate(
      {
        changeType: data.changeType,
        payload,
        scheduledAt: scheduledAt.toISOString(),
      },
      {
        onSuccess: () => {
          toast.success(t('form.success'));
          reset();
          onOpenChange(false);
        },
        onError: () => toast.error(t('form.error')),
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('form.title')}</DialogTitle>
          <DialogDescription>{t('form.description')}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <ScheduleChangeTypeField changeType={changeType} setValue={setValue} />
          <SchedulePayloadField
            changeType={changeType}
            register={register}
            setValue={setValue}
            toggleValue={toggleValue}
          />
          <ScheduleDateField errors={errors} register={register} />

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t('actions.cancel', { ns: 'common' })}
            </Button>
            <Button type="submit" disabled={createSchedule.isPending}>
              {t('scheduleChange')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

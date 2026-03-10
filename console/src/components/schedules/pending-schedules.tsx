import { useQuery } from '@tanstack/react-query';
import { Clock, X } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import type { ScheduledChange } from '@/api/schedules';

import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useAppLocale, useLocaleFormatters } from '@/hooks/use-locale-formatters';
import { useCancelSchedule } from '@/mutations/schedule-mutations';
import { scheduleQueries } from '@/queries/schedule-queries';

function formatCountdown(targetDate: string): string {
  const diff = new Date(targetDate).getTime() - Date.now();
  if (diff <= 0) return '< 1m';

  const days = Math.floor(diff / (1000 * 60 * 60 * 24));
  const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
  const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));

  const parts: string[] = [];
  if (days > 0) parts.push(`${days}d`);
  if (hours > 0) parts.push(`${hours}h`);
  if (minutes > 0 || parts.length === 0) parts.push(`${minutes}m`);
  return parts.join(' ');
}

function formatCountdownParts(targetDate: string, locale: 'es' | 'en'): string {
  const countdown = formatCountdown(targetDate);
  return locale === 'en'
    ? countdown
    : countdown.replace('d', ' d').replace('h', ' h').replace('m', ' min').trim();
}

function ScheduleCard({ schedule }: { schedule: ScheduledChange }) {
  const { t } = useTranslation('schedules');
  const locale = useAppLocale();
  const { formatDateTime } = useLocaleFormatters();
  const cancelSchedule = useCancelSchedule();
  const [cancelOpen, setCancelOpen] = useState(false);
  const [countdown, setCountdown] = useState(formatCountdownParts(schedule.scheduledAt, locale));

  useEffect(() => {
    const interval = setInterval(() => {
      setCountdown(formatCountdownParts(schedule.scheduledAt, locale));
    }, 60_000);
    return () => clearInterval(interval);
  }, [locale, schedule.scheduledAt]);

  const handleCancel = () => {
    cancelSchedule.mutate(schedule.id, {
      onSuccess: () => {
        toast.success(t('cancel.success'));
        setCancelOpen(false);
      },
      onError: () => toast.error(t('cancel.error')),
    });
  };

  const statusVariant = schedule.status === 'pending'
    ? 'warning'
    : schedule.status === 'completed'
      ? 'success'
      : schedule.status === 'failed'
        ? 'destructive'
        : 'secondary';

  return (
    <>
      <div className="flex items-center justify-between rounded-md border p-3">
        <div className="flex items-center gap-3">
          <Clock className="text-muted-foreground h-4 w-4 shrink-0" />
          <div>
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium">
                {t(`changeTypes.${schedule.changeType}`)}
              </span>
              <Badge variant={statusVariant} className="text-xs">
                {t(`status.${schedule.status}`)}
              </Badge>
              {schedule.status === 'pending' && (
                <span className="text-muted-foreground text-xs font-mono">
                  {t('countdown.in')} {countdown}
                </span>
              )}
            </div>
            <p className="text-muted-foreground text-xs">
              {t('scheduledFor')} {formatDateTime(schedule.scheduledAt)}
              {' - '}
              {t('scheduledBy')} {schedule.createdBy}
            </p>
          </div>
        </div>
        {schedule.status === 'pending' && (
          <Button
            variant="ghost"
            size="icon"
            className="text-destructive h-7 w-7"
            onClick={() => setCancelOpen(true)}
          >
            <X className="h-3 w-3" />
          </Button>
        )}
      </div>
      <ConfirmDialog
        open={cancelOpen}
        onOpenChange={setCancelOpen}
        title={t('cancel.title')}
        description={t('cancel.description')}
        variant="destructive"
        onConfirm={handleCancel}
        loading={cancelSchedule.isPending}
      />
    </>
  );
}

interface PendingSchedulesProps {
  featureKey: string;
}

export function PendingSchedules({ featureKey }: PendingSchedulesProps) {
  const { t } = useTranslation('schedules');
  const { data: schedules = [] } = useQuery(scheduleQueries.list(featureKey));

  const activeSchedules = schedules.filter(
    (s) => s.status === 'pending' || s.status === 'executing',
  );

  if (activeSchedules.length === 0) return null;

  return (
    <div className="space-y-3">
      <h2 className="text-lg font-semibold">
        {t('title')}
        <span className="text-muted-foreground ml-2 text-sm font-normal">
          ({activeSchedules.length})
        </span>
      </h2>
      <div className="space-y-2">
        {activeSchedules.map((schedule) => (
          <ScheduleCard key={schedule.id} schedule={schedule} />
        ))}
      </div>
    </div>
  );
}

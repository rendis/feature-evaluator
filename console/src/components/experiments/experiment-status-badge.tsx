import { useTranslation } from 'react-i18next';

import type { ExperimentStatus } from '@/api/types';

import { Badge } from '@/components/ui/badge';

const statusVariant: Record<ExperimentStatus, 'secondary' | 'success' | 'warning' | 'default'> = {
  draft: 'secondary',
  running: 'success',
  paused: 'warning',
  completed: 'default',
};

export function ExperimentStatusBadge({ status }: { status: ExperimentStatus }) {
  const { t } = useTranslation('experiments');
  return <Badge variant={statusVariant[status]}>{t(`status.${status}`)}</Badge>;
}

import { useTranslation } from 'react-i18next';

import type { FeatureListItem } from '@/api/types';

import { Badge } from '@/components/ui/badge';

type FeatureStatus = 'disabled' | 'active' | 'scheduled' | 'expired';

const statusVariant: Record<FeatureStatus, 'secondary' | 'success' | 'default' | 'warning'> = {
  disabled: 'secondary',
  active: 'success',
  scheduled: 'default',
  expired: 'warning',
};

function getFeatureStatus(feature: FeatureListItem): FeatureStatus {
  if (!feature.enabled) return 'disabled';
  const now = Date.now();
  const activeFrom = 'activeFrom' in feature ? feature.activeFrom : undefined;
  const activeUntil = 'activeUntil' in feature ? feature.activeUntil : undefined;
  if (activeFrom && new Date(activeFrom).getTime() > now) return 'scheduled';
  if (activeUntil && new Date(activeUntil).getTime() < now) return 'expired';
  return 'active';
}

export function FeatureStatusBadge({ feature }: { feature: FeatureListItem }) {
  const { t } = useTranslation('features');
  const status = getFeatureStatus(feature);

  return (
    <Badge variant={statusVariant[status]} className="text-xs">
      {t(`status.${status}`)}
    </Badge>
  );
}

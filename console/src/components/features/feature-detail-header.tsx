import { Link, useNavigate } from '@tanstack/react-router';
import { ArrowLeft, Calendar, Package, Pencil, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { FeatureStatusBadge } from './feature-status-badge';
import { FeatureToggle } from './feature-toggle';

import type { Feature } from '@/api/types';

import { ScheduleDialog } from '@/components/schedules/schedule-dialog';
import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { PermissionButton } from '@/components/shared/permission-button';
import { TagBadge } from '@/components/shared/tag-badge';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { useLocaleFormatters } from '@/hooks/use-locale-formatters';
import { usePermissions } from '@/hooks/use-permissions';
import { useDeleteFeature } from '@/mutations/feature-mutations';

interface FeatureDetailHeaderProps {
  feature: Feature;
}

function ScheduleInfo({ feature }: { feature: Feature }) {
  const { t } = useTranslation('features');
  const { formatDateTime } = useLocaleFormatters();
  const activeFromDisplay = feature.activeFrom ? formatDateTime(feature.activeFrom) : null;
  const activeUntilDisplay = feature.activeUntil ? formatDateTime(feature.activeUntil) : null;
  const hasSchedule = activeFromDisplay || activeUntilDisplay;
  const hasEnvironments = feature.environments && feature.environments.length > 0;

  return (
    <div className="text-muted-foreground mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm">
      {hasSchedule ? (
        <>
          {activeFromDisplay ? <span>{t('fields.activeFrom')}: {activeFromDisplay}</span> : null}
          {activeUntilDisplay ? <span>{t('fields.activeUntil')}: {activeUntilDisplay}</span> : null}
        </>
      ) : (
        <span>{t('schedule.alwaysActive')}</span>
      )}
      <span className="text-muted-foreground/50">|</span>
      {hasEnvironments ? (
        <span className="flex items-center gap-1">
          {t('fields.environments')}:
          {(feature.environments ?? []).map((env) => (
            <Badge key={env} variant="secondary" className="text-xs">
              {env}
            </Badge>
          ))}
        </span>
      ) : (
        <span>{t('environments.all')}</span>
      )}
    </div>
  );
}

function FeatureHeaderInfo({ feature }: { feature: Feature }) {
  const { can } = usePermissions();

  return (
    <div>
      <div className="flex items-center gap-3">
        <h1 className="text-2xl font-bold">{feature.name}</h1>
        <FeatureToggle featureKey={feature.key} enabled={feature.enabled} />
        <FeatureStatusBadge feature={feature} />
      </div>
      <p className="text-muted-foreground font-mono text-sm">{feature.key}</p>
      {feature.tags.length > 0 ? (
        <div className="mt-1 flex flex-wrap gap-1">
          {feature.tags.map((tag) => (
            <TagBadge key={tag.key} tag={tag} size="sm" />
          ))}
        </div>
      ) : null}
      {feature.packs && feature.packs.length > 0 ? (
        <div className="mt-1 flex flex-wrap items-center gap-1">
          <Package className="text-muted-foreground h-3 w-3" />
          {feature.packs.map((pack) => (
            <Link key={pack.key} to="/settings/packs/$packKey" params={{ packKey: pack.key }}>
              <Badge variant="outline" className="hover:bg-muted cursor-pointer text-xs">
                {pack.name}
              </Badge>
            </Link>
          ))}
        </div>
      ) : null}
      <FeatureHeaderTabs featureKey={feature.key} showObservability={can('audit.read')} />
    </div>
  );
}

function FeatureHeaderTabs({
  featureKey,
  showObservability,
}: {
  featureKey: string;
  showObservability: boolean;
}) {
  const { t } = useTranslation('features');

  return (
    <div className="mt-3 flex flex-wrap gap-2">
      <Link
        to="/features/$featureKey"
        params={{ featureKey }}
        className="fe-tab-trigger rounded-full"
        activeProps={{
          className: 'fe-tab-trigger rounded-full bg-primary text-primary-foreground',
          'aria-current': 'page',
        }}
      >
        {t('detail.tabs.summary', { defaultValue: 'Resumen' })}
      </Link>
      {showObservability ? (
        <Link
          to="/features/$featureKey/observability"
          params={{ featureKey }}
          className="fe-tab-trigger rounded-full"
          activeProps={{
            className: 'fe-tab-trigger rounded-full bg-primary text-primary-foreground',
            'aria-current': 'page',
          }}
        >
          {t('detail.tabs.observability', { defaultValue: 'Observability' })}
        </Link>
      ) : null}
    </div>
  );
}

function FeatureHeaderActions({
  feature,
  onDelete,
  onEdit,
  onSchedule,
}: {
  feature: Feature;
  onDelete: () => void;
  onEdit: () => void;
  onSchedule: () => void;
}) {
  const { t } = useTranslation(['features', 'schedules']);

  return (
    <div className="flex items-center gap-2">
      <Badge variant="secondary">{t(`valueTypes.${feature.valueType}`, { ns: 'features' })}</Badge>
      <PermissionButton permission="features.write" variant="outline" size="sm" onClick={onSchedule}>
        <Calendar className="mr-1 h-3 w-3" />
        {t('scheduleChange', { ns: 'schedules' })}
      </PermissionButton>
      <PermissionButton permission="features.write" variant="outline" size="sm" onClick={onEdit}>
        <Pencil className="mr-1 h-3 w-3" />
        {t('editFeature', { ns: 'features' })}
      </PermissionButton>
      <PermissionButton
        permission="features.write"
        variant="outline"
        size="sm"
        className="text-destructive"
        onClick={onDelete}
      >
        <Trash2 className="h-3 w-3" />
      </PermissionButton>
    </div>
  );
}

export function FeatureDetailHeader({ feature }: FeatureDetailHeaderProps) {
  const { t } = useTranslation(['features', 'schedules']);
  const navigate = useNavigate();
  const deleteFeature = useDeleteFeature();
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [scheduleOpen, setScheduleOpen] = useState(false);

  const handleDelete = () => {
    deleteFeature.mutate(feature.key, {
      onSuccess: () => {
        toast.success(t('delete.success'));
        void navigate({ to: '/features' });
      },
      onError: () => toast.error(t('delete.error')),
    });
  };

  return (
    <>
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" onClick={() => navigate({ to: '/features' })}>
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <FeatureHeaderInfo feature={feature} />
        </div>
        <FeatureHeaderActions
          feature={feature}
          onDelete={() => setDeleteOpen(true)}
          onEdit={() =>
            navigate({ to: '/features/$featureKey/edit', params: { featureKey: feature.key } })
          }
          onSchedule={() => setScheduleOpen(true)}
        />
      </div>

      <ScheduleInfo feature={feature} />

      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t('delete.title', { ns: 'features' })}
        description={t('delete.description', { ns: 'features', key: feature.key })}
        variant="destructive"
        onConfirm={handleDelete}
        loading={deleteFeature.isPending}
      />
      <ScheduleDialog feature={feature} open={scheduleOpen} onOpenChange={setScheduleOpen} />
    </>
  );
}

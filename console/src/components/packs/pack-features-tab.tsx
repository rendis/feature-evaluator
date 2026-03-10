import { useSuspenseQuery } from '@tanstack/react-query';
import { Package, Plus, X } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import type { Feature, Pack } from '@/api/types';

import { ConfirmDialog } from '@/components/shared/confirm-dialog';
import { EmptyState } from '@/components/shared/empty-state';
import { PermissionButton } from '@/components/shared/permission-button';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { useSubmissionLoadingModal } from '@/hooks/use-global-loading';
import { useUpdatePack } from '@/mutations/pack-mutations';
import { featureQueries } from '@/queries/feature-queries';

interface PackFeaturesTabProps {
  pack: Pack;
}

function updatePackFeatures(
  pack: Pack,
  featureKeys: string[],
  updatePack: ReturnType<typeof useUpdatePack>,
  handlers: { onSuccess: () => void; onError: () => void },
) {
  updatePack.mutate(
    { key: pack.key, data: { name: pack.name, description: pack.description, featureKeys } },
    handlers,
  );
}

function filterPackFeatures(allFeatures: Feature[], pack: Pack) {
  return allFeatures.filter((feature) => pack.featureKeys.includes(feature.key));
}

function filterAvailableFeatures(allFeatures: Feature[], pack: Pack, search: string) {
  const normalizedSearch = search.toLowerCase();

  return allFeatures.filter(
    (feature) =>
      !pack.featureKeys.includes(feature.key) &&
      (feature.name.toLowerCase().includes(normalizedSearch) ||
        feature.key.toLowerCase().includes(normalizedSearch)),
  );
}

function PackFeaturesHeader({ onAdd }: { onAdd: () => void }) {
  const { t } = useTranslation('packs');

  return (
    <div className="flex items-center justify-between">
      <h2 className="text-lg font-semibold">{t('tabs.features')}</h2>
      <PermissionButton permission="features.write" size="sm" onClick={onAdd}>
        <Plus className="mr-1 h-3 w-3" />
        {t('features.add')}
      </PermissionButton>
    </div>
  );
}

function PackFeaturesEmptyState({ onAdd }: { onAdd: () => void }) {
  const { t } = useTranslation('packs');

  return (
    <EmptyState
      icon={<Package className="h-10 w-10" />}
      title={t('features.empty.title')}
      description={t('features.empty.description')}
      action={
        <PermissionButton permission="features.write" size="sm" onClick={onAdd}>
          {t('features.add')}
        </PermissionButton>
      }
    />
  );
}

function PackFeatureCard({
  feature,
  onRemove,
}: {
  feature: Feature;
  onRemove: (feature: Feature) => void;
}) {
  const { t } = useTranslation('packs');

  return (
    <div className="group relative rounded-lg border p-4 hover:shadow-sm">
      <div className="flex items-start justify-between">
        <div className="min-w-0 flex-1">
          <p className="font-medium">{feature.name}</p>
          <p className="text-muted-foreground mt-0.5 font-mono text-xs">{feature.key}</p>
        </div>
        <PermissionButton
          permission="features.write"
          variant="ghost"
          size="icon"
          className="h-6 w-6 opacity-0 group-hover:opacity-100"
          onClick={() => onRemove(feature)}
        >
          <X className="h-3 w-3" />
        </PermissionButton>
      </div>
      <div className="mt-2 flex gap-1">
        <Badge variant={feature.enabled ? 'success' : 'secondary'}>
          {feature.enabled ? t('enabled') : t('disabled')}
        </Badge>
        <Badge variant="outline">{feature.valueType}</Badge>
      </div>
    </div>
  );
}

function PackFeaturesGrid({
  features,
  onRemove,
}: {
  features: Feature[];
  onRemove: (feature: Feature) => void;
}) {
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {features.map((feature) => (
        <PackFeatureCard key={feature.key} feature={feature} onRemove={onRemove} />
      ))}
    </div>
  );
}

function AddFeaturesResults({
  available,
  selected,
  onToggle,
}: {
  available: Feature[];
  selected: string[];
  onToggle: (key: string) => void;
}) {
  const { t } = useTranslation('packs');

  return (
    <div className="max-h-[60vh] space-y-1 overflow-y-auto">
      {available.length === 0 ? (
        <p className="text-muted-foreground py-8 text-center text-sm">{t('features.addDialog.noResults')}</p>
      ) : (
        available.map((feature) => (
          <label
            key={feature.key}
            className="flex cursor-pointer items-center gap-3 rounded-md px-3 py-2 hover:bg-muted"
          >
            <input
              type="checkbox"
              checked={selected.includes(feature.key)}
              onChange={() => onToggle(feature.key)}
              className="accent-primary"
            />
            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium">{feature.name}</p>
              <p className="text-muted-foreground font-mono text-xs">{feature.key}</p>
            </div>
            <Badge variant="outline" className="text-xs">
              {feature.valueType}
            </Badge>
          </label>
        ))
      )}
    </div>
  );
}

function AddFeaturesDialogActions({
  disabled,
  onCancel,
  onConfirm,
  selectedCount,
}: {
  disabled: boolean;
  onCancel: () => void;
  onConfirm: () => void;
  selectedCount: number;
}) {
  const { t } = useTranslation('packs');

  return (
    <DialogFooter>
      <Button variant="outline" onClick={onCancel}>
        {t('cancel', { ns: 'common', defaultValue: 'Cancel' })}
      </Button>
      <Button onClick={onConfirm} disabled={disabled}>
        {t('features.addDialog.confirm', { count: selectedCount })}
      </Button>
    </DialogFooter>
  );
}

function AddFeaturesDialog({
  allFeatures,
  onOpenChange,
  open,
  pack,
}: {
  allFeatures: Feature[];
  onOpenChange: (open: boolean) => void;
  open: boolean;
  pack: Pack;
}) {
  const { t } = useTranslation('packs');
  const [search, setSearch] = useState('');
  const [selected, setSelected] = useState<string[]>([]);
  const updatePack = useUpdatePack();
  const available = filterAvailableFeatures(allFeatures, pack, search);
  useSubmissionLoadingModal(updatePack.isPending, 'update');

  const toggleFeature = (key: string) => {
    setSelected((previous) => (
      previous.includes(key) ? previous.filter((currentKey) => currentKey !== key) : [...previous, key]
    ));
  };
  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      setSelected([]);
      setSearch('');
    }
    onOpenChange(nextOpen);
  };
  const handleAdd = () => {
    updatePackFeatures(
      pack,
      [...pack.featureKeys, ...selected],
      updatePack,
      {
        onSuccess: () => {
          setSelected([]);
          setSearch('');
          onOpenChange(false);
          toast.success(t('features.addSuccess'));
        },
        onError: () => toast.error(t('features.addError')),
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('features.addDialog.title')}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <Input
            placeholder={t('features.addDialog.search')}
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
          {selected.length > 0 ? (
            <p className="text-muted-foreground text-sm">
              {t('features.addDialog.selected', { count: selected.length })}
            </p>
          ) : null}
          <AddFeaturesResults available={available} selected={selected} onToggle={toggleFeature} />
        </div>
        <AddFeaturesDialogActions
          disabled={selected.length === 0 || updatePack.isPending}
          onCancel={() => handleOpenChange(false)}
          onConfirm={handleAdd}
          selectedCount={selected.length}
        />
      </DialogContent>
    </Dialog>
  );
}

function PackFeatureDialogs({
  addOpen,
  allFeatures,
  onAddOpenChange,
  onConfirmRemove,
  onRemoveOpenChange,
  pack,
  removeTarget,
  updatePack,
}: {
  addOpen: boolean;
  allFeatures: Feature[];
  onAddOpenChange: (open: boolean) => void;
  onConfirmRemove: () => void;
  onRemoveOpenChange: (open: boolean) => void;
  pack: Pack;
  removeTarget: Feature | null;
  updatePack: ReturnType<typeof useUpdatePack>;
}) {
  const { t } = useTranslation('packs');

  return (
    <>
      <AddFeaturesDialog
        open={addOpen}
        onOpenChange={onAddOpenChange}
        pack={pack}
        allFeatures={allFeatures}
      />
      <ConfirmDialog
        open={!!removeTarget}
        onOpenChange={onRemoveOpenChange}
        title={t('features.removeConfirm.title')}
        description={t('features.removeConfirm.description', { key: removeTarget?.key })}
        variant="destructive"
        confirmLabel={t('features.removeConfirm.confirm')}
        onConfirm={onConfirmRemove}
        loading={updatePack.isPending}
      />
    </>
  );
}

export function PackFeaturesTab({ pack }: PackFeaturesTabProps) {
  const { t } = useTranslation('packs');
  const [addOpen, setAddOpen] = useState(false);
  const [removeTarget, setRemoveTarget] = useState<Feature | null>(null);
  const updatePack = useUpdatePack();
  const { data: featuresResponse } = useSuspenseQuery(featureQueries.list({ pageSize: 200 }));
  const allFeatures = featuresResponse.data;
  const packFeatures = filterPackFeatures(allFeatures, pack);
  useSubmissionLoadingModal(updatePack.isPending, 'update');

  const handleRemoveFeature = () => {
    if (!removeTarget) return;
    updatePackFeatures(
      pack,
      pack.featureKeys.filter((key) => key !== removeTarget.key),
      updatePack,
      {
        onSuccess: () => {
          setRemoveTarget(null);
          toast.success(t('features.removeSuccess'));
        },
        onError: () => toast.error(t('features.removeError')),
      },
    );
  };

  return (
    <div className="space-y-4">
      <PackFeaturesHeader onAdd={() => setAddOpen(true)} />
      {packFeatures.length === 0 ? (
        <PackFeaturesEmptyState onAdd={() => setAddOpen(true)} />
      ) : (
        <PackFeaturesGrid features={packFeatures} onRemove={setRemoveTarget} />
      )}
      <PackFeatureDialogs
        addOpen={addOpen}
        allFeatures={allFeatures}
        onAddOpenChange={setAddOpen}
        onConfirmRemove={handleRemoveFeature}
        onRemoveOpenChange={(open) => !open && setRemoveTarget(null)}
        pack={pack}
        removeTarget={removeTarget}
        updatePack={updatePack}
      />
    </div>
  );
}

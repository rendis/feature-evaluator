import { zodResolver } from '@hookform/resolvers/zod';
import { useQuery, useSuspenseQuery } from '@tanstack/react-query';
import { X } from 'lucide-react';
import { useState } from 'react';
import { useForm, type UseFormRegisterReturn } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { z } from 'zod';

import type { Feature, Pack } from '@/api/types';

import { TierBadge } from '@/components/shared/tier-badge';
import { Badge } from '@/components/ui/badge';
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
import { localToUtc, utcToLocal } from '@/components/features/feature-builder-utils';
import { useSubmissionLoadingModal } from '@/hooks/use-global-loading';
import { getVisibleErrorMessage } from '@/lib/display-error';
import { buildNormalizedKeyFieldProps, resourceKeySchema, slugifyResourceKey } from '@/lib/resource-key';
import { useCreatePack, useUpdatePack } from '@/mutations/pack-mutations';
import { featureQueries } from '@/queries/feature-queries';
import { packQueries } from '@/queries/pack-queries';
import { tierQueries } from '@/queries/tier-queries';

const packSchema = z.object({
  key: resourceKeySchema,
  name: z.string().min(1).max(256),
  description: z.string().max(1024),
});

type FormValues = z.infer<typeof packSchema>;

interface PackFormProps {
  pack?: Pack;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: (key: string) => void;
}

interface SubmitPackParams {
  createPack: ReturnType<typeof useCreatePack>;
  isEditing: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: (key: string) => void;
  pack?: Pack;
  reset: ReturnType<typeof useForm<FormValues>>['reset'];
  selectedKeys: string[];
  selectedTierKey: string | null;
  selectedInheritsFrom: string[];
  trialUntil: string;
  setSelectedKeys: React.Dispatch<React.SetStateAction<string[]>>;
  setSelectedTierKey: React.Dispatch<React.SetStateAction<string | null>>;
  setSelectedInheritsFrom: React.Dispatch<React.SetStateAction<string[]>>;
  setTrialUntil: React.Dispatch<React.SetStateAction<string>>;
  t: ReturnType<typeof useTranslation<'packs'>>['t'];
  updatePack: ReturnType<typeof useUpdatePack>;
}

function filterAvailableFeatures(allFeatures: Feature[], featureSearch: string, selectedKeys: string[]) {
  const normalizedSearch = featureSearch.toLowerCase();

  return allFeatures.filter(
    (feature) =>
      !selectedKeys.includes(feature.key) &&
      (feature.name.toLowerCase().includes(normalizedSearch) ||
        feature.key.toLowerCase().includes(normalizedSearch)),
  );
}

function filterSelectedFeatures(allFeatures: Feature[], selectedKeys: string[]) {
  return allFeatures.filter((feature) => selectedKeys.includes(feature.key));
}

function createPackSubmitHandler({
  createPack,
  isEditing,
  onOpenChange,
  onSuccess,
  pack,
  reset,
  selectedKeys,
  selectedTierKey,
  selectedInheritsFrom,
  trialUntil,
  setSelectedKeys,
  setSelectedTierKey,
  setSelectedInheritsFrom,
  setTrialUntil,
  t,
  updatePack,
}: SubmitPackParams) {
  return (data: FormValues) => {
    const callbacks = {
      onSuccess: (key: string) => {
        toast.success(t('form.success'));
        reset();
        setSelectedKeys([]);
        setSelectedTierKey(null);
        setSelectedInheritsFrom([]);
        setTrialUntil('');
        onOpenChange(false);
        onSuccess?.(key);
      },
      onError: (error: unknown) => toast.error(getVisibleErrorMessage(error, t('form.error'))),
    };

    const tierKey = selectedTierKey || undefined;
    const inheritsFrom = selectedInheritsFrom.length > 0 ? selectedInheritsFrom : undefined;
    const trialUntilUtc = localToUtc(trialUntil) || undefined;

    if (isEditing && pack) {
      updatePack.mutate(
        {
          key: pack.key,
          data: {
            name: data.name,
            description: data.description,
            featureKeys: selectedKeys,
            tierKey: selectedTierKey,
            inheritsFrom,
            trialUntil: trialUntilUtc,
          },
        },
        { onSuccess: () => callbacks.onSuccess(pack.key), onError: callbacks.onError },
      );
      return;
    }

    createPack.mutate(
      {
        key: data.key,
        name: data.name,
        description: data.description || undefined,
        featureKeys: selectedKeys,
        tierKey,
        inheritsFrom,
        trialUntil: trialUntilUtc,
      },
      { onSuccess: (createdPack) => callbacks.onSuccess(createdPack.key), onError: callbacks.onError },
    );
  };
}

function PackDialogHeader({ isEditing }: { isEditing: boolean }) {
  const { t } = useTranslation('packs');

  return (
    <DialogHeader>
      <DialogTitle>{isEditing ? t('form.editTitle') : t('form.createTitle')}</DialogTitle>
      <DialogDescription>
        {t('form.dialogDescription', { defaultValue: 'Group features into a pack for bulk evaluation.' })}
      </DialogDescription>
    </DialogHeader>
  );
}

function PackMetadataFields({
  errors,
  isEditing,
  keyField,
  nameField,
  onNameChange,
  register,
}: {
  errors: ReturnType<typeof useForm<FormValues>>['formState']['errors'];
  isEditing: boolean;
  keyField: UseFormRegisterReturn<'key'>;
  nameField: UseFormRegisterReturn<'name'>;
  onNameChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
  register: ReturnType<typeof useForm<FormValues>>['register'];
}) {
  const { t } = useTranslation('packs');

  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="pack-name">{t('fields.name')}</Label>
        <Input id="pack-name" {...nameField} onChange={onNameChange} />
        {errors.name ? <p className="text-destructive text-sm">{t('form.nameRequired')}</p> : null}
      </div>

      <div className="space-y-2">
        <Label htmlFor="pack-key">{t('fields.key')}</Label>
        <Input id="pack-key" {...keyField} disabled={isEditing} className="font-mono" />
        {errors.key ? <p className="text-destructive text-sm">{t('form.keyInvalid')}</p> : null}
        <p className="text-muted-foreground text-xs">{t('form.keyPattern')}</p>
      </div>

      <div className="space-y-2">
        <Label htmlFor="pack-description">{t('fields.description')}</Label>
        <Input id="pack-description" {...register('description')} />
      </div>
    </>
  );
}

function SelectedFeatureBadges({
  features,
  onRemove,
}: {
  features: Feature[];
  onRemove: (key: string) => void;
}) {
  if (features.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-wrap gap-1">
      {features.map((feature) => (
        <Badge key={feature.key} variant="secondary" className="gap-1 pr-1">
          {feature.name}
          <button
            type="button"
            onClick={() => onRemove(feature.key)}
            className="hover:text-destructive ml-0.5 rounded-full p-0.5"
          >
            <X className="h-3 w-3" />
          </button>
        </Badge>
      ))}
    </div>
  );
}

function PackFeatureSelector({
  featureSearch,
  filteredFeatures,
  onAddFeature,
  onRemoveFeature,
  onSearchChange,
  selectedFeatures,
}: {
  featureSearch: string;
  filteredFeatures: Feature[];
  onAddFeature: (feature: Feature) => void;
  onRemoveFeature: (key: string) => void;
  onSearchChange: (value: string) => void;
  selectedFeatures: Feature[];
}) {
  const { t } = useTranslation('packs');

  return (
    <div className="space-y-2">
      <Label>{t('fields.features')}</Label>
      <SelectedFeatureBadges features={selectedFeatures} onRemove={onRemoveFeature} />
      <div className="relative">
        <Input
          placeholder={t('form.featureSearchPlaceholder')}
          value={featureSearch}
          onChange={(event) => onSearchChange(event.target.value)}
        />
        {featureSearch && filteredFeatures.length > 0 ? (
          <div className="bg-popover absolute z-10 mt-1 max-h-36 w-full overflow-y-auto rounded-md border shadow-md">
            {filteredFeatures.slice(0, 8).map((feature) => (
              <button
                key={feature.key}
                type="button"
                onClick={() => onAddFeature(feature)}
                className="hover:bg-muted flex w-full items-center gap-2 px-3 py-2 text-left text-sm"
              >
                <span className="font-medium">{feature.name}</span>
                <span className="text-muted-foreground font-mono text-xs">{feature.key}</span>
              </button>
            ))}
          </div>
        ) : null}
        {featureSearch && filteredFeatures.length === 0 ? (
          <div className="bg-popover absolute z-10 mt-1 w-full rounded-md border p-3 shadow-md">
            <p className="text-muted-foreground text-sm">{t('form.noFeaturesFound')}</p>
          </div>
        ) : null}
      </div>
    </div>
  );
}

function PackTierSelector({
  selectedTierKey,
  onTierChange,
}: {
  selectedTierKey: string | null;
  onTierChange: (value: string | null) => void;
}) {
  const { t } = useTranslation('packs');
  const { data: tiers } = useQuery(tierQueries.list());

  const selectedTier = tiers?.find((tier) => tier.key === selectedTierKey);

  return (
    <div className="space-y-2">
      <Label>{t('tier.label')}</Label>
      <div className="flex items-center gap-2">
        <Select
          value={selectedTierKey ?? '__none__'}
          onValueChange={(value) => onTierChange(value === '__none__' ? null : value)}
        >
          <SelectTrigger className="flex-1">
            <SelectValue placeholder={t('tier.placeholder')} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__none__">{t('tier.none')}</SelectItem>
            {tiers?.map((tier) => (
              <SelectItem key={tier.key} value={tier.key}>
                {tier.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {selectedTier ? <TierBadge tier={selectedTier} size="sm" /> : null}
      </div>
    </div>
  );
}

function PackInheritanceSelector({
  currentPackKey,
  selectedInheritsFrom,
  onInheritsFromChange,
}: {
  currentPackKey?: string;
  selectedInheritsFrom: string[];
  onInheritsFromChange: (keys: string[]) => void;
}) {
  const { t } = useTranslation('packs');
  const [inheritSearch, setInheritSearch] = useState('');
  const { data: packsResponse } = useQuery(packQueries.list({ pageSize: 200 }));
  const allPacks = packsResponse?.data ?? [];

  const normalizedSearch = inheritSearch.toLowerCase();
  const availablePacks = allPacks.filter(
    (p) =>
      p.key !== currentPackKey &&
      !selectedInheritsFrom.includes(p.key) &&
      (p.name.toLowerCase().includes(normalizedSearch) ||
        p.key.toLowerCase().includes(normalizedSearch)),
  );

  const selectedPacks = allPacks.filter((p) => selectedInheritsFrom.includes(p.key));

  return (
    <div className="space-y-2">
      <Label>{t('inheritance.label')}</Label>
      {selectedPacks.length > 0 ? (
        <div className="flex flex-wrap gap-1">
          {selectedPacks.map((p) => (
            <Badge key={p.key} variant="secondary" className="gap-1 pr-1">
              {p.name}
              <button
                type="button"
                onClick={() => onInheritsFromChange(selectedInheritsFrom.filter((k) => k !== p.key))}
                className="hover:text-destructive ml-0.5 rounded-full p-0.5"
              >
                <X className="h-3 w-3" />
              </button>
            </Badge>
          ))}
        </div>
      ) : null}
      <div className="relative">
        <Input
          placeholder={t('inheritance.placeholder')}
          value={inheritSearch}
          onChange={(event) => setInheritSearch(event.target.value)}
        />
        {inheritSearch && availablePacks.length > 0 ? (
          <div className="bg-popover absolute z-10 mt-1 max-h-36 w-full overflow-y-auto rounded-md border shadow-md">
            {availablePacks.slice(0, 8).map((p) => (
              <button
                key={p.key}
                type="button"
                onClick={() => {
                  onInheritsFromChange([...selectedInheritsFrom, p.key]);
                  setInheritSearch('');
                }}
                className="hover:bg-muted flex w-full items-center gap-2 px-3 py-2 text-left text-sm"
              >
                <span className="font-medium">{p.name}</span>
                <span className="text-muted-foreground font-mono text-xs">{p.key}</span>
              </button>
            ))}
          </div>
        ) : null}
        {inheritSearch && availablePacks.length === 0 ? (
          <div className="bg-popover absolute z-10 mt-1 w-full rounded-md border p-3 shadow-md">
            <p className="text-muted-foreground text-sm">{t('inheritance.noResults')}</p>
          </div>
        ) : null}
      </div>
    </div>
  );
}

function PackTrialField({
  trialUntil,
  onTrialUntilChange,
}: {
  trialUntil: string;
  onTrialUntilChange: (value: string) => void;
}) {
  const { t } = useTranslation('packs');

  return (
    <div className="space-y-2">
      <Label htmlFor="pack-trialUntil">{t('trial.label')}</Label>
      <div className="flex gap-1">
        <Input
          id="pack-trialUntil"
          type="datetime-local"
          value={trialUntil}
          onChange={(e) => onTrialUntilChange(e.target.value)}
          className="flex-1"
        />
        {trialUntil ? (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="shrink-0"
            onClick={() => onTrialUntilChange('')}
          >
            <X className="h-4 w-4" />
          </Button>
        ) : null}
      </div>
      <p className="text-muted-foreground text-xs">{t('trial.helper')}</p>
    </div>
  );
}

function PackFormActions({
  isPending,
  onOpenChange,
}: {
  isPending: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { t } = useTranslation('packs');

  return (
    <DialogFooter>
      <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
        {t('actions.cancel', { ns: 'common' })}
      </Button>
      <Button type="submit" disabled={isPending}>
        {t('actions.save', { ns: 'common' })}
      </Button>
    </DialogFooter>
  );
}

function PackFormContent({
  currentPackKey,
  errors,
  featureSearch,
  filteredFeatures,
  handleSubmit,
  isEditing,
  isPending,
  keyField,
  nameField,
  onAddFeature,
  onNameChange,
  onOpenChange,
  onRemoveFeature,
  onSearchChange,
  onSubmit,
  register,
  selectedFeatures,
  selectedTierKey,
  onTierChange,
  selectedInheritsFrom,
  onInheritsFromChange,
  trialUntil,
  onTrialUntilChange,
}: {
  currentPackKey?: string;
  errors: ReturnType<typeof useForm<FormValues>>['formState']['errors'];
  featureSearch: string;
  filteredFeatures: Feature[];
  handleSubmit: ReturnType<typeof useForm<FormValues>>['handleSubmit'];
  isEditing: boolean;
  isPending: boolean;
  keyField: UseFormRegisterReturn<'key'>;
  nameField: UseFormRegisterReturn<'name'>;
  onAddFeature: (feature: Feature) => void;
  onNameChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
  onOpenChange: (open: boolean) => void;
  onRemoveFeature: (key: string) => void;
  onSearchChange: (value: string) => void;
  onSubmit: (data: FormValues) => void;
  register: ReturnType<typeof useForm<FormValues>>['register'];
  selectedFeatures: Feature[];
  selectedTierKey: string | null;
  onTierChange: (value: string | null) => void;
  selectedInheritsFrom: string[];
  onInheritsFromChange: (keys: string[]) => void;
  trialUntil: string;
  onTrialUntilChange: (value: string) => void;
}) {
  return (
    <DialogContent className="sm:max-w-lg">
      <PackDialogHeader isEditing={isEditing} />
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <PackMetadataFields
          errors={errors}
          isEditing={isEditing}
          keyField={keyField}
          nameField={nameField}
          onNameChange={onNameChange}
          register={register}
        />
        <PackFeatureSelector
          featureSearch={featureSearch}
          filteredFeatures={filteredFeatures}
          onAddFeature={onAddFeature}
          onRemoveFeature={onRemoveFeature}
          onSearchChange={onSearchChange}
          selectedFeatures={selectedFeatures}
        />

        <div className="border-t pt-4">
          <PackTierSelector selectedTierKey={selectedTierKey} onTierChange={onTierChange} />
        </div>

        <PackInheritanceSelector
          currentPackKey={currentPackKey}
          selectedInheritsFrom={selectedInheritsFrom}
          onInheritsFromChange={onInheritsFromChange}
        />

        <PackTrialField trialUntil={trialUntil} onTrialUntilChange={onTrialUntilChange} />

        <PackFormActions isPending={isPending} onOpenChange={onOpenChange} />
      </form>
    </DialogContent>
  );
}

export function PackForm({ pack, open, onOpenChange, onSuccess }: PackFormProps) {
  const { t } = useTranslation('packs');
  const createPack = useCreatePack();
  const updatePack = useUpdatePack();
  const isEditing = !!pack;
  const { data: featuresResponse } = useSuspenseQuery(featureQueries.list({ pageSize: 200 }));
  const allFeatures = featuresResponse.data;
  const [selectedKeys, setSelectedKeys] = useState<string[]>(pack?.featureKeys ?? []);
  const [featureSearch, setFeatureSearch] = useState('');
  const [autoSlug, setAutoSlug] = useState(!isEditing);
  const [selectedTierKey, setSelectedTierKey] = useState<string | null>(pack?.tierKey ?? null);
  const [selectedInheritsFrom, setSelectedInheritsFrom] = useState<string[]>(
    pack?.inheritsFrom?.map((p) => p.key) ?? [],
  );
  const [trialUntil, setTrialUntil] = useState<string>(utcToLocal(pack?.trialUntil));
  const { register, handleSubmit, setValue, reset, formState: { errors } } = useForm<FormValues>({
    resolver: zodResolver(packSchema),
    defaultValues: {
      key: pack?.key ?? '',
      name: pack?.name ?? '',
      description: pack?.description ?? '',
    },
  });
  const keyField = buildNormalizedKeyFieldProps({ name: 'key', onChangeNormalized: () => setAutoSlug(false), register, setValue });
  const nameField = register('name');
  const filteredFeatures = filterAvailableFeatures(allFeatures, featureSearch, selectedKeys);
  const selectedFeatures = filterSelectedFeatures(allFeatures, selectedKeys);
  const isPending = createPack.isPending || updatePack.isPending;
  const onSubmit = createPackSubmitHandler({
    createPack,
    isEditing,
    onOpenChange,
    onSuccess,
    pack,
    reset,
    selectedKeys,
    selectedTierKey,
    selectedInheritsFrom,
    trialUntil,
    setSelectedKeys,
    setSelectedTierKey,
    setSelectedInheritsFrom,
    setTrialUntil,
    t,
    updatePack,
  });
  useSubmissionLoadingModal(isPending, isEditing ? 'update' : 'create');
  const addFeature = (feature: Feature) => {
    setSelectedKeys((previous) => [...previous, feature.key]);
    setFeatureSearch('');
  };
  const removeFeature = (key: string) => setSelectedKeys((previous) => previous.filter((currentKey) => currentKey !== key));
  const handleNameChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const name = event.target.value;
    nameField.onChange(event);
    if (autoSlug && !isEditing) {
      setValue('key', slugifyResourceKey(name), {
        shouldDirty: true,
        shouldValidate: true,
      });
    }
  };
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <PackFormContent
        currentPackKey={pack?.key}
        errors={errors}
        featureSearch={featureSearch}
        filteredFeatures={filteredFeatures}
        handleSubmit={handleSubmit}
        isEditing={isEditing}
        isPending={isPending}
        keyField={keyField}
        nameField={nameField}
        onAddFeature={addFeature}
        onNameChange={handleNameChange}
        onOpenChange={onOpenChange}
        onRemoveFeature={removeFeature}
        onSearchChange={setFeatureSearch}
        onSubmit={onSubmit}
        register={register}
        selectedFeatures={selectedFeatures}
        selectedTierKey={selectedTierKey}
        onTierChange={setSelectedTierKey}
        selectedInheritsFrom={selectedInheritsFrom}
        onInheritsFromChange={setSelectedInheritsFrom}
        trialUntil={trialUntil}
        onTrialUntilChange={setTrialUntil}
      />
    </Dialog>
  );
}

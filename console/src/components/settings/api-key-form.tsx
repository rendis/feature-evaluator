import { zodResolver } from '@hookform/resolvers/zod';
import { useForm, useWatch } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { z } from 'zod';

import type { ApiKeyPermission } from '@/api/types';

import { AllowedAPIKeyPermissions } from '@/api/types';
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
import { useCreateApiKey } from '@/mutations/api-key-mutations';

const apiKeySchema = z.object({
  name: z.string().min(1),
  description: z.string().optional(),
  permissions: z.array(z.string()).optional(),
  expiresAt: z.string().optional(),
});

type FormValues = z.infer<typeof apiKeySchema>;

interface ApiKeyFormProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (rawKey: string) => void;
}

function ApiKeyPermissionsSection({
  permissionsValue,
  togglePermission,
}: {
  permissionsValue: string[];
  togglePermission: (permission: string) => void;
}) {
  const { t } = useTranslation('settings');

  return (
    <div className="space-y-2">
      <Label>{t('apiKeys.permissions')}</Label>
      <div className="space-y-2">
        {AllowedAPIKeyPermissions.map((permission) => (
          <label key={permission} className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={permissionsValue.includes(permission)}
              onChange={() => togglePermission(permission)}
              className="accent-primary"
            />
            {t(`apiKeys.perm.${permission}`)}
          </label>
        ))}
      </div>
    </div>
  );
}

function ApiKeyDescriptionField({
  register,
}: {
  register: ReturnType<typeof useForm<FormValues>>['register'];
}) {
  const { t } = useTranslation('settings');

  return (
    <div className="space-y-2">
      <Label htmlFor="apikey-description">{t('apiKeys.description')}</Label>
      <textarea
        id="apikey-description"
        className="border-input bg-background ring-offset-background placeholder:text-muted-foreground focus-visible:ring-ring flex min-h-[60px] w-full rounded-md border px-3 py-2 text-sm focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
        placeholder={t('apiKeys.descriptionPlaceholder')}
        {...register('description')}
      />
    </div>
  );
}

function ApiKeyDialogHeader() {
  const { t } = useTranslation('settings');

  return (
    <DialogHeader>
      <DialogTitle>{t('apiKeys.createTitle')}</DialogTitle>
      <DialogDescription>
        {t('apiKeys.createDescription', {
          defaultValue: 'Configure a new API key for programmatic access.',
        })}
      </DialogDescription>
    </DialogHeader>
  );
}

function ApiKeyFormActions({
  isPending,
  onClose,
}: {
  isPending: boolean;
  onClose: () => void;
}) {
  const { t } = useTranslation('settings');

  return (
    <DialogFooter>
      <Button type="button" variant="outline" onClick={onClose}>
        {t('actions.cancel', { ns: 'common' })}
      </Button>
      <Button type="submit" disabled={isPending}>
        {t('apiKeys.create')}
      </Button>
    </DialogFooter>
  );
}

export function ApiKeyForm({ open, onOpenChange, onCreated }: ApiKeyFormProps) {
  const { t } = useTranslation('settings');
  const createApiKey = useCreateApiKey();

  const {
    control,
    register,
    handleSubmit,
    setValue,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(apiKeySchema),
    defaultValues: { name: '', description: '', permissions: [], expiresAt: '' },
  });

  const permissionsValue = useWatch({ control, name: 'permissions' }) ?? [];
  useSubmissionLoadingModal(createApiKey.isPending, 'create');

  const togglePermission = (perm: string) => {
    const current = permissionsValue;
    const next = current.includes(perm) ? current.filter((p) => p !== perm) : [...current, perm];
    setValue('permissions', next);
  };

  const onSubmit = (data: FormValues) => {
    createApiKey.mutate(
      {
        name: data.name,
        description: data.description || undefined,
        type: 'admin',
        permissions: data.permissions as ApiKeyPermission[] | undefined,
        expiresAt: data.expiresAt || null,
      },
      {
        onSuccess: (res) => {
          reset();
          onOpenChange(false);
          onCreated(res.key);
        },
        onError: () => {
          toast.error(t('apiKeys.createError'));
        },
      },
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <ApiKeyDialogHeader />
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="apikey-name">{t('apiKeys.name')}</Label>
            <Input id="apikey-name" placeholder={t('apiKeys.namePlaceholder')} {...register('name')} />
            {errors.name ? (
              <p className="text-destructive text-sm">{t('apiKeys.nameRequired')}</p>
            ) : null}
          </div>

          <ApiKeyDescriptionField register={register} />

          <ApiKeyPermissionsSection
            permissionsValue={permissionsValue}
            togglePermission={togglePermission}
          />

          <div className="space-y-2">
            <Label htmlFor="apikey-expires">{t('apiKeys.expiration')}</Label>
            <Input id="apikey-expires" type="date" {...register('expiresAt')} />
          </div>

          <ApiKeyFormActions isPending={createApiKey.isPending} onClose={() => onOpenChange(false)} />
        </form>
      </DialogContent>
    </Dialog>
  );
}

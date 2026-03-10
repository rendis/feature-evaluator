import { zodResolver } from '@hookform/resolvers/zod';
import { useState } from 'react';
import { useForm, type UseFormRegisterReturn } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { z } from 'zod';

import type { Workspace } from '@/api/workspaces';

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
import { buildNormalizedKeyFieldProps, resourceKeySchema, slugifyResourceKey } from '@/lib/resource-key';
import { useCreateWorkspace, useUpdateWorkspace } from '@/mutations/workspace-mutations';

const workspaceSchema = z.object({
  key: resourceKeySchema,
  name: z.string().min(1).max(256),
  description: z.string().max(1024).optional(),
});

type FormValues = z.infer<typeof workspaceSchema>;

interface WorkspaceFormDialogProps {
  workspace?: Workspace;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: (workspace: Workspace) => void;
}

function WorkspaceDialogHeader({ isEditing }: { isEditing: boolean }) {
  const { t } = useTranslation('workspaces');

  return (
    <DialogHeader>
      <DialogTitle>{isEditing ? t('form.editTitle') : t('form.createTitle')}</DialogTitle>
      <DialogDescription>
        {isEditing ? t('form.editDescription') : t('form.createDescription')}
      </DialogDescription>
    </DialogHeader>
  );
}

function WorkspaceFormFields({
  autoSlug,
  errors,
  isEditing,
  keyField,
  nameField,
  register,
  setValue,
}: {
  autoSlug: boolean;
  errors: ReturnType<typeof useForm<FormValues>>['formState']['errors'];
  isEditing: boolean;
  keyField: UseFormRegisterReturn<'key'>;
  nameField: UseFormRegisterReturn<'name'>;
  register: ReturnType<typeof useForm<FormValues>>['register'];
  setValue: ReturnType<typeof useForm<FormValues>>['setValue'];
}) {
  const { t } = useTranslation('workspaces');

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
    <>
      <div className="space-y-2">
        <Label htmlFor="ws-name">{t('fields.name')}</Label>
        <Input id="ws-name" {...nameField} onChange={handleNameChange} />
        {errors.name && <p className="text-destructive text-sm">{t('form.nameRequired')}</p>}
      </div>

      <div className="space-y-2">
        <Label htmlFor="ws-key">{t('fields.key')}</Label>
        <Input id="ws-key" {...keyField} disabled={isEditing} className="font-mono" />
        {errors.key && <p className="text-destructive text-sm">{t('form.keyInvalid')}</p>}
        <p className="text-muted-foreground text-xs">{t('form.keyPattern')}</p>
      </div>

      <div className="space-y-2">
        <Label htmlFor="ws-description">{t('fields.description')}</Label>
        <Input id="ws-description" {...register('description')} />
      </div>
    </>
  );
}

export function WorkspaceFormDialog({
  workspace,
  open,
  onOpenChange,
  onSuccess,
}: WorkspaceFormDialogProps) {
  const { t } = useTranslation('workspaces');
  const createWorkspace = useCreateWorkspace();
  const updateWorkspace = useUpdateWorkspace(workspace?.key ?? '');
  const isEditing = !!workspace;
  const [autoSlug, setAutoSlug] = useState(!isEditing);
  const { register, handleSubmit, setValue, reset, formState: { errors } } = useForm<FormValues>({
    resolver: zodResolver(workspaceSchema),
    defaultValues: {
      key: workspace?.key ?? '',
      name: workspace?.name ?? '',
      description: workspace?.description ?? '',
    },
  });
  const keyField = buildNormalizedKeyFieldProps({ name: 'key', onChangeNormalized: () => setAutoSlug(false), register, setValue });
  const nameField = register('name');
  const isPending = createWorkspace.isPending || updateWorkspace.isPending;
  useSubmissionLoadingModal(isPending, isEditing ? 'update' : 'create');
  const onSubmit = (data: FormValues) => {
    const handleSuccess = (savedWorkspace: Workspace) => {
      toast.success(t('form.success'));
      reset();
      onOpenChange(false);
      onSuccess?.(savedWorkspace);
    };
    const onError = () => toast.error(t('form.error'));

    if (isEditing) {
      updateWorkspace.mutate(
        { name: data.name, description: data.description },
        { onSuccess: handleSuccess, onError },
      );
    } else {
      createWorkspace.mutate(
        { key: data.key, name: data.name, description: data.description },
        { onSuccess: handleSuccess, onError },
      );
    }
  };
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <WorkspaceDialogHeader isEditing={isEditing} />
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <WorkspaceFormFields
            autoSlug={autoSlug}
            errors={errors}
            isEditing={isEditing}
            keyField={keyField}
            nameField={nameField}
            register={register}
            setValue={setValue}
          />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t('actions.cancel', { ns: 'common' })}
            </Button>
            <Button type="submit" disabled={isPending}>
              {t('actions.save', { ns: 'common' })}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

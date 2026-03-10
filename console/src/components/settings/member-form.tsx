import { zodResolver } from '@hookform/resolvers/zod';
import { useForm, useWatch } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { z } from 'zod';

import { RoleSelect } from './role-select';

import type { MemberRole } from '@/api/types';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useSubmissionLoadingModal } from '@/hooks/use-global-loading';
import { useCreateMember } from '@/mutations/member-mutations';

const createMemberSchema = z.object({
  email: z.string().min(1).email(),
  role: z.enum(['admin', 'editor', 'viewer']),
});

type FormValues = z.infer<typeof createMemberSchema>;

interface MemberFormProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function MemberForm({ open, onOpenChange }: MemberFormProps) {
  const { t } = useTranslation('settings');
  const createMember = useCreateMember();

  const {
    control,
    register,
    handleSubmit,
    setValue,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(createMemberSchema),
    defaultValues: { email: '', role: 'viewer' },
  });

  const roleValue = useWatch({ control, name: 'role' });
  useSubmissionLoadingModal(createMember.isPending, 'create');

  const onSubmit = (data: FormValues) => {
    createMember.mutate(data, {
      onSuccess: () => {
        toast.success(t('members.addForm.success'));
        reset();
        onOpenChange(false);
      },
      onError: () => {
        toast.error(t('members.addForm.error'));
      },
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('members.addForm.title')}</DialogTitle>
          <DialogDescription>{t('members.addForm.description', { defaultValue: 'Add a new team member by email and assign a role.' })}</DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="email">{t('members.email')}</Label>
            <Input
              id="email"
              type="email"
              placeholder={t('members.addForm.emailPlaceholder')}
              {...register('email')}
            />
            {errors.email ? (
              <p className="text-destructive text-sm">{t('members.addForm.emailInvalid')}</p>
            ) : null}
          </div>
          <div className="space-y-2">
            <Label>{t('members.role')}</Label>
            <RoleSelect
              value={roleValue as MemberRole}
              onValueChange={(v) => setValue('role', v as FormValues['role'])}
              excludeOwner
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {t('actions.cancel', { ns: 'common' })}
            </Button>
            <Button type="submit" disabled={createMember.isPending}>
              {t('members.addMember')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

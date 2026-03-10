import { Building2, GraduationCap, BookOpen } from 'lucide-react';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import type { TargetType } from '@/api/types';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useSubmissionLoadingModal } from '@/hooks/use-global-loading';
import { useActivatePack } from '@/mutations/pack-mutations';

interface ActivateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  packKey: string;
}

const TARGET_TYPES: { value: TargetType; icon: React.ComponentType<{ className?: string }> }[] = [
  { value: 'tenant', icon: Building2 },
  { value: 'campus', icon: GraduationCap },
  { value: 'program', icon: BookOpen },
];

function TargetTypeSelector({
  targetType,
  onSelect,
}: {
  targetType: TargetType;
  onSelect: (value: TargetType) => void;
}) {
  const { t } = useTranslation('packs');

  return (
    <div className="space-y-2">
      <Label>{t('activations.targetType')}</Label>
      <div className="inline-flex rounded-lg border">
        {TARGET_TYPES.map(({ value, icon: Icon }) => (
          <button
            key={value}
            type="button"
            onClick={() => onSelect(value)}
            className={`flex items-center gap-2 px-4 py-2 text-sm font-medium transition-colors first:rounded-l-lg last:rounded-r-lg ${
              targetType === value ? 'bg-primary text-primary-foreground' : 'hover:bg-muted'
            }`}
          >
            <Icon className="h-4 w-4" />
            {t(`targetTypes.${value}`)}
          </button>
        ))}
      </div>
    </div>
  );
}

function ExpirationField({
  expiresAt,
  hasExpiry,
  onToggle,
  onValueChange,
}: {
  expiresAt: string;
  hasExpiry: boolean;
  onToggle: (checked: boolean) => void;
  onValueChange: (value: string) => void;
}) {
  const { t } = useTranslation('packs');

  return (
    <div className="space-y-2">
      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={hasExpiry}
          onChange={(event) => onToggle(event.target.checked)}
          className="accent-primary"
        />
        {t('activations.setExpiration')}
      </label>
      {hasExpiry ? (
        <Input
          type="datetime-local"
          value={expiresAt}
          onChange={(event) => onValueChange(event.target.value)}
        />
      ) : null}
    </div>
  );
}

export function ActivateDialog({ open, onOpenChange, packKey }: ActivateDialogProps) {
  const { t } = useTranslation('packs');
  const activate = useActivatePack();
  const [targetType, setTargetType] = useState<TargetType>('tenant');
  const [targetId, setTargetId] = useState('');
  const [hasExpiry, setHasExpiry] = useState(false);
  const [expiresAt, setExpiresAt] = useState('');
  useSubmissionLoadingModal(activate.isPending, 'create');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!targetId.trim()) return;

    activate.mutate(
      {
        key: packKey,
        data: {
          targetType,
          targetId: targetId.trim(),
          expiresAt: hasExpiry && expiresAt ? new Date(expiresAt).toISOString() : null,
        },
      },
      {
        onSuccess: () => {
          resetForm();
          onOpenChange(false);
          toast.success(t('activations.activateSuccess'));
        },
        onError: () => toast.error(t('activations.activateError')),
      },
    );
  };

  const resetForm = () => {
    setTargetType('tenant');
    setTargetId('');
    setHasExpiry(false);
    setExpiresAt('');
  };

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) resetForm();
    onOpenChange(nextOpen);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('activations.activateDialog.title')}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <TargetTypeSelector targetType={targetType} onSelect={setTargetType} />

          <div className="space-y-2">
            <Label htmlFor="targetId">
              {t(`activations.targetIdLabel.${targetType}`)}
            </Label>
            <Input
              id="targetId"
              value={targetId}
              onChange={(e) => setTargetId(e.target.value)}
              placeholder={t('activations.targetIdPlaceholder')}
              className="font-mono"
            />
          </div>

          <ExpirationField
            expiresAt={expiresAt}
            hasExpiry={hasExpiry}
            onToggle={setHasExpiry}
            onValueChange={setExpiresAt}
          />

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
              {t('cancel', { ns: 'common', defaultValue: 'Cancel' })}
            </Button>
            <Button type="submit" disabled={!targetId.trim() || activate.isPending}>
              {t('activations.activate')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

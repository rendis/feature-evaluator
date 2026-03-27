import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import type { Rule } from '@/api/types';

import { Switch } from '@/components/ui/switch';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { usePermissions } from '@/hooks/use-permissions';
import { useToggleRule } from '@/mutations/rule-mutations';

interface RuleToggleProps {
  featureKey: string;
  rule: Rule;
}

export function RuleToggle({ featureKey, rule }: RuleToggleProps) {
  const { t } = useTranslation('rules');
  const { can } = usePermissions();
  const toggle = useToggleRule(featureKey);
  const canWrite = can('features.write');
  const disabled = toggle.isPending || !canWrite;
  const ariaLabel = t('toggle.ariaLabel', {
    name: rule.name,
    defaultValue: 'Activar o desactivar {{name}}',
  });

  const handleToggle = () => {
    const newEnabled = !rule.enabled;
    toggle.mutate(
      { rule },
      {
        onSuccess: () => {
          toast.success(newEnabled ? t('toggle.enableSuccess') : t('toggle.disableSuccess'));
        },
        onError: () => toast.error(t('toggle.error')),
      },
    );
  };

  const control = (
    <Switch
      checked={rule.enabled}
      onCheckedChange={handleToggle}
      disabled={disabled}
      aria-label={ariaLabel}
    />
  );

  if (canWrite) {
    return control;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span tabIndex={0}>{control}</span>
      </TooltipTrigger>
      <TooltipContent>Sin permisos</TooltipContent>
    </Tooltip>
  );
}

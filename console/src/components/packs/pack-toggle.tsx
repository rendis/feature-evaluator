import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { Switch } from '@/components/ui/switch';
import { useTogglePack } from '@/mutations/pack-mutations';

interface PackToggleProps {
  packKey: string;
  enabled: boolean;
}

export function PackToggle({ packKey, enabled }: PackToggleProps) {
  const { t } = useTranslation('packs');
  const toggle = useTogglePack();

  const handleToggle = () => {
    const newEnabled = !enabled;
    toggle.mutate(
      { key: packKey, enabled: newEnabled },
      {
        onSuccess: () => {
          toast.success(newEnabled ? t('toggle.enableSuccess') : t('toggle.disableSuccess'));
        },
        onError: () => toast.error(t('toggle.error')),
      },
    );
  };

  return <Switch checked={enabled} onCheckedChange={handleToggle} disabled={toggle.isPending} />;
}

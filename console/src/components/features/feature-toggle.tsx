import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';

import { Switch } from '@/components/ui/switch';
import { useToggleFeature } from '@/mutations/feature-mutations';

interface FeatureToggleProps {
  featureKey: string;
  enabled: boolean;
}

export function FeatureToggle({ featureKey, enabled }: FeatureToggleProps) {
  const { t } = useTranslation('features');
  const toggle = useToggleFeature();

  const handleToggle = () => {
    const newEnabled = !enabled;
    toggle.mutate(
      { key: featureKey, enabled: newEnabled },
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

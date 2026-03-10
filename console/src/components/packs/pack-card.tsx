import { Link } from '@tanstack/react-router';
import { useTranslation } from 'react-i18next';

import { PackToggle } from './pack-toggle';

import type { Pack } from '@/api/types';

import { Badge } from '@/components/ui/badge';

interface PackCardProps {
  pack: Pack;
}

export function PackCard({ pack }: PackCardProps) {
  const { t } = useTranslation('packs');

  return (
    <div className="rounded-lg border p-4">
      <div className="flex items-start justify-between">
        <div>
          <Link
            to="/settings/packs/$packKey"
            params={{ packKey: pack.key }}
            className="hover:text-primary font-medium transition-colors"
          >
            {pack.name}
          </Link>
          <p className="text-muted-foreground mt-1 font-mono text-xs">{pack.key}</p>
        </div>
        <PackToggle packKey={pack.key} enabled={pack.enabled} />
      </div>
      {pack.description ? (
        <p className="text-muted-foreground mt-2 line-clamp-2 text-sm">{pack.description}</p>
      ) : null}
      <div className="mt-3 flex flex-wrap items-center gap-2">
        <Badge variant="secondary">
          {pack.featureKeys.length} {t('featureCount')}
        </Badge>
      </div>
    </div>
  );
}

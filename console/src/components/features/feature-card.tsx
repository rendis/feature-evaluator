import { Link } from '@tanstack/react-router';
import { Package } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { FeatureStatusBadge } from './feature-status-badge';
import { FeatureToggle } from './feature-toggle';

import type { FeatureListItem } from '@/api/types';

import { TagBadge } from '@/components/shared/tag-badge';
import { Badge } from '@/components/ui/badge';

interface FeatureCardProps {
  feature: FeatureListItem;
}

export function FeatureCard({ feature }: FeatureCardProps) {
  const { t } = useTranslation('features');

  return (
    <div className="rounded-lg border p-4">
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-2">
          <Link
            to="/features/$featureKey"
            params={{ featureKey: feature.key }}
            className="hover:text-primary font-medium transition-colors"
          >
            {feature.name}
          </Link>
          <FeatureStatusBadge feature={feature} />
        </div>
        <FeatureToggle featureKey={feature.key} enabled={feature.enabled} />
      </div>
      <p className="text-muted-foreground mt-1 text-xs font-mono">{feature.key}</p>
      {feature.description ? (
        <p className="text-muted-foreground mt-2 line-clamp-2 text-sm">{feature.description}</p>
      ) : null}
      <div className="mt-3 flex flex-wrap items-center gap-2">
        <Badge variant="secondary">{t(`valueTypes.${feature.valueType}`)}</Badge>
        <span className="text-muted-foreground text-xs">
          {feature.ruleCount ?? ('rules' in feature ? (feature.rules?.length ?? 0) : 0)}{' '}
          {(feature.ruleCount ?? ('rules' in feature ? (feature.rules?.length ?? 0) : 0)) === 1
            ? 'rule'
            : 'rules'}
        </span>
        {feature.tags.map((tag) => (
          <TagBadge key={tag.key} tag={tag} size="sm" />
        ))}
        {feature.environments && feature.environments.length > 0
          ? feature.environments.map((env) => (
              <Badge key={env} variant="secondary" className="text-xs">
                {env}
              </Badge>
            ))
          : null}
        {('packCount' in feature ? feature.packCount : (feature.packs?.length ?? 0)) > 0 ? (
          <span className="text-muted-foreground flex items-center gap-1 text-xs">
            <Package className="h-3 w-3" />
            {'packCount' in feature ? feature.packCount : (feature.packs?.length ?? 0)}
          </span>
        ) : null}
      </div>
    </div>
  );
}

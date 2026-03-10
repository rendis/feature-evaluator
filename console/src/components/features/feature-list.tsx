import { Link } from '@tanstack/react-router';
import { useTranslation } from 'react-i18next';

import { FeatureCard } from './feature-card';
import { FeatureStatusBadge } from './feature-status-badge';
import { FeatureToggle } from './feature-toggle';

import type { FeatureListItem } from '@/api/types';

import { TagBadge } from '@/components/shared/tag-badge';
import { Badge } from '@/components/ui/badge';
import { useMobile } from '@/hooks/use-mobile';

interface FeatureListProps {
  features: FeatureListItem[];
}

export function FeatureList({ features }: FeatureListProps) {
  const isMobile = useMobile();

  if (isMobile) {
    return (
      <div className="space-y-3">
        {features.map((f) => (
          <FeatureCard key={f.key} feature={f} />
        ))}
      </div>
    );
  }

  return <FeatureDesktopTable features={features} />;
}

function FeatureDesktopTable({ features }: { features: FeatureListItem[] }) {
  const { t } = useTranslation('features');

  return (
    <div className="rounded-md border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/50">
            <th className="px-4 py-3 text-left font-medium">{t('fields.name')}</th>
            <th className="px-4 py-3 text-left font-medium">{t('fields.key')}</th>
            <th className="px-4 py-3 text-left font-medium">{t('fields.valueType')}</th>
            <th className="px-4 py-3 text-left font-medium">{t('fields.rules')}</th>
            <th className="px-4 py-3 text-left font-medium">{t('fields.tags')}</th>
            <th className="px-4 py-3 text-left font-medium">{t('fields.environments')}</th>
            <th className="hidden px-4 py-3 text-left font-medium md:table-cell">
              {t('fields.packs', { defaultValue: 'Packs' })}
            </th>
            <th className="px-4 py-3 text-center font-medium">{t('fields.enabled')}</th>
          </tr>
        </thead>
        <tbody>
          {features.map((f) => (
            <tr key={f.key} className="border-b last:border-0">
              <td className="px-4 py-3">
                <div className="flex items-center gap-2">
                  <Link
                    to="/features/$featureKey"
                    params={{ featureKey: f.key }}
                    className="hover:text-primary font-medium transition-colors"
                  >
                    {f.name}
                  </Link>
                  <FeatureStatusBadge feature={f} />
                </div>
              </td>
              <td className="text-muted-foreground px-4 py-3 font-mono text-xs">{f.key}</td>
              <td className="px-4 py-3">
                <Badge variant="secondary">{t(`valueTypes.${f.valueType}`)}</Badge>
              </td>
              <td className="text-muted-foreground px-4 py-3">
                {f.ruleCount ?? ('rules' in f ? (f.rules?.length ?? 0) : 0)}
              </td>
              <td className="px-4 py-3">
                <div className="flex flex-wrap gap-1">
                  {f.tags.map((tag) => (
                    <TagBadge key={tag.key} tag={tag} size="sm" />
                  ))}
                </div>
              </td>
              <td className="px-4 py-3">
                <div className="flex flex-wrap gap-1">
                  {f.environments && f.environments.length > 0 ? (
                    f.environments.map((env) => (
                      <Badge key={env} variant="secondary" className="text-xs">
                        {env}
                      </Badge>
                    ))
                  ) : (
                    <span className="text-muted-foreground text-xs">{t('environments.all')}</span>
                  )}
                </div>
              </td>
              <td className="hidden px-4 py-3 md:table-cell">
                {('packCount' in f ? f.packCount : (f.packs?.length ?? 0)) > 0 ? (
                  <Badge variant="secondary" className="text-xs">
                    {'packCount' in f ? f.packCount : (f.packs?.length ?? 0)}
                  </Badge>
                ) : null}
              </td>
              <td className="px-4 py-3 text-center">
                <FeatureToggle featureKey={f.key} enabled={f.enabled} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

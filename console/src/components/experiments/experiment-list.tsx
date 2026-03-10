import { Link } from '@tanstack/react-router';
import { useTranslation } from 'react-i18next';

import type { Experiment } from '@/api/types';

import { ExperimentStatusBadge } from '@/components/experiments/experiment-status-badge';
import { Badge } from '@/components/ui/badge';
import { useLocaleFormatters } from '@/hooks/use-locale-formatters';

interface ExperimentListProps {
  experiments: Experiment[];
}

export function ExperimentList({ experiments }: ExperimentListProps) {
  const { t } = useTranslation('experiments');
  const { formatDate } = useLocaleFormatters();

  return (
    <div className="space-y-2">
      {experiments.map((exp) => (
        <Link
          key={exp.id}
          to="/experiments/$experimentId"
          params={{ experimentId: exp.id }}
          className="flex items-center justify-between rounded-md border p-4 transition-colors hover:bg-accent"
        >
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span className="font-medium">{exp.name}</span>
              <ExperimentStatusBadge status={exp.status} />
              {exp.winnerKey && (
                <Badge variant="success">
                  {t('winner')}: {exp.winnerKey}
                </Badge>
              )}
            </div>
            <div className="text-muted-foreground mt-1 flex items-center gap-3 text-sm">
              <span>
                {t('featureKey')}: <span className="font-mono">{exp.featureKey}</span>
              </span>
              <span>
                {exp.variants.length} {t('variantsCount')}
              </span>
            </div>
          </div>
          <span className="text-muted-foreground text-xs">
            {formatDate(exp.createdAt)}
          </span>
        </Link>
      ))}
    </div>
  );
}

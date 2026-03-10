import { useSuspenseQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';

import { PackActivationsTab } from '@/components/packs/pack-activations-tab';
import { PackDetailHeader } from '@/components/packs/pack-detail-header';
import { PackFeaturesTab } from '@/components/packs/pack-features-tab';
import { PackForm } from '@/components/packs/pack-form';
import { ApiErrorState } from '@/components/shared/error-state';
import { LoadingSkeleton } from '@/components/shared/loading-skeleton';
import { packQueries } from '@/queries/pack-queries';

export const Route = createFileRoute('/_authenticated/settings/packs/$packKey/')({
  component: PackDetailPage,
  pendingComponent: () => <LoadingSkeleton rows={8} />,
  errorComponent: ({ error }) => (
    <ApiErrorState error={error} />
  ),
});

type TabValue = 'features' | 'activations';

function PackDetailPage() {
  const { packKey } = Route.useParams();
  const { t } = useTranslation('packs');
  const { data: pack } = useSuspenseQuery(packQueries.detail(packKey));
  const [activeTab, setActiveTab] = useState<TabValue>('features');
  const [editOpen, setEditOpen] = useState(false);

  return (
    <div className="space-y-6">
      <PackDetailHeader pack={pack} onEdit={() => setEditOpen(true)} />

      <div className="border-b">
        <nav className="flex gap-4">
          <button
            type="button"
            onClick={() => setActiveTab('features')}
            className={`border-b-2 px-1 pb-3 text-sm font-medium transition-colors ${
              activeTab === 'features'
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {t('tabs.features')}
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('activations')}
            className={`border-b-2 px-1 pb-3 text-sm font-medium transition-colors ${
              activeTab === 'activations'
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {t('tabs.activations')}
          </button>
        </nav>
      </div>

      {activeTab === 'features' ? (
        <PackFeaturesTab pack={pack} />
      ) : (
        <PackActivationsTab pack={pack} />
      )}

      <PackForm
        pack={pack}
        open={editOpen}
        onOpenChange={setEditOpen}
      />
    </div>
  );
}

import * as Collapsible from '@radix-ui/react-collapsible';
import { useQueryClient } from '@tanstack/react-query';
import { BarChart3, ChevronDown, RefreshCw } from 'lucide-react';
import { Suspense, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { EnvironmentChart, EnvironmentChartSkeleton } from './environment-chart';
import { MetricsOverview, MetricsOverviewSkeleton } from './metrics-overview';
import { ReasonBreakdown, ReasonBreakdownSkeleton } from './reason-breakdown';
import { TopFeaturesChart, TopFeaturesChartSkeleton } from './top-features-chart';

import { Button } from '@/components/ui/button';
import { dashboardQueries } from '@/queries/dashboard-queries';

export function MetricsSection() {
  const { t } = useTranslation('dashboard');
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const handleRefresh = async () => {
    setRefreshing(true);
    await queryClient.invalidateQueries({ queryKey: [...dashboardQueries.all()] });
    setRefreshing(false);
  };

  return (
    <Collapsible.Root open={open} onOpenChange={setOpen}>
      <div className="rounded-lg border">
        <div className="flex items-center justify-between border-b px-4 py-3">
          <Collapsible.Trigger asChild>
            <button
              type="button"
              className="flex items-center gap-2 text-sm font-semibold hover:opacity-80"
            >
              <BarChart3 className="h-4 w-4" />
              {t('metrics.title')}
              <ChevronDown
                className={`h-4 w-4 transition-transform ${open ? '' : '-rotate-90'}`}
              />
            </button>
          </Collapsible.Trigger>
          <Button
            variant="ghost"
            size="sm"
            onClick={handleRefresh}
            disabled={refreshing}
          >
            <RefreshCw className={`h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />
          </Button>
        </div>
        <Collapsible.Content>
          <div className="space-y-6 p-4">
            <Suspense fallback={<MetricsOverviewSkeleton />}>
              <MetricsOverview />
            </Suspense>
            <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
              <Suspense fallback={<TopFeaturesChartSkeleton />}>
                <TopFeaturesChart />
              </Suspense>
              <Suspense fallback={<ReasonBreakdownSkeleton />}>
                <ReasonBreakdown />
              </Suspense>
            </div>
            <Suspense fallback={<EnvironmentChartSkeleton />}>
              <EnvironmentChart />
            </Suspense>
          </div>
        </Collapsible.Content>
      </div>
    </Collapsible.Root>
  );
}

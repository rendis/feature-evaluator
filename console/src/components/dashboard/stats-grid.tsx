import { useSuspenseQuery } from '@tanstack/react-query';
import { Layers, ToggleLeft, Users, UsersRound } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { StatCard } from './stat-card';

import { dashboardQueries } from '@/queries/dashboard-queries';

export function StatsGrid() {
  const { t } = useTranslation('dashboard');
  const { data } = useSuspenseQuery(dashboardQueries.stats());

  const cards = [
    { title: t('stats.totalFeatures'), value: data.totalFeatures, icon: <ToggleLeft className="h-5 w-5" /> },
    { title: t('stats.activeFeatures'), value: data.activeFeatures, icon: <Layers className="h-5 w-5" /> },
    { title: t('stats.totalSegments'), value: data.totalSegments, icon: <UsersRound className="h-5 w-5" /> },
    { title: t('stats.totalMembers'), value: data.totalSegmentMembers, icon: <Users className="h-5 w-5" /> },
  ];

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {cards.map((card) => (
        <StatCard key={card.title} {...card} />
      ))}
    </div>
  );
}

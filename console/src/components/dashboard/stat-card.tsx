import type { ReactNode } from 'react';

import { useLocaleFormatters } from '@/hooks/use-locale-formatters';
import { Skeleton } from '@/components/ui/skeleton';

interface StatCardProps {
  title: string;
  value: number;
  icon: ReactNode;
  loading?: boolean;
}

export function StatCard({ title, value, icon, loading }: StatCardProps) {
  const { formatNumber } = useLocaleFormatters();

  return (
    <div className="rounded-lg border bg-card p-6 shadow-sm">
      <div className="flex items-center justify-between">
        <p className="text-muted-foreground text-sm font-medium">{title}</p>
        <div className="text-muted-foreground">{icon}</div>
      </div>
      {loading ? (
        <Skeleton className="mt-2 h-8 w-20" />
      ) : (
        <p className="mt-2 text-3xl font-bold tracking-tight">
          {formatNumber(value)}
        </p>
      )}
    </div>
  );
}

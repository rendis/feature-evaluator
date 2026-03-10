import { ArrowLeft } from 'lucide-react';

import type { ReactNode } from 'react';

import { Button } from '@/components/ui/button';

interface PageHeaderProps {
  title: string;
  description?: string;
  actions?: ReactNode;
  onBack?: () => void;
}

export function PageHeader({ title, description, actions, onBack }: PageHeaderProps) {
  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex items-center gap-3">
        {onBack ? (
          <Button variant="ghost" size="icon" onClick={onBack} className="shrink-0">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        ) : null}
        <div>
          <h1 className="text-content-strong text-2xl font-bold tracking-tight">{title}</h1>
          {description ? <p className="text-content-muted">{description}</p> : null}
        </div>
      </div>
      {actions ? <div className="flex flex-wrap items-center gap-2">{actions}</div> : null}
    </div>
  );
}

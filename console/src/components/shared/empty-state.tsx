import type { ReactNode } from 'react';

interface EmptyStateProps {
  icon?: ReactNode;
  title: string;
  description?: string;
  action?: ReactNode;
}

export function EmptyState({ icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="fe-empty-surface flex flex-col items-center justify-center px-6 py-16 text-center">
      {icon ? <div className="text-content-muted mb-4">{icon}</div> : null}
      <h3 className="text-content-strong text-lg font-semibold">{title}</h3>
      {description ? <p className="text-content-muted mt-1 max-w-sm">{description}</p> : null}
      {action ? <div className="mt-4">{action}</div> : null}
    </div>
  );
}

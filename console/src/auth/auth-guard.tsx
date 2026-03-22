import { Navigate } from '@tanstack/react-router';

import type { ReactNode } from 'react';

import { FullScreenSpinner } from '@/components/shared/full-screen-spinner';
import { useAuth } from '@/hooks/use-auth';

export function AuthGuard({ children }: { children: ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return <FullScreenSpinner />;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" />;
  }

  return <>{children}</>;
}

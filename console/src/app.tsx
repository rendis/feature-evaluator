import { QueryClientProvider } from '@tanstack/react-query';
import { RouterProvider, createRouter } from '@tanstack/react-router';
import { Suspense } from 'react';

import { AuthProvider } from '@/auth/auth-provider';
import { FullScreenSpinner } from '@/components/shared/full-screen-spinner';
import { GlobalLoadingProvider } from '@/components/shared/global-loading-provider';
import { Toaster } from '@/components/ui/sonner';
import { TooltipProvider } from '@/components/ui/tooltip';
import { queryClient } from '@/config/query-client';
import { routeTree } from '@/routeTree.gen';

const basepath = import.meta.env.BASE_URL.replace(/\/+$/, '') || '/';
const router = createRouter({ routeTree, basepath });

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}

export function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <GlobalLoadingProvider>
          <TooltipProvider>
            <Suspense fallback={<FullScreenSpinner />}>
              <RouterProvider router={router} />
            </Suspense>
            <Toaster />
          </TooltipProvider>
        </GlobalLoadingProvider>
      </AuthProvider>
    </QueryClientProvider>
  );
}

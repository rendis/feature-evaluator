import { QueryClientProvider } from '@tanstack/react-query';
import { RouterProvider, createRouter } from '@tanstack/react-router';
import { Suspense } from 'react';

import { AuthProvider } from '@/auth/auth-provider';
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
            <Suspense
              fallback={
                <div className="flex h-screen items-center justify-center">
                  <div className="border-primary h-8 w-8 animate-spin rounded-full border-4 border-t-transparent" />
                </div>
              }
            >
              <RouterProvider router={router} />
            </Suspense>
            <Toaster />
          </TooltipProvider>
        </GlobalLoadingProvider>
      </AuthProvider>
    </QueryClientProvider>
  );
}

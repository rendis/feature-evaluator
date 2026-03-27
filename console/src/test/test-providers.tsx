import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { TooltipProvider } from '@radix-ui/react-tooltip';
import { createInstance } from 'i18next';
import { type ReactNode } from 'react';
import { I18nextProvider, initReactI18next } from 'react-i18next';

import { GlobalLoadingProvider } from '@/components/shared/global-loading-provider';
import { getLocaleNamespaces, buildTestLocaleResources } from '@/test/locale-resources';

export interface TestProviderProps {
  children: ReactNode;
  locale?: 'es' | 'en';
  namespaces?: string[];
  useRealTranslations?: boolean;
}

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
}

function createTestI18n({
  locale = 'es',
  namespaces = ['common'],
  useRealTranslations = false,
}: Omit<TestProviderProps, 'children'>) {
  const i18n = createInstance();
  const resolvedNamespaces =
    useRealTranslations && namespaces.length === 0 ? getLocaleNamespaces() : namespaces;
  const resources = useRealTranslations
    ? buildTestLocaleResources(resolvedNamespaces)
    : {
        es: Object.fromEntries(resolvedNamespaces.map((namespace) => [namespace, {}])),
        en: Object.fromEntries(resolvedNamespaces.map((namespace) => [namespace, {}])),
      };

  void i18n.use(initReactI18next).init({
    lng: locale,
    fallbackLng: 'es',
    ns: resolvedNamespaces,
    defaultNS: resolvedNamespaces[0] ?? 'common',
    resources,
    interpolation: { escapeValue: false },
    initImmediate: false,
  });

  return i18n;
}

export function TestProviders({
  children,
  locale = 'es',
  namespaces = ['common'],
  useRealTranslations = false,
}: TestProviderProps) {
  const queryClient = createTestQueryClient();
  const i18n = createTestI18n({ locale, namespaces, useRealTranslations });

  return (
    <QueryClientProvider client={queryClient}>
      <I18nextProvider i18n={i18n}>
        <TooltipProvider>
          <GlobalLoadingProvider>{children}</GlobalLoadingProvider>
        </TooltipProvider>
      </I18nextProvider>
    </QueryClientProvider>
  );
}

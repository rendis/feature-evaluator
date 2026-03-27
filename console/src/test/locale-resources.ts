import type { Resource } from 'i18next';

interface LocaleModule {
  default: unknown;
}

const localeModules = import.meta.glob('../../public/assets/locales/*/*.json', {
  eager: true,
}) as Record<string, LocaleModule>;

function parseLocaleModulePath(path: string) {
  const match = path.match(/\/locales\/([^/]+)\/([^/]+)\.json$/);
  if (!match) {
    return null;
  }

  return {
    locale: match[1] as 'es' | 'en',
    namespace: match[2],
  };
}

export function getLocaleCatalogs() {
  const catalogs: Record<string, Record<string, unknown>> = {};

  for (const [path, module] of Object.entries(localeModules)) {
    const parsed = parseLocaleModulePath(path);
    if (!parsed) {
      continue;
    }

    catalogs[parsed.locale] ??= {};
    catalogs[parsed.locale][parsed.namespace] = module.default;
  }

  return catalogs;
}

export function getLocaleNamespaces() {
  return Object.keys(getLocaleCatalogs().es ?? {}).sort();
}

export function buildTestLocaleResources(namespaces: string[]): Resource {
  const catalogs = getLocaleCatalogs();
  const resources: Resource = {};

  for (const locale of ['es', 'en'] as const) {
    resources[locale] = {};

    for (const namespace of namespaces) {
      resources[locale][namespace] =
        (catalogs[locale]?.[namespace] as Record<string, unknown> | undefined) ?? {};
    }
  }

  return resources;
}

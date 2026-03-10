export type AppLocale = 'es' | 'en';

export function normalizeAppLocale(locale?: string | null): AppLocale {
  if (!locale) {
    return 'es';
  }

  return locale.toLowerCase().startsWith('en') ? 'en' : 'es';
}

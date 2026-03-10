import i18n from 'i18next';
import HttpBackend from 'i18next-http-backend';
import { initReactI18next } from 'react-i18next';

import { normalizeAppLocale } from '@/lib/locale';

const storedLocale = (() => {
  try {
    const raw = localStorage.getItem('fe-locale');
    if (raw) {
      const parsed = JSON.parse(raw) as { state?: { locale?: string } };
      return normalizeAppLocale(parsed.state?.locale);
    }
  } catch {
    /* empty */
  }
  return 'es';
})();

await i18n
  .use(HttpBackend)
  .use(initReactI18next)
  .init({
    lng: storedLocale,
    fallbackLng: 'es',
    ns: ['common'],
    defaultNS: 'common',
    backend: {
      loadPath: `${import.meta.env.BASE_URL}assets/locales/{{lng}}/{{ns}}.json`,
    },
    interpolation: {
      escapeValue: false,
    },
    react: {
      useSuspense: true,
    },
  });

export default i18n;

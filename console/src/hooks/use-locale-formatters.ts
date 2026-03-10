import { useTranslation } from 'react-i18next';

import {
  formatDate,
  formatDateTime,
  formatNumber,
  formatRelativeTime,
  formatTime,
} from '@/lib/locale-format';
import { normalizeAppLocale } from '@/lib/locale';

export function useAppLocale() {
  const { i18n } = useTranslation();
  return normalizeAppLocale(i18n.resolvedLanguage ?? i18n.language);
}

export function useLocaleFormatters() {
  const locale = useAppLocale();

  return {
    locale,
    formatDate: (value: Date | number | string | null | undefined, options?: Intl.DateTimeFormatOptions) =>
      formatDate(value, locale, options),
    formatDateTime: (value: Date | number | string | null | undefined, options?: Intl.DateTimeFormatOptions) =>
      formatDateTime(value, locale, options),
    formatNumber: (value: number, options?: Intl.NumberFormatOptions) =>
      formatNumber(value, locale, options),
    formatRelativeTime: (value: Date | number | string | null | undefined, now?: Date) =>
      formatRelativeTime(value, locale, now),
    formatTime: (value: Date | number | string | null | undefined, options?: Intl.DateTimeFormatOptions) =>
      formatTime(value, locale, options),
  };
}

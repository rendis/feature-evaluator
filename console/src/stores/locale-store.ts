import { create } from 'zustand';
import { persist } from 'zustand/middleware';

import type { AppLocale } from '@/lib/locale';

interface LocaleState {
  locale: AppLocale;
  setLocale: (locale: AppLocale) => void;
}

export const useLocaleStore = create<LocaleState>()(
  persist(
    (set) => ({
      locale: 'es',
      setLocale: (locale) => set({ locale }),
    }),
    { name: 'fe-locale' },
  ),
);

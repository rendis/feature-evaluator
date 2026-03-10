import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type Theme = 'light' | 'dark' | 'system';

interface ThemeState {
  theme: Theme;
  setTheme: (theme: Theme) => void;
}

const THEME_STORAGE_KEY = 'fe-theme';
const THEME_MEDIA_QUERY = '(prefers-color-scheme: dark)';

let mediaCleanup: (() => void) | null = null;

function isTheme(value: unknown): value is Theme {
  return value === 'light' || value === 'dark' || value === 'system';
}

export function getStoredTheme(storageValue: string | null): Theme {
  if (!storageValue) {
    return 'system';
  }

  try {
    const parsed = JSON.parse(storageValue) as { state?: { theme?: unknown } };
    return isTheme(parsed.state?.theme) ? parsed.state.theme : 'system';
  } catch {
    return 'system';
  }
}

export function getSystemTheme() {
  return window.matchMedia(THEME_MEDIA_QUERY).matches ? 'dark' : 'light';
}

export function resolveTheme(theme: Theme) {
  return theme === 'system' ? getSystemTheme() : theme;
}

export function applyTheme(theme: Theme) {
  const resolvedTheme = resolveTheme(theme);
  const root = document.documentElement;

  root.classList.remove('light', 'dark');
  root.classList.add(resolvedTheme);
  root.style.colorScheme = resolvedTheme;

  return resolvedTheme;
}

function detachSystemThemeListener() {
  if (mediaCleanup) {
    mediaCleanup();
    mediaCleanup = null;
  }
}

export function syncSystemThemeListener(theme: Theme) {
  detachSystemThemeListener();

  if (theme !== 'system') {
    return;
  }

  const mediaQuery = window.matchMedia(THEME_MEDIA_QUERY);
  const listener = () => {
    applyTheme('system');
  };

  mediaQuery.addEventListener('change', listener);
  mediaCleanup = () => mediaQuery.removeEventListener('change', listener);
}

export function readInitialTheme(): Theme {
  if (typeof window === 'undefined') {
    return 'system';
  }

  return getStoredTheme(window.localStorage.getItem(THEME_STORAGE_KEY));
}

const initialTheme = readInitialTheme();

if (typeof window !== 'undefined') {
  syncSystemThemeListener(initialTheme);
  applyTheme(initialTheme);
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      theme: initialTheme,
      setTheme: (theme) => {
        syncSystemThemeListener(theme);
        applyTheme(theme);
        set({ theme });
      },
    }),
    {
      name: THEME_STORAGE_KEY,
      onRehydrateStorage: () => (state) => {
        const nextTheme = state?.theme ?? initialTheme;
        syncSystemThemeListener(nextTheme);
        applyTheme(nextTheme);
      },
    },
  ),
);

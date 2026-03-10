import { beforeEach, describe, expect, it, vi } from 'vitest';

type ThemeModule = typeof import('./theme-store');

const STORAGE_KEY = 'fe-theme';

interface MatchMediaController {
  mediaQuery: MediaQueryList;
  setMatches: (matches: boolean) => void;
  trigger: (matches: boolean) => void;
}

function createMatchMediaController(initialMatches: boolean): MatchMediaController {
  const listeners = new Set<(event: MediaQueryListEvent) => void>();
  const mediaQuery = {
    matches: initialMatches,
    media: '(prefers-color-scheme: dark)',
    onchange: null,
    addEventListener: vi.fn((_type: string, listener: (event: MediaQueryListEvent) => void) => {
      listeners.add(listener);
    }),
    removeEventListener: vi.fn((_type: string, listener: (event: MediaQueryListEvent) => void) => {
      listeners.delete(listener);
    }),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  } as unknown as MediaQueryList;

  return {
    mediaQuery,
    setMatches(matches: boolean) {
      Object.defineProperty(mediaQuery, 'matches', {
        configurable: true,
        value: matches,
      });
    },
    trigger(matches: boolean) {
      this.setMatches(matches);
      const event = { matches, media: mediaQuery.media } as MediaQueryListEvent;
      listeners.forEach((listener) => listener(event));
    },
  };
}

async function loadThemeStore(
  initialMatches = false,
  storedTheme?: 'light' | 'dark' | 'system',
): Promise<{ controller: MatchMediaController; module: ThemeModule }> {
  vi.resetModules();
  document.documentElement.className = '';
  document.documentElement.style.colorScheme = '';
  window.localStorage.removeItem(STORAGE_KEY);

  if (storedTheme) {
    window.localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ state: { theme: storedTheme }, version: 0 }),
    );
  }

  const controller = createMatchMediaController(initialMatches);
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    writable: true,
    value: vi.fn(() => controller.mediaQuery),
  });

  return {
    controller,
    module: await import('./theme-store'),
  };
}

describe('theme store', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    window.localStorage.removeItem(STORAGE_KEY);
    document.documentElement.className = '';
    document.documentElement.style.colorScheme = '';
  });

  it('parses persisted theme values safely', async () => {
    const { module } = await loadThemeStore();

    expect(module.getStoredTheme(null)).toBe('system');
    expect(module.getStoredTheme('{"state":{"theme":"dark"}}')).toBe('dark');
    expect(module.getStoredTheme('{"state":{"theme":"invalid"}}')).toBe('system');
    expect(module.getStoredTheme('not-json')).toBe('system');
  });

  it('applies the stored theme on bootstrap and keeps html classes in sync', async () => {
    const { module } = await loadThemeStore(false, 'dark');

    expect(document.documentElement.classList.contains('dark')).toBe(true);
    expect(document.documentElement.style.colorScheme).toBe('dark');

    module.useThemeStore.getState().setTheme('light');

    expect(document.documentElement.classList.contains('light')).toBe(true);
    expect(document.documentElement.classList.contains('dark')).toBe(false);
    expect(document.documentElement.style.colorScheme).toBe('light');
    expect(window.localStorage.getItem(STORAGE_KEY)).toContain('"theme":"light"');
  });

  it('reacts to system color-scheme changes while the store theme is system', async () => {
    const { controller, module } = await loadThemeStore(false, 'system');

    expect(document.documentElement.classList.contains('light')).toBe(true);

    module.useThemeStore.getState().setTheme('system');
    controller.trigger(true);

    expect(document.documentElement.classList.contains('dark')).toBe(true);
    expect(document.documentElement.style.colorScheme).toBe('dark');

    controller.trigger(false);

    expect(document.documentElement.classList.contains('light')).toBe(true);
    expect(document.documentElement.style.colorScheme).toBe('light');
  });
});
